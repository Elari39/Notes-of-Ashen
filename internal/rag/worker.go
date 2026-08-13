package rag

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/ragclient"
	"notes-of-ashen/model"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	workerPollInterval   = 2 * time.Second
	workerRequestTimeout = 8 * time.Minute
	workerBatchSize      = 2
	// 任务按批次顺序执行；租约必须覆盖整批最坏请求时长，额外保留一个
	// 请求窗口给完成/失败状态回写，避免第二条任务尚未开始就被另一实例接手。
	workerLeaseDuration = workerRequestTimeout * time.Duration(workerBatchSize+1)
	// 全量重建可能超过单次上游请求窗口，因此使用独立 lease 并在运行期间续租。
	// 租约至少覆盖单个请求超时，续租周期保留足够空间处理一次瞬时数据库延迟。
	workerRebuildLeaseDuration      = workerRequestTimeout * 2
	workerRebuildLeaseRenewInterval = workerRebuildLeaseDuration / 3
	workerRebuildLeaseRenewTimeout  = 30 * time.Second
	// 重建失败通常是 Qdrant 或模型服务暂不可用。使用状态更新时间作为跨实例
	// 共享的最小退避信号，避免每个轮询周期都重新创建 collection 或打满上游。
	workerRebuildRetryDelay = time.Minute
)

var (
	ErrRebuildInProgress   = errors.New("rag rebuild is already in progress")
	ErrIndexEpochExhausted = errors.New("rag index epoch is exhausted")
	// ErrRAGIndexChanged instructs the outbox loop to release the lease and retry
	// after another instance has taken ownership of a newer index epoch.
	ErrRAGIndexChanged = errors.New("rag index state changed")
)

// Worker 以 MySQL outbox 为唯一任务来源，任意 API 实例都能通过 lease 安全接手。
// Qdrant 始终是可再生派生索引，失败只会延迟检索可用性而不会破坏文章数据。
type Worker struct {
	store      *model.Store
	ragConfig  config.RAGConf
	authSecret string
	qdrant     *ragclient.QdrantClient
	// lifecycleMu 保护运行期 context 与重建标记；workMu 让已领取的普通任务
	// 与 collection 重建串行，避免旧 epoch 的写入和 DeleteCollection 交错。
	lifecycleMu sync.Mutex
	workMu      sync.Mutex
	runCtx      context.Context
	rebuilding  bool
	// pendingRebuild 表示本轮重建过程中又收到 embedding 配置变更。由
	// lifecycleMu 保护；当前轮释放 workMu 后会自动以数据库中的最新配置接续。
	pendingRebuild bool
	cancel         context.CancelFunc
	done           chan struct{}
	once           sync.Once
}

// rebuildLeaseGuard 将 MySQL 中的 fencing lease 映射为当前重建的 context。
// 一旦续租失败或持有者被接管，立即取消所有后续网络调用；已有在途请求仍可能
// 返回，但其 payload 带旧 epoch，且后续状态写入受 token 条件保护。
type rebuildLeaseGuard struct {
	ctx        context.Context
	cancel     context.CancelFunc
	stop       chan struct{}
	done       chan struct{}
	stopOnce   sync.Once
	lossMu     sync.RWMutex
	lossReason error
}

func (g *rebuildLeaseGuard) lost(err error) {
	if g == nil {
		return
	}
	g.lossMu.Lock()
	if g.lossReason == nil {
		g.lossReason = err
	}
	g.lossMu.Unlock()
	g.cancel()
}

func (g *rebuildLeaseGuard) err() error {
	if g == nil {
		return nil
	}
	g.lossMu.RLock()
	err := g.lossReason
	g.lossMu.RUnlock()
	return err
}

func (g *rebuildLeaseGuard) close() {
	if g == nil {
		return
	}
	g.stopOnce.Do(func() {
		close(g.stop)
		g.cancel()
		<-g.done
	})
}

