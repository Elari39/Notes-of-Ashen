package ragclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// QdrantClient 只连接由进程配置提供的内部 Qdrant 服务。它不使用公共 AI
// HTTP transport，避免把 Docker 私有地址错误地交给 SSRF 防护拒绝。
type QdrantClient struct {
	baseURL    *url.URL
	apiKey     string
	collection string
	client     *http.Client
}

type Point struct {
	ID      string
	Vector  []float64
	Payload PointPayload
}

type PointPayload struct {
	ArticleID            uint64 `json:"articleId"`
	ChunkIndex           int    `json:"chunkIndex"`
	SourceHash           string `json:"sourceHash"`
	Content              string `json:"content"`
	EmbeddingFingerprint string `json:"embeddingFingerprint"`
	Epoch                uint64 `json:"epoch"`
}

type SearchPoint struct {
	ID      string
	Score   float64
	Payload PointPayload
}

func NewQdrantClient(baseURL, apiKey, collection string) (*QdrantClient, error) {
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("qdrant url is invalid")
	}
	if strings.TrimSpace(collection) == "" {
		return nil, errors.New("qdrant collection is required")
	}
	return &QdrantClient{
		baseURL: parsed, apiKey: strings.TrimSpace(apiKey), collection: strings.TrimSpace(collection),
		client: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func (c *QdrantClient) Collection() string { return c.collection }

func (c *QdrantClient) Health(ctx context.Context) error {
	_, err := c.request(ctx, http.MethodGet, "/healthz", nil, nil)
	return err
}

func (c *QdrantClient) RecreateCollection(ctx context.Context, dimensions int) error {
	if dimensions <= 0 {
		return errors.New("qdrant vector dimensions are invalid")
	}
	payload := map[string]any{"vectors": map[string]any{"size": dimensions, "distance": "Cosine"}}
	_, err := c.request(ctx, http.MethodPut, "/collections/"+url.PathEscape(c.collection), payload, nil)
	return err
}

func (c *QdrantClient) DeleteCollection(ctx context.Context) error {
	_, err := c.request(ctx, http.MethodDelete, "/collections/"+url.PathEscape(c.collection), nil, nil)
	if statusErr, ok := err.(*HTTPStatusError); ok && statusErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

func (c *QdrantClient) DeleteArticle(ctx context.Context, articleID uint64) error {
	payload := map[string]any{"filter": map[string]any{"must": []any{map[string]any{"key": "articleId", "match": map[string]any{"value": articleID}}}}}
	_, err := c.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(c.collection)+"/points/delete?wait=true", payload, nil)
	return err
}

// DeleteArticleAtEpoch 只删除指定索引 epoch 的文章分段。普通同步任务可能在另一
// 实例开始重建后才结束；epoch 过滤可确保旧任务绝不会删掉新 collection 中的向量。
func (c *QdrantClient) DeleteArticleAtEpoch(ctx context.Context, articleID, epoch uint64) error {
	payload := map[string]any{"filter": map[string]any{"must": []any{
		map[string]any{"key": "articleId", "match": map[string]any{"value": articleID}},
		map[string]any{"key": "epoch", "match": map[string]any{"value": epoch}},
	}}}
	_, err := c.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(c.collection)+"/points/delete?wait=true", payload, nil)
	return err
}

func (c *QdrantClient) Upsert(ctx context.Context, points []Point) error {
	if len(points) == 0 {
		return nil
	}
	type qdrantPoint struct {
		ID      string       `json:"id"`
		Vector  []float64    `json:"vector"`
		Payload PointPayload `json:"payload"`
	}
	items := make([]qdrantPoint, 0, len(points))
	for _, point := range points {
		items = append(items, qdrantPoint{ID: point.ID, Vector: point.Vector, Payload: point.Payload})
	}
	_, err := c.request(ctx, http.MethodPut, "/collections/"+url.PathEscape(c.collection)+"/points?wait=true", map[string]any{"points": items}, nil)
	return err
}

func (c *QdrantClient) Search(ctx context.Context, vector []float64, limit int, epoch uint64) ([]SearchPoint, error) {
	if len(vector) == 0 {
		return []SearchPoint{}, nil
	}
	if limit < 1 {
		limit = 24
	}
	payload := map[string]any{
		"vector": vector, "limit": limit, "with_payload": true,
		"filter": map[string]any{"must": []any{map[string]any{"key": "epoch", "match": map[string]any{"value": epoch}}}},
	}
	raw, err := c.request(ctx, http.MethodPost, "/collections/"+url.PathEscape(c.collection)+"/points/search", payload, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Result []struct {
			ID      json.RawMessage `json:"id"`
			Score   float64         `json:"score"`
			Payload PointPayload    `json:"payload"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("decode qdrant search response: %w", err)
	}
	result := make([]SearchPoint, 0, len(response.Result))
	for _, item := range response.Result {
		id := strings.Trim(string(item.ID), "\"")
		result = append(result, SearchPoint{ID: id, Score: item.Score, Payload: item.Payload})
	}
	return result, nil
}

func (c *QdrantClient) request(ctx context.Context, method, requestPath string, payload any, _ any) ([]byte, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(c.baseURL.Path, requestPath)
	if strings.Contains(requestPath, "?") {
		parts := strings.SplitN(requestPath, "?", 2)
		endpoint.Path = path.Join(c.baseURL.Path, parts[0])
		endpoint.RawQuery = parts[1]
	}
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode qdrant request: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create qdrant request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send qdrant request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Qdrant 的错误体也可能包含其拒绝的 payload；该 payload 中含文章分段，
		// 因此只有限读取以复用连接，不能把正文塞进会被 Worker 记录的错误。
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorBytes+1))
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Message:    "qdrant http status " + fmt.Sprintf("%d", resp.StatusCode),
		}
	}
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if readErr != nil {
		return nil, fmt.Errorf("read qdrant response: %w", readErr)
	}
	if len(raw) > maxResponseBytes {
		return nil, errors.New("qdrant response body exceeds limit")
	}
	return raw, nil
}
