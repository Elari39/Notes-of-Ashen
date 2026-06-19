package site

import (
	"context"
	"net/http"
	"time"

	"notes-of-ashen/internal/svc"
)

// HealthStatus 描述单个依赖的存活状态。
type HealthStatus struct {
	Status string `json:"status"` // "up" | "down"
	Error  string `json:"error,omitempty"`
}

// HealthReport 是 /healthz 的响应体。
type HealthReport struct {
	Status string                  `json:"status"` // 整体：所有必需依赖 up 时为 "ok"，否则 "degraded"
	Checks map[string]HealthStatus `json:"checks"`
}

const healthCheckTimeout = 2 * time.Second

// Health 探测 DB、Redis 等关键依赖的存活状态。
// MQ 与 Search 仅报告其启用状态，不做真实探测（无轻量 ping 接口，避免增加复杂度）。
// 任一必需依赖 down 时整体为 "degraded"，但接口本身始终返回 200，便于探针读取明细；
// 调用方可根据 checks 判定。
func Health(ctx context.Context, svcCtx *svc.ServiceContext) *HealthReport {
	checks := make(map[string]HealthStatus)
	allUp := true

	checks["db"] = pingDB(ctx, svcCtx)
	if checks["db"].Status != "up" {
		allUp = false
	}

	checks["redis"] = pingRedis(ctx, svcCtx)
	if checks["redis"].Status != "up" {
		allUp = false
	}

	overall := "ok"
	if !allUp {
		overall = "degraded"
	}
	return &HealthReport{Status: overall, Checks: checks}
}

func pingDB(ctx context.Context, svcCtx *svc.ServiceContext) HealthStatus {
	if svcCtx.Store == nil || svcCtx.Store.DB() == nil {
		return HealthStatus{Status: "down", Error: "db not initialized"}
	}
	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	if err := svcCtx.Store.DB().PingContext(checkCtx); err != nil {
		return HealthStatus{Status: "down", Error: err.Error()}
	}
	return HealthStatus{Status: "up"}
}

func pingRedis(ctx context.Context, svcCtx *svc.ServiceContext) HealthStatus {
	if svcCtx.Redis == nil {
		return HealthStatus{Status: "down", Error: "redis not initialized"}
	}
	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	if err := svcCtx.Redis.Ping(checkCtx).Err(); err != nil {
		return HealthStatus{Status: "down", Error: err.Error()}
	}
	return HealthStatus{Status: "up"}
}

// HTTPStatus 根据 report 决定 /healthz 的 HTTP 状态码：
// 全部 up 返回 200，存在 down 返回 503，便于 Docker/部署探针直接判定。
func HTTPStatus(report *HealthReport) int {
	if report.Status == "ok" {
		return http.StatusOK
	}
	return http.StatusServiceUnavailable
}
