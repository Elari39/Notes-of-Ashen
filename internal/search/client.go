package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"notes-of-ashen/internal/config"
)

type Client struct {
	enabled bool
	host    string
	apiKey  string
	index   string
	http    *http.Client
}

const taskWaitTimeout = 2 * time.Minute

type Document struct {
	ID          uint64   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Content     string   `json:"content"`
	Status      string   `json:"status"`
	VisibleAt   int64    `json:"visibleAt"`
	CategoryID  uint64   `json:"categoryId,omitempty"`
	Category    string   `json:"category,omitempty"`
	TagIDs      []uint64 `json:"tagIds"`
	Tags        []string `json:"tags"`
	CreatedAt   int64    `json:"createdAt"`
	PublishedAt int64    `json:"publishedAt,omitempty"`
}

type SearchRequest struct {
	Query      string
	Page       int
	Size       int
	CategoryID uint64
	TagID      uint64
	Now        time.Time
}

type Highlight struct {
	Title   string
	Summary string
	Content string
}

type SearchResult struct {
	IDs        []uint64
	Total      int64
	Highlights map[uint64]Highlight
}

func NewClient(c config.SearchConf) *Client {
	host := strings.TrimRight(strings.TrimSpace(c.MeilisearchHost), "/")
	index := strings.TrimSpace(c.MeilisearchIndex)
	if index == "" {
		index = "articles"
	}
	return &Client{
		enabled: c.Enabled && host != "",
		host:    host,
		apiKey:  strings.TrimSpace(c.MeilisearchAPIKey),
		index:   index,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Client) Search(ctx context.Context, req SearchRequest) (*SearchResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("search is disabled")
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = 10
	}
	body := map[string]interface{}{
		"q":                     req.Query,
		"limit":                 req.Size,
		"offset":                (req.Page - 1) * req.Size,
		"filter":                c.publicFilter(req),
		"attributesToHighlight": []string{"title", "summary", "content"},
		"attributesToCrop":      []string{"content"},
		"cropLength":            32,
		"highlightPreTag":       "<mark>",
		"highlightPostTag":      "</mark>",
	}
	var resp searchResponse
	if err := c.do(ctx, http.MethodPost, "/indexes/"+url.PathEscape(c.index)+"/search", nil, body, &resp); err != nil {
		return nil, err
	}
	result := &SearchResult{
		IDs:        make([]uint64, 0, len(resp.Hits)),
		Total:      resp.EstimatedTotalHits,
		Highlights: make(map[uint64]Highlight, len(resp.Hits)),
	}
	for _, hit := range resp.Hits {
		if hit.ID == 0 {
			continue
		}
		result.IDs = append(result.IDs, hit.ID)
		result.Highlights[hit.ID] = Highlight{
			// Meili 返回的 _formatted 仅在命中词外包 <mark>，原文 HTML 字符未转义。
			// 对 <mark>/</mark> 之外的内容做 HTML 转义，防止前端渲染高亮片段时 XSS。
			Title:   escapeHighlight(hit.Formatted.Title),
			Summary: escapeHighlight(hit.Formatted.Summary),
			Content: escapeHighlight(hit.Formatted.Content),
		}
	}
	return result, nil
}

