package rag

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"notes-of-ashen/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPointIDIsStableUUIDv5(t *testing.T) {
	first := pointID(3, 42, 1, "abc")
	if again := pointID(3, 42, 1, "abc"); again != first {
		t.Fatalf("pointID() is not stable: first=%q again=%q", first, again)
	}
	if other := pointID(3, 42, 2, "abc"); other == first {
		t.Fatalf("pointID() = %q for different chunk, want distinct id", other)
	}
	if ok, err := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, first); err != nil || !ok {
		t.Fatalf("pointID() = %q, want UUIDv5 format (match=%v err=%v)", first, ok, err)
	}
}

func TestIndexStatsSkipsArticlesWithoutIndexableContent(t *testing.T) {
	articleCount, chunkCount := indexStats([]model.Article{
		{ID: 1, Content: ""},
		{ID: 2, Content: "# 标题\n\n可检索正文"},
	})
	if articleCount != 1 || chunkCount != 1 {
		t.Fatalf("indexStats() = (%d, %d), want (1, 1)", articleCount, chunkCount)
	}
}

func TestWorkerLeaseCoversSequentialClaimedBatch(t *testing.T) {
	minimum := workerRequestTimeout * time.Duration(workerBatchSize)
	if workerLeaseDuration <= minimum {
		t.Fatalf("workerLeaseDuration = %s, want > sequential batch timeout %s", workerLeaseDuration, minimum)
	}
}

func TestRebuildLeaseExpired(t *testing.T) {
	now := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	before := now.Add(-time.Nanosecond)
	at := now
	after := now.Add(time.Nanosecond)

	for _, tt := range []struct {
		name  string
		state model.RAGIndexState
		want  bool
	}{
		{name: "missing lease is recoverable", want: true},
		{name: "past lease is expired", state: model.RAGIndexState{RebuildLeaseExpiresAt: &before}, want: true},
		{name: "lease expiring now is expired", state: model.RAGIndexState{RebuildLeaseExpiresAt: &at}, want: true},
		{name: "future lease remains owned", state: model.RAGIndexState{RebuildLeaseExpiresAt: &after}, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := rebuildLeaseExpired(tt.state, now); got != tt.want {
				t.Fatalf("rebuildLeaseExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRebuildLeaseGuardLosingLeaseCancelsWork(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()
	guard := newRebuildLeaseGuard(parent, nil, model.RAGIndexState{})
	t.Cleanup(guard.close)

	guard.lost(ErrRAGIndexChanged)
	select {
	case <-guard.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("lease loss did not cancel rebuild context")
	}
	if !errors.Is(guard.err(), ErrRAGIndexChanged) {
		t.Fatalf("guard.err() = %v, want %v", guard.err(), ErrRAGIndexChanged)
	}
	if !errors.Is(guard.check(), ErrRAGIndexChanged) {
		t.Fatalf("guard.check() = %v, want %v", guard.check(), ErrRAGIndexChanged)
	}

	// 首次丢失原因用于驱动收尾路径，后续错误不能把它覆盖。
	guard.lost(errors.New("later renewal error"))
	if !errors.Is(guard.err(), ErrRAGIndexChanged) {
		t.Fatalf("guard.err() after second loss = %v, want original %v", guard.err(), ErrRAGIndexChanged)
	}
}

func TestRebuildLeaseGuardPropagatesParentCancellation(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	guard := newRebuildLeaseGuard(parent, nil, model.RAGIndexState{})

	parentCancel()
	select {
	case <-guard.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("parent cancellation did not cancel rebuild context")
	}
	if !errors.Is(guard.check(), context.Canceled) {
		t.Fatalf("guard.check() = %v, want context.Canceled", guard.check())
	}

	// close 必须等待续租 goroutine 退出，避免重建结束后留下后台访问。
	guard.close()
}

func TestRebuildLeaseGuardRenewFailureCancelsWork(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	defer db.Close()

	state := model.RAGIndexState{
		Status:               model.RAGIndexStatusRebuilding,
		Epoch:                9,
		EmbeddingFingerprint: "fingerprint",
		RebuildLeaseToken:    "owner-token",
	}
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE rag_index_state SET rebuild_lease_expires_at = ?
WHERE id = 1 AND status = ? AND epoch = ? AND embedding_fingerprint = ?
  AND rebuild_lease_token = ? AND rebuild_lease_expires_at > NOW(6)`)).
		WithArgs(sqlmock.AnyArg(), model.RAGIndexStatusRebuilding, state.Epoch, state.EmbeddingFingerprint, state.RebuildLeaseToken).
		WillReturnResult(sqlmock.NewResult(0, 0))

	guard := newRebuildLeaseGuard(context.Background(), model.NewStore(db), state)
	t.Cleanup(guard.close)
	if err := guard.renew(model.NewStore(db), state); !errors.Is(err, ErrRAGIndexChanged) {
		t.Fatalf("guard.renew() error = %v, want %v", err, ErrRAGIndexChanged)
	}
	select {
	case <-guard.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("renewal ownership loss did not cancel rebuild context")
	}
	if !errors.Is(guard.check(), ErrRAGIndexChanged) {
		t.Fatalf("guard.check() = %v, want %v", guard.check(), ErrRAGIndexChanged)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNextRAGEpochNeverWraps(t *testing.T) {
	for _, tt := range []struct {
		name    string
		current *model.RAGIndexState
		want    uint64
		wantErr error
	}{
		{name: "missing state starts at one", want: 1},
		{name: "increments existing state", current: &model.RAGIndexState{Epoch: 41}, want: 42},
		{name: "max epoch is rejected", current: &model.RAGIndexState{Epoch: ^uint64(0)}, wantErr: ErrIndexEpochExhausted},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextRAGEpoch(tt.current)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("nextRAGEpoch() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("nextRAGEpoch() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRebuildGuardIsMutuallyExclusive(t *testing.T) {
	worker := &Worker{}
	if !worker.beginRebuild() {
		t.Fatal("first beginRebuild() = false, want true")
	}
	if worker.beginRebuild() {
		t.Fatal("second beginRebuild() = true, want false")
	}
	worker.finishRebuild()
	if !worker.beginRebuild() {
		t.Fatal("beginRebuild() after finish = false, want true")
	}
}

func TestBeginOrQueueRebuildKeepsOneFollowUpRequest(t *testing.T) {
	worker := &Worker{}
	if !worker.beginRebuild() {
		t.Fatal("initial beginRebuild() = false, want true")
	}
	if worker.beginOrQueueRebuild() {
		t.Fatal("beginOrQueueRebuild() = true while rebuilding, want false")
	}
	worker.lifecycleMu.Lock()
	pending := worker.pendingRebuild
	worker.lifecycleMu.Unlock()
	if !pending {
		t.Fatal("beginOrQueueRebuild() did not retain follow-up rebuild request")
	}
	// 空 Worker 没有 Store，finishRebuild 不会启动 goroutine；这里验证该标记
	// 会被消费，且下一轮可正常开始。
	worker.finishRebuild()
	if !worker.beginRebuild() {
		t.Fatal("beginRebuild() after queued finish = false, want true")
	}
}

func TestSafeErrorSummaryNeverPersistsArbitraryProviderText(t *testing.T) {
	if got := safeErrorSummary(errors.New("question=private article content")); got != "rag upstream request failed" {
		t.Fatalf("safeErrorSummary() = %q", got)
	}
	if got := safeErrorSummary(context.DeadlineExceeded); got != "upstream request timed out" {
		t.Fatalf("safeErrorSummary(deadline) = %q", got)
	}
}
