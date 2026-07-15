package system

import (
	"context"
	"os"
	"sync"
	"time"

	"notes-of-ashen/internal/authutil"
	medialogic "notes-of-ashen/internal/logic/media"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

const healthCacheTTL = 30 * time.Second

var healthCache struct {
	sync.Mutex
	at     time.Time
	report *types.SystemHealthResp
}

func Health(ctx context.Context, svcCtx *svc.ServiceContext, force bool) (*types.SystemHealthResp, error) {
	if err := authutil.RequireAdmin(ctx); err != nil {
		return nil, err
	}
	if !force {
		healthCache.Lock()
		if healthCache.report != nil && time.Since(healthCache.at) < healthCacheTTL {
			copyReport := *healthCache.report
			copyReport.Checks = append([]types.DependencyCheckResp(nil), healthCache.report.Checks...)
			healthCache.Unlock()
			return &copyReport, nil
		}
		healthCache.Unlock()
	}

	type result struct {
		index int
		check types.DependencyCheckResp
	}
	results := make(chan result, 6)
	run := func(index int, name string, enabled bool, probe func(context.Context) error) {
		go func() {
			if !enabled {
				results <- result{index, types.DependencyCheckResp{Name: name, Status: "disabled", Message: "未启用"}}
				return
			}
			started := time.Now()
			probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			check := types.DependencyCheckResp{Name: name, Status: "up"}
			if err := probe(probeCtx); err != nil {
				check.Status = "down"
				check.Message = "探测失败"
			} else {
				check.Message = "正常"
			}
			check.LatencyMs = time.Since(started).Milliseconds()
			results <- result{index, check}
		}()
	}
	run(0, "mysql", true, func(ctx context.Context) error { return svcCtx.Store.DB().PingContext(ctx) })
	run(1, "redis", true, func(ctx context.Context) error { return svcCtx.Redis.Ping(ctx).Err() })
	run(2, "meilisearch", svcCtx.Config.Search.Enabled, func(ctx context.Context) error { return svcCtx.Search.Health(ctx) })
	run(3, "rabbitmq", svcCtx.Config.RabbitMQ.Enabled, func(ctx context.Context) error { return svcCtx.Events.Health(ctx) })
	run(4, "smtp", svcCtx.Config.Email.Enabled, func(ctx context.Context) error { return svcCtx.Mailer.Health(ctx) })
	run(5, "media", true, func(context.Context) error {
		root, err := medialogic.Root(svcCtx)
		if err != nil {
			return err
		}
		file, err := os.CreateTemp(root, ".health-*")
		if err != nil {
			return err
		}
		name := file.Name()
		defer os.Remove(name)
		if err := file.Close(); err != nil {
			return err
		}
		return os.Remove(name)
	})

	checks := make([]types.DependencyCheckResp, 6)
	status := "healthy"
	for range checks {
		item := <-results
		checks[item.index] = item.check
		if item.check.Status == "down" {
			status = "degraded"
		}
	}
	report := &types.SystemHealthResp{Status: status, CheckedAt: time.Now().UTC(), Checks: checks}
	healthCache.Lock()
	healthCache.at = time.Now()
	healthCache.report = report
	healthCache.Unlock()
	return report, nil
}
