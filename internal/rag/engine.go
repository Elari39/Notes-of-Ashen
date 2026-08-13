package rag

import (
	"context"
	"errors"

	"notes-of-ashen/internal/ragclient"
	"notes-of-ashen/model"
)

// ProviderConfig 与模型供应商的运行时配置。该接口让 HTTP 业务层可以解析密钥，
// 而 Worker 只依赖底层 Store，不反向导入 ServiceContext。
func ProviderConfig(settings model.RAGSettings, authSecret string) (ragclient.Config, error) {
	return EffectiveProviderConfig(settings, authSecret)
}

func NormalizeContent(content string) string            { return NormalizeMarkdown(content) }
func ContentHash(title, summary, content string) string { return SourceHash(title, summary, content) }
func SplitContent(content string) []string              { return SplitMarkdown(content) }

func (w *Worker) Query(ctx context.Context, vector []float64, limit int, epoch uint64) ([]ragclient.SearchPoint, error) {
	if w == nil || w.qdrant == nil {
		return nil, errors.New("rag engine is unavailable")
	}
	return w.qdrant.Search(ctx, vector, limit, epoch)
}