// newRebuildLeaseGuard 在持有重建 fencing token 的整个期间续租。续租失败时
// 不能再信任当前实例仍有权写入 collection，因此立刻取消派生 context；调用方
// 随后只会通过带 token 条件的状态更新尝试收尾。
func newRebuildLeaseGuard(parent context.Context, store *model.Store, state model.RAGIndexState) *rebuildLeaseGuard {
	ctx, cancel := context.WithCancel(parent)
	guard := &rebuildLeaseGuard{
		ctx:    ctx,
		cancel: cancel,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go func() {
		defer close(guard.done)
		ticker := time.NewTicker(workerRebuildLeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-guard.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if err := guard.renew(store, state); err != nil {
				return
			}
		}
	}()
	return guard
}

// renew 执行一次租约续期。数据库不可用时也必须 fail closed：继续向 Qdrant
// 写入会让实例在无法确认租约是否被接管的情况下污染派生索引。
func (g *rebuildLeaseGuard) renew(store *model.Store, state model.RAGIndexState) error {
	if err := g.check(); err != nil {
		return err
	}
	renewCtx, renewCancel := context.WithTimeout(g.ctx, workerRebuildLeaseRenewTimeout)
	renewed, err := store.RenewRAGIndexRebuildLease(renewCtx, state.Epoch, state.EmbeddingFingerprint, state.RebuildLeaseToken, workerRebuildLeaseDuration)
	renewCancel()
	if err != nil || !renewed {
		g.lost(ErrRAGIndexChanged)
		return ErrRAGIndexChanged
	}
	return nil
}

func (g *rebuildLeaseGuard) check() error {
	if g == nil {
		return errors.New("rag rebuild lease guard is unavailable")
	}
	if err := g.err(); err != nil {
		return err
	}
	if err := g.ctx.Err(); err != nil {
		if leaseErr := g.err(); leaseErr != nil {
			return leaseErr
		}
		return err
	}
	return nil
}

func rebuildLeaseExpired(state model.RAGIndexState, now time.Time) bool {
	return state.RebuildLeaseExpiresAt == nil || !state.RebuildLeaseExpiresAt.After(now.UTC())
}

func NewWorker(ragConfig config.RAGConf, authSecret string, store *model.Store) (*Worker, error) {
	if store == nil || !ragConfig.Enabled {
		return nil, nil
	}
	qdrant, err := ragclient.NewQdrantClient(ragConfig.QdrantURL, ragConfig.QdrantAPIKey, ragConfig.QdrantCollection)
	if err != nil {
		return nil, err
	}
	return &Worker{store: store, ragConfig: ragConfig, authSecret: authSecret, qdrant: qdrant, done: make(chan struct{})}, nil
}

func (w *Worker) Start() {
	if w == nil {
		return
	}
	w.lifecycleMu.Lock()
	if w.cancel != nil {
		w.lifecycleMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.runCtx = ctx
	w.cancel = cancel
	w.lifecycleMu.Unlock()
	go func() {
		defer close(w.done)
		w.run(ctx)
	}()
}

func (w *Worker) Close() {
	if w == nil {
		return
	}
	w.once.Do(func() {
		w.lifecycleMu.Lock()
		cancel := w.cancel
		w.lifecycleMu.Unlock()
		if cancel != nil {
			cancel()
			<-w.done
		}
	})
}

func (w *Worker) Qdrant() *ragclient.QdrantClient {
	if w == nil {
		return nil
	}
	return w.qdrant
}

func (w *Worker) run(ctx context.Context) {
	// 清理过期私有会话是低频维护任务，失败不影响索引队列。
	w.cleanupSessions(ctx)
	w.process(ctx)
	pollTicker := time.NewTicker(workerPollInterval)
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer pollTicker.Stop()
	defer cleanupTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			w.process(ctx)
		case <-cleanupTicker.C:
			w.cleanupSessions(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	if w.isRebuilding() {
		return
	}
	// beginRebuild 在 worker 取得 workMu 前先设置 rebuilding；反过来，已经
	// 进入这里的任务会在重建清空 collection 前结束，避免写入被错误保留。
	w.workMu.Lock()
	defer w.workMu.Unlock()
	if w.isRebuilding() {
		return
	}
	settings, err := w.store.RAGSettings(ctx)
	if err != nil || !w.ragConfig.Enabled || !settings.Enabled {
		return
	}
	state, err := w.store.RAGIndexState(ctx)
	if err != nil {
		return
	}
	// 正常启动、配置变更后的 pending、异常实例留下的过期 rebuilding，以及
	// 上游短暂故障留下的 error 都必须由任意存活 Worker 自动接管。
	// BeginRAGIndexRebuild 在数据库中做
	// 最终排他判断，因此多个实例同时发现过期状态也是安全的。
	now := time.Now().UTC()
	shouldRebuild := state.Status == model.RAGIndexStatusNeedsRebuild ||
		(state.Status == model.RAGIndexStatusRebuilding && rebuildLeaseExpired(*state, now)) ||
		(state.Status == model.RAGIndexStatusError && !state.UpdatedAt.After(now.Add(-workerRebuildRetryDelay)))
	if shouldRebuild {
		if _, rebuildErr := w.StartRebuild(*settings); rebuildErr != nil {
			logx.Errorf("rag automatic rebuild start failed: %v", safeErrorSummary(rebuildErr))
		}
		return
	}
	if state.Status != model.RAGIndexStatusReady {
		return
	}
	leaseToken := newLeaseToken()
	jobs, err := w.store.ClaimRAGSyncJobs(ctx, workerBatchSize, leaseToken, workerLeaseDuration)
	if err != nil {
		logx.Errorf("rag claim sync jobs failed: %v", err)
		return
	}
	completedAny := false
	for _, job := range jobs {
		jobCtx, cancel := context.WithTimeout(ctx, workerRequestTimeout)
		err := w.processJob(jobCtx, *settings, *state, job)
		cancel()
		if err == nil {
			if completeErr := w.store.CompleteRAGSyncJob(ctx, job.ID, leaseToken); completeErr != nil {
				logx.Errorf("rag complete sync job failed: id=%d err=%v", job.ID, completeErr)
			} else {
				completedAny = true
			}
			continue
		}
		if errors.Is(err, ErrRAGIndexChanged) {
			// 另一实例已切换到新 epoch。立即释放租约，避免旧任务等到
			// 长租约到期；新 collection ready 后由任一 Worker 重试。
			if releaseErr := w.store.FailRAGSyncJob(ctx, job.ID, leaseToken, time.Now().UTC(), "rag index state changed; retry pending"); releaseErr != nil {
				logx.Errorf("rag release stale sync job failed: id=%d err=%v", job.ID, releaseErr)
			}
			continue
		}
		// 错误摘要仅保存类型与简短文本，不能记录文章正文、问题或 API Key。
		backoff := retryDelay(job.Attempts)
		if failErr := w.store.FailRAGSyncJob(ctx, job.ID, leaseToken, time.Now().UTC().Add(backoff), safeErrorSummary(err)); failErr != nil {
			logx.Errorf("rag fail sync job update failed: id=%d err=%v", job.ID, failErr)
		}
		logx.Errorf("rag sync job failed: id=%d article=%d operation=%s err=%v", job.ID, job.ArticleID, job.Operation, safeErrorSummary(err))
	}
	if completedAny {
		if err := w.refreshIndexStats(ctx, *state); err != nil {
			// 统计刷新失败不会让已成功写入的 outbox 重新执行；后续批次会再次尝试。
			logx.Errorf("rag refresh index statistics failed: %v", err)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, settings model.RAGSettings, state model.RAGIndexState, job model.RAGSyncJob) error {
	if err := w.assertIndexState(ctx, state); err != nil {
		return err
	}
	if job.Operation == model.RAGSyncOperationDelete {
		if err := w.qdrant.DeleteArticleAtEpoch(ctx, job.ArticleID, state.Epoch); err != nil {
			return err
		}
		if err := w.assertIndexState(ctx, state); err != nil {
			return err
		}
		return w.store.HideRAGChatMessagesForArticle(ctx, job.ArticleID)
	}
	if job.Operation != model.RAGSyncOperationUpsert {
		return fmt.Errorf("invalid rag sync operation: %s", job.Operation)
	}
	article, err := w.store.FindArticle(ctx, job.ArticleID)
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return err
		}
		if err := w.qdrant.DeleteArticleAtEpoch(ctx, job.ArticleID, state.Epoch); err != nil {
			return err
		}
		if err := w.assertIndexState(ctx, state); err != nil {
			return err
		}
		return w.store.HideRAGChatMessagesForArticle(ctx, job.ArticleID)
	}
	if !model.IsArticlePubliclyVisible(*article, time.Now()) {
		if err := w.qdrant.DeleteArticleAtEpoch(ctx, job.ArticleID, state.Epoch); err != nil {
			return err
		}
		if err := w.assertIndexState(ctx, state); err != nil {
			return err
		}
		return w.store.HideRAGChatMessagesForArticle(ctx, job.ArticleID)
	}
	conf, err := EffectiveProviderConfig(settings, w.authSecret)
	if err != nil {
		return err
	}
	chunks := SplitMarkdown(article.Content)
	if len(chunks) == 0 {
		return w.qdrant.DeleteArticleAtEpoch(ctx, job.ArticleID, state.Epoch)
	}
	// 先删旧分段，失败任务会完整重试；payload epoch 隔离重建期的数据。
	if err := w.qdrant.DeleteArticleAtEpoch(ctx, job.ArticleID, state.Epoch); err != nil {
		return err
	}
	if err := w.assertIndexState(ctx, state); err != nil {
		return err
	}
	vectors, err := ragclient.Embeddings(ctx, conf, chunks)
	if err != nil {
		return err
	}
	sourceHash := SourceHash(article.Title, article.Summary, article.Content)
	fingerprint := EmbeddingFingerprint(settings)
	points := make([]ragclient.Point, 0, len(chunks))
	for index, chunk := range chunks {
		points = append(points, ragclient.Point{
			ID: pointID(state.Epoch, article.ID, index, sourceHash), Vector: vectors[index],
			Payload: ragclient.PointPayload{ArticleID: article.ID, ChunkIndex: index, SourceHash: sourceHash, Content: chunk, EmbeddingFingerprint: fingerprint, Epoch: state.Epoch},
		})
	}
	if err := w.assertIndexState(ctx, state); err != nil {
		return err
	}
	if err := w.qdrant.Upsert(ctx, points); err != nil {
		return err
	}
	return w.assertIndexState(ctx, state)
}

func (w *Worker) assertIndexState(ctx context.Context, state model.RAGIndexState) error {
	if w == nil || w.store == nil {
		return errors.New("rag worker is unavailable")
	}
	current, err := w.store.IsRAGIndexStateCurrent(ctx, state.Status, state.Epoch, state.EmbeddingFingerprint, state.RebuildLeaseToken)
	if err != nil {
		return err
	}
	if !current {
		return ErrRAGIndexChanged
	}
	return nil
}

// Rebuild 重建集合并为当前已公开文章重新写入 outbox；问答端在 status != ready 时
// fail closed。由于任务重新读取文章，重建开始后任何并发写入都不会丢失。
func (w *Worker) Rebuild(ctx context.Context, settings model.RAGSettings) (*model.RAGIndexState, error) {
	if !w.beginRebuild() {
		return nil, ErrRebuildInProgress
	}
	defer w.finishRebuild()
	return w.rebuild(ctx, settings)
}

// StartRebuild 在 Worker 生命周期内异步执行一次重建。返回 started=false、err=nil
// 表示已经有手动或自动重建在运行，调用方无需将其视为失败。
func (w *Worker) StartRebuild(settings model.RAGSettings) (started bool, err error) {
	if w == nil || w.qdrant == nil || !w.ragConfig.Enabled {
		return false, errors.New("rag engine is unavailable")
	}
	if !w.beginOrQueueRebuild() {
		return false, nil
	}
	w.lifecycleMu.Lock()
	ctx := w.runCtx
	w.lifecycleMu.Unlock()
	if ctx == nil {
		w.finishRebuild()
		return false, errors.New("rag worker is not started")
	}
	if err := ctx.Err(); err != nil {
		w.finishRebuild()
		return false, errors.New("rag worker is stopped")
	}
	go func() {
		defer w.finishRebuild()
		if _, rebuildErr := w.rebuild(ctx, settings); rebuildErr != nil && !errors.Is(rebuildErr, context.Canceled) {
			logx.Errorf("rag asynchronous rebuild failed: %v", safeErrorSummary(rebuildErr))
		}
	}()
	return true, nil
}

func (w *Worker) rebuild(ctx context.Context, settings model.RAGSettings) (*model.RAGIndexState, error) {
	if w == nil || w.qdrant == nil || !w.ragConfig.Enabled {
		return nil, errors.New("rag engine is unavailable")
	}
	w.workMu.Lock()
	defer w.workMu.Unlock()
	if _, err := EffectiveProviderConfig(settings, w.authSecret); err != nil {
		return nil, err
	}
	fingerprint := EmbeddingFingerprint(settings)
	statePtr, claimed, err := w.store.BeginRAGIndexRebuild(ctx, fingerprint, newLeaseToken(), workerRebuildLeaseDuration)
	if err != nil {
		if errors.Is(err, model.ErrRAGIndexEpochExhausted) {
			return nil, ErrIndexEpochExhausted
		}
		return nil, err
	}
	if !claimed || statePtr == nil {
		return nil, ErrRebuildInProgress
	}
	state := *statePtr
	guard := newRebuildLeaseGuard(ctx, w.store, state)
	defer guard.close()
	workCtx := guard.ctx
	if err := w.assertRebuildLease(workCtx, state, guard); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	if err := w.qdrant.DeleteCollection(workCtx); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	if err := w.assertRebuildLease(workCtx, state, guard); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	if err := w.qdrant.RecreateCollection(workCtx, settings.EmbeddingDimensions); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	if err := w.assertRebuildLease(workCtx, state, guard); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	articles, err := w.store.ListRAGPublicArticles(workCtx)
	if err != nil {
		return w.failRebuild(ctx, state, err)
	}
	// 同步重建可明确完成时刻与计数，避免仅靠常驻 worker 将状态停留在 rebuilding。
	// rebuild 期间文章写入仍会创建新的 outbox，完成后这些任务交由常驻 worker
	// 处理；这里不预先写入全量任务，从而不覆盖并发文章写入的最新任务。
	for _, article := range articles {
		job := model.RAGSyncJob{ArticleID: article.ID, Operation: model.RAGSyncOperationUpsert}
		if err := w.assertRebuildLease(workCtx, state, guard); err != nil {
			return w.failRebuild(ctx, state, err)
		}
		jobCtx, cancel := context.WithTimeout(workCtx, workerRequestTimeout)
		err := w.processJob(jobCtx, settings, state, job)
		cancel()
		if err != nil {
			return w.failRebuild(ctx, state, err)
		}
	}
	// 设置可能在长时间重建期间更新。绝不能将旧 embedding 的 collection
	// 标记为新设置的 ready 状态；保留 needs_rebuild 供后续自动/手动重建接手。
	if err := w.assertRebuildLease(workCtx, state, guard); err != nil {
		return w.failRebuild(ctx, state, err)
	}
	latestSettings, settingsErr := w.store.RAGSettings(workCtx)
	if settingsErr != nil {
		return w.failRebuild(ctx, state, settingsErr)
	}
	if !latestSettings.Enabled || EmbeddingFingerprint(*latestSettings) != state.EmbeddingFingerprint {
		// 停止续租后再回写最终状态，避免 ticker 与 Release 并发而将其误判为
		// 租约丢失。回写仍以 token + 未过期条件保护。
		guard.close()
		released, err := w.store.ReleaseRAGIndexRebuildToPending(ctx, state.Epoch, state.RebuildLeaseToken)
		if err != nil {
			return nil, err
		}
		if !released {
			w.queueRebuildAfterCurrent()
			return nil, ErrRAGIndexChanged
		}
		if latestSettings.Enabled {
			w.queueRebuildAfterCurrent()
		}
		state.Status = model.RAGIndexStatusNeedsRebuild
		state.EmbeddingFingerprint = EmbeddingFingerprint(*latestSettings)
		state.RebuildLeaseToken, state.RebuildLeaseExpiresAt = "", nil
		return &state, nil
	}
	completed := time.Now().UTC()
	articleCount, chunkCount := indexStats(articles)
	guard.close()
	completedOK, err := w.store.CompleteRAGIndexRebuild(ctx, state.Epoch, state.EmbeddingFingerprint, state.RebuildLeaseToken, articleCount, chunkCount)
	if err != nil {
		return nil, err
	}
	if !completedOK {
		w.queueRebuildAfterCurrent()
		return nil, ErrRAGIndexChanged
	}
	state.Status = model.RAGIndexStatusReady
	state.IndexedArticleCount, state.IndexedChunkCount = articleCount, chunkCount
	state.CompletedAt = &completed
	state.RebuildLeaseToken, state.RebuildLeaseExpiresAt = "", nil
	return &state, nil
}

func (w *Worker) assertRebuildLease(ctx context.Context, state model.RAGIndexState, guard *rebuildLeaseGuard) error {
	if err := guard.check(); err != nil {
		return err
	}
	return w.assertIndexState(ctx, state)
}

func (w *Worker) beginRebuild() bool {
	if w == nil {
		return false
	}
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.rebuilding {
		return false
	}
	w.rebuilding = true
	return true
}

// beginOrQueueRebuild 将“发现已有重建”和“登记后续重建”放在同一把锁内，
// 避免旧轮刚结束而新请求才写入 pending 标记，造成待重建状态无人接手。
func (w *Worker) beginOrQueueRebuild() bool {
	if w == nil {
		return false
	}
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.rebuilding {
		w.pendingRebuild = true
		return false
	}
	w.rebuilding = true
	return true
}

func (w *Worker) queueRebuildAfterCurrent() {
	if w == nil {
		return
	}
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	if w.rebuilding {
		w.pendingRebuild = true
	}
}

func (w *Worker) finishRebuild() {
	if w == nil {
		return
	}
	w.lifecycleMu.Lock()
	pending := w.pendingRebuild
	w.rebuilding = false
	w.pendingRebuild = false
	w.lifecycleMu.Unlock()
	if pending {
		w.startQueuedRebuild()
	}
}

// startQueuedRebuild 只在上一轮彻底结束后读取数据库中的最新配置。这样连续修改
// embedding 地址、模型或维度时，不会把某一次旧请求捕获的配置再用于重建。
func (w *Worker) startQueuedRebuild() {
	if w == nil || w.store == nil {
		return
	}
	go func() {
		settings, err := w.store.RAGSettings(context.Background())
		if err != nil {
			logx.Errorf("read settings for queued rag rebuild failed: %v", safeErrorSummary(err))
			return
		}
		if !settings.Enabled {
			return
		}
		if _, err := w.StartRebuild(*settings); err != nil {
			logx.Errorf("start queued rag rebuild failed: %v", safeErrorSummary(err))
		}
	}()
}

func (w *Worker) isRebuilding() bool {
	if w == nil {
		return false
	}
	w.lifecycleMu.Lock()
	defer w.lifecycleMu.Unlock()
	return w.rebuilding
}

func (w *Worker) refreshIndexStats(ctx context.Context, state model.RAGIndexState) error {
	articles, err := w.store.ListRAGPublicArticles(ctx)
	if err != nil {
		return err
	}
	articleCount, chunkCount := indexStats(articles)
	return w.store.UpdateRAGIndexStats(ctx, state.Epoch, articleCount, chunkCount)
}

func indexStats(articles []model.Article) (articleCount, chunkCount uint64) {
	for _, article := range articles {
		chunks := SplitMarkdown(article.Content)
		if len(chunks) == 0 {
			continue
		}
		articleCount++
		chunkCount += uint64(len(chunks))
	}
	return articleCount, chunkCount
}

// nextRAGEpoch 为每次 collection 重建分配一个从不回绕的 epoch。epoch 同时
// 出现在 Qdrant payload 和稳定 point ID 中；一旦回绕，极端情况下旧向量可能
// 被新索引误认为属于同一命名空间，因此宁可明确停止重建等待人工处理。
func nextRAGEpoch(current *model.RAGIndexState) (uint64, error) {
	if current == nil {
		return 1, nil
	}
	if current.Epoch == ^uint64(0) {
		return 0, ErrIndexEpochExhausted
	}
	return current.Epoch + 1, nil
}

func (w *Worker) failRebuild(ctx context.Context, state model.RAGIndexState, cause error) (*model.RAGIndexState, error) {
	updated, err := w.store.FailRAGIndexRebuild(ctx, state.Epoch, state.EmbeddingFingerprint, state.RebuildLeaseToken, safeErrorSummary(cause))
	if err != nil {
		return nil, err
	}
	if !updated {
		// 配置已变更或另一实例接管了 epoch；旧错误不能覆盖新目标。
		_, _ = w.store.ReleaseRAGIndexRebuildToPending(ctx, state.Epoch, state.RebuildLeaseToken)
		w.queueRebuildAfterCurrent()
		return nil, ErrRAGIndexChanged
	}
	state.Status = model.RAGIndexStatusError
	state.LastError = safeErrorSummary(cause)
	return nil, cause
}

func (w *Worker) cleanupSessions(ctx context.Context) {
	if _, err := w.store.CleanupExpiredRAGChatSessions(ctx, time.Now().UTC()); err != nil {
		logx.Errorf("rag chat session cleanup failed: %v", err)
	}
}

func retryDelay(attempts uint) time.Duration {
	if attempts > 8 {
		attempts = 8
	}
	return time.Minute * time.Duration(1<<attempts)
}

func pointID(epoch, articleID uint64, chunkIndex int, sourceHash string) string {
	// Qdrant 的字符串 point ID 必须是 UUID。基于稳定输入派生 UUIDv5 形式的
	// 值，既不引入额外依赖，也让同一文章版本的重试幂等覆盖相同 point。
	value := fmt.Sprintf("%d|%d|%d|%s", epoch, articleID, chunkIndex, sourceHash)
	// SHA-256 的前 16 字节足以生成稳定 UUID；按 RFC 4122 版本/变体位重写。
	sum := sha256.Sum256([]byte(value))
	id := sum[:16]
	id[6] = (id[6] & 0x0f) | 0x50
	id[8] = (id[8] & 0x3f) | 0x80
	return hex.EncodeToString(id[0:4]) + "-" + hex.EncodeToString(id[4:6]) + "-" + hex.EncodeToString(id[6:8]) + "-" + hex.EncodeToString(id[8:10]) + "-" + hex.EncodeToString(id[10:16])
}

func EmbeddingFingerprint(settings model.RAGSettings) string {
	value := strings.Join([]string{strings.TrimRight(strings.TrimSpace(settings.EmbeddingBaseURL), "/"), strings.TrimSpace(settings.EmbeddingModel), fmt.Sprintf("%d", settings.EmbeddingDimensions)}, "|")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func newLeaseToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func safeErrorSummary(err error) string {
	return ragclient.SafeErrorSummary(err)
}