func (c *Client) Reindex(ctx context.Context, docs []Document) error {
	if !c.Enabled() {
		return fmt.Errorf("search is disabled")
	}
	if err := c.ensureIndex(ctx); err != nil {
		return err
	}
	// 先 deleteAll 再 add 会让索引在重建期间为空，公开搜索出现“搜不到”空窗（P4-6）。
	// 改为：先 add（Meilisearch 按 primaryKey id upsert），再删除不在新文档集合中的旧文档。
	// 这样重建期间索引始终包含新旧数据，仅在删除阶段移除已不存在的文章，无空窗。
	if err := c.configureIndex(ctx); err != nil {
		return err
	}
	if len(docs) > 0 {
		if err := c.addDocuments(ctx, docs); err != nil {
			return err
		}
	}
	if err := c.configureIndex(ctx); err != nil {
		return err
	}
	// 删除不在新集合中的旧文档：构建新 id 集合，遍历现有 id 逐个删除缺失项。
	newIDs := make(map[uint64]struct{}, len(docs))
	for _, d := range docs {
		newIDs[d.ID] = struct{}{}
	}
	stale, err := c.listStaleDocumentIDs(ctx, newIDs)
	if err != nil {
		// 列举失败不影响新文档已入库，仅记日志跳过清理，下次 Reindex 会再清。
		return nil
	}
	for _, id := range stale {
		if err := c.Delete(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// EnsureIndex 在启动阶段一次性创建并配置索引，避免每次 Upsert/Delete 都重复
// ensureIndex + configureIndex 的冗余 HTTP 往返。搜索禁用时为 no-op。
func (c *Client) EnsureIndex(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}
	if err := c.ensureIndex(ctx); err != nil {
		return err
	}
	return c.configureIndex(ctx)
}

func (c *Client) Upsert(ctx context.Context, doc Document) error {
	if !c.Enabled() {
		return nil
	}
	// 索引创建与配置由 EnsureIndex 在启动阶段完成，这里仅推送文档。
	return c.addDocuments(ctx, []Document{doc})
}

func (c *Client) Delete(ctx context.Context, id uint64) error {
	if !c.Enabled() {
		return nil
	}
	path := "/indexes/" + url.PathEscape(c.index) + "/documents/" + strconv.FormatUint(id, 10)
	err := c.submitTask(ctx, http.MethodDelete, path, nil, nil)
	if isMeiliNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) ensureIndex(ctx context.Context) error {
	path := "/indexes/" + url.PathEscape(c.index)
	if err := c.do(ctx, http.MethodGet, path, nil, nil, nil); err == nil {
		return nil
	} else if !isMeiliNotFound(err) {
		return err
	}
	payload := map[string]string{
		"uid":        c.index,
		"primaryKey": "id",
	}
	return c.submitTask(ctx, http.MethodPost, "/indexes", nil, payload)
}

func (c *Client) addDocuments(ctx context.Context, docs []Document) error {
	path := "/indexes/" + url.PathEscape(c.index) + "/documents"
	query := url.Values{"primaryKey": []string{"id"}}
	return c.submitTask(ctx, http.MethodPost, path, query, docs)
}

// listStaleDocumentIDs 列出索引中存在但不在 newIDs 集合内的文档 id，
// 用于 Reindex 时清理已删除文章，避免 deleteAll+add 造成搜索空窗（P4-6）。
func (c *Client) listStaleDocumentIDs(ctx context.Context, newIDs map[uint64]struct{}) ([]uint64, error) {
	var stale []uint64
	const pageSize = 1000
	offset := 0
	for {
		query := url.Values{
			"fields": []string{"id"},
			"limit":  []string{strconv.Itoa(pageSize)},
			"offset": []string{strconv.Itoa(offset)},
		}
		var batch []struct {
			ID uint64 `json:"id"`
		}
		path := "/indexes/" + url.PathEscape(c.index) + "/documents"
		if err := c.do(ctx, http.MethodGet, path, query, nil, &batch); err != nil {
			if isMeiliNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		for _, d := range batch {
			if _, ok := newIDs[d.ID]; !ok {
				stale = append(stale, d.ID)
			}
		}
		if len(batch) < pageSize {
			break
		}
		offset += pageSize
	}
	return stale, nil
}

func (c *Client) configureIndex(ctx context.Context) error {
	settings := map[string]interface{}{
		"searchableAttributes": []string{"title", "summary", "content", "tags", "category"},
		"displayedAttributes":  []string{"id", "title", "summary", "content", "status", "visibleAt", "categoryId", "category", "tagIds", "tags", "createdAt", "publishedAt"},
		"filterableAttributes": []string{"status", "visibleAt", "categoryId", "tagIds"},
		"sortableAttributes":   []string{"visibleAt", "createdAt", "publishedAt"},
	}
	path := "/indexes/" + url.PathEscape(c.index) + "/settings"
	err := c.submitTask(ctx, http.MethodPatch, path, nil, settings)
	if isMeiliNotFound(err) {
		return nil
	}
	return err
}

func (c *Client) publicFilter(req SearchRequest) string {
	now := req.Now
	if now.IsZero() {
		now = time.Now()
	}
	parts := []string{
		`status = "published"`,
		fmt.Sprintf("visibleAt <= %d", now.Unix()),
	}
	if req.CategoryID > 0 {
		parts = append(parts, fmt.Sprintf("categoryId = %d", req.CategoryID))
	}
	if req.TagID > 0 {
		parts = append(parts, fmt.Sprintf("tagIds = %d", req.TagID))
	}
	return strings.Join(parts, " AND ")
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload interface{}, target interface{}) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	u := c.host + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("X-Meili-API-Key", c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return meiliError{status: resp.StatusCode, body: strings.TrimSpace(string(raw))}
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) submitTask(ctx context.Context, method, path string, query url.Values, payload interface{}) error {
	var task meiliTaskResp
	if err := c.do(ctx, method, path, query, payload, &task); err != nil {
		return err
	}
	if task.TaskUID == nil {
		return nil
	}
	return c.waitTask(ctx, *task.TaskUID)
}

func (c *Client) waitTask(ctx context.Context, taskUID int64) error {
	waitCtx, cancel := context.WithTimeout(ctx, taskWaitTimeout)
	defer cancel()

	path := "/tasks/" + strconv.FormatInt(taskUID, 10)
	for {
		var task meiliTaskStatus
		if err := c.do(waitCtx, http.MethodGet, path, nil, nil, &task); err != nil {
			return err
		}
		switch task.Status {
		case "succeeded":
			return nil
		case "failed", "canceled":
			if task.Error != nil && task.Error.Message != "" {
				return meiliTaskFailedError{
					taskUID: taskUID,
					status:  task.Status,
					code:    task.Error.Code,
					message: task.Error.Message,
				}
			}
			return meiliTaskFailedError{taskUID: taskUID, status: task.Status}
		}
		select {
		case <-waitCtx.Done():
			return waitCtx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
}

type searchResponse struct {
	Hits               []searchHit `json:"hits"`
	EstimatedTotalHits int64       `json:"estimatedTotalHits"`
}

// escapeHighlight 对 Meilisearch 高亮片段做 HTML 转义：仅保留 <mark>/</mark> 标签，
// 标签之外的原文 <>&"' 用 html.EscapeString 转义，避免前端渲染时存储型 XSS。
func escapeHighlight(s string) string {
	if !strings.Contains(s, "<mark>") {
		return html.EscapeString(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	rest := s
	for {
		open := strings.Index(rest, "<mark>")
		if open < 0 {
			b.WriteString(html.EscapeString(rest))
			break
		}
		b.WriteString(html.EscapeString(rest[:open]))
		b.WriteString("<mark>")
		rest = rest[open+len("<mark>"):]
		closeIdx := strings.Index(rest, "</mark>")
		if closeIdx < 0 {
			// 缺失闭合标签，转义剩余部分并结束。
			b.WriteString(html.EscapeString(rest))
			break
		}
		b.WriteString(html.EscapeString(rest[:closeIdx]))
		b.WriteString("</mark>")
		rest = rest[closeIdx+len("</mark>"):]
	}
	return b.String()
}

type searchHit struct {
	ID        uint64          `json:"id"`
	Formatted formattedFields `json:"_formatted"`
}

type formattedFields struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

type meiliError struct {
	status int
	body   string
}

type meiliTaskResp struct {
	TaskUID *int64 `json:"taskUid"`
}

type meiliTaskStatus struct {
	Status string          `json:"status"`
	Error  *meiliTaskError `json:"error"`
}

type meiliTaskError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e meiliError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("meilisearch returned status %d", e.status)
	}
	return fmt.Sprintf("meilisearch returned status %d: %s", e.status, e.body)
}

type meiliTaskFailedError struct {
	taskUID int64
	status  string
	code    string
	message string
}

func (e meiliTaskFailedError) Error() string {
	if e.message != "" {
		return fmt.Sprintf("meilisearch task %d %s: %s", e.taskUID, e.status, e.message)
	}
	return fmt.Sprintf("meilisearch task %d %s", e.taskUID, e.status)
}

func isMeiliNotFound(err error) bool {
	if err == nil {
		return false
	}
	var meiliErr meiliError
	if errors.As(err, &meiliErr) && meiliErr.status == http.StatusNotFound {
		return true
	}
	var taskErr meiliTaskFailedError
	return errors.As(err, &taskErr) && taskErr.code == "index_not_found"
}
