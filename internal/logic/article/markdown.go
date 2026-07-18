package article

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/logicutil"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"

	"gopkg.in/yaml.v2"
)

type markdownImportDoc struct {
	Title           string
	Slug            string
	Summary         string
	Content         string
	Category        string
	Tags            []string
	ScheduledAt     *time.Time
	CoverURL        string
	IsPinned        bool
	DisplayPriority int
	SEOTitle        string
	SEODescription  string
	SEOKeywords     string
}

type markdownExportFrontMatter struct {
	Title          string   `yaml:"title"`
	Slug           string   `yaml:"slug"`
	Summary        string   `yaml:"summary,omitempty"`
	Category       string   `yaml:"category,omitempty"`
	Tags           []string `yaml:"tags,omitempty"`
	Date           string   `yaml:"date,omitempty"`
	CoverURL       string   `yaml:"cover_url,omitempty"`
	Pinned         bool     `yaml:"pinned"`
	Priority       int      `yaml:"priority"`
	SEOTitle       string   `yaml:"seo_title,omitempty"`
	SEODescription string   `yaml:"seo_description,omitempty"`
	SEOKeywords    string   `yaml:"seo_keywords,omitempty"`
}

func ImportMarkdown(ctx context.Context, svcCtx *svc.ServiceContext, filename, content string, meta types.RequestMeta) (*types.ArticleResp, error) {
	if err := authutil.RequireContentManager(ctx); err != nil {
		return nil, err
	}
	userID, err := authutil.UserID(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := parseMarkdownImport(filename, content)
	if err != nil {
		return nil, err
	}
	slug, err := uniqueArticleSlug(ctx, svcCtx, doc.Slug)
	if err != nil {
		return nil, err
	}
	req := types.ArticleReq{
		Title:           doc.Title,
		Slug:            slug,
		Summary:         doc.Summary,
		Content:         doc.Content,
		CoverURL:        doc.CoverURL,
		Status:          model.ArticleStatusDraft,
		ScheduledAt:     doc.ScheduledAt,
		IsPinned:        boolPtr(doc.IsPinned),
		DisplayPriority: intPtr(doc.DisplayPriority),
		SEOTitle:        doc.SEOTitle,
		SEODescription:  doc.SEODescription,
		SEOKeywords:     doc.SEOKeywords,
	}
	if err := validateArticleFields(req); err != nil {
		return nil, err
	}
	category, tags, err := importTaxonomies(userID, doc.Category, doc.Tags)
	if err != nil {
		return nil, err
	}
	id, err := svcCtx.Store.CreateMarkdownArticle(ctx, model.MarkdownArticleImport{
		Article: articleCreateFromReq(req, userID), Category: category, Tags: tags,
	})
	if err != nil {
		if logicutil.IsDuplicate(err) {
			return nil, apperrors.Conflict("article slug already exists")
		}
		return nil, logicutil.MapError(err)
	}
	return finishArticleCreate(ctx, svcCtx, id, userID, meta)
}

func ExportMarkdown(ctx context.Context, svcCtx *svc.ServiceContext, id uint64) (string, string, error) {
	userID, role, err := currentActor(ctx)
	if err != nil {
		return "", "", err
	}
	item, err := svcCtx.Store.FindArticle(ctx, id)
	if err != nil {
		return "", "", logicutil.MapError(err)
	}
	if err := canManageArticle(userID, role, *item); err != nil {
		return "", "", err
	}
	tags, err := svcCtx.Store.ArticleTags(ctx, item.ID)
	if err != nil {
		return "", "", err
	}
	var categoryName string
	if item.CategoryID > 0 {
		if category, err := svcCtx.Store.FindCategory(ctx, item.CategoryID); err == nil {
			categoryName = category.Name
		}
	}
	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	var date string
	if item.PublishedAt != nil {
		date = item.PublishedAt.Format(time.RFC3339)
	} else if item.ScheduledAt != nil {
		date = item.ScheduledAt.Format(time.RFC3339)
	}
	header, err := yaml.Marshal(markdownExportFrontMatter{
		Title:          item.Title,
		Slug:           item.Slug,
		Summary:        item.Summary,
		Category:       categoryName,
		Tags:           tagNames,
		Date:           date,
		CoverURL:       item.CoverURL,
		Pinned:         item.IsPinned,
		Priority:       item.DisplayPriority,
		SEOTitle:       item.SEOTitle,
		SEODescription: item.SEODescription,
		SEOKeywords:    item.SEOKeywords,
	})
	if err != nil {
		return "", "", err
	}
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.Write(header)
	builder.WriteString("---\n\n")
	builder.WriteString(strings.TrimSpace(item.Content))
	builder.WriteString("\n")
	return safeMarkdownFilename(item.Slug, item.Title), builder.String(), nil
}

func parseMarkdownImport(filename, raw string) (markdownImportDoc, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	meta, body, err := splitFrontMatter(raw)
	if err != nil {
		return markdownImportDoc{}, err
	}
	title := stringField(meta, "title")
	usedH1 := false
	if title == "" {
		var h1 string
		h1, body, usedH1 = firstH1Title(body)
		title = h1
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled"
	}
	body = strings.TrimSpace(body)
	if body == "" {
		if usedH1 {
			return markdownImportDoc{}, apperrors.BadRequest("markdown content is empty after title extraction")
		}
		return markdownImportDoc{}, apperrors.BadRequest("markdown content is empty")
	}
	slug := slugify(stringField(meta, "slug"))
	if slug == "" {
		slug = slugify(title)
	}
	if slug == "" {
		slug = fmt.Sprintf("article-%d", time.Now().Unix())
	}
	return markdownImportDoc{
		Title:           title,
		Slug:            slug,
		Summary:         strings.TrimSpace(stringField(meta, "summary")),
		Content:         body,
		Category:        taxonomyField(meta["category"]),
		Tags:            tagsField(meta["tags"]),
		ScheduledAt:     timeField(meta, "date", "published_at", "publishedAt"),
		CoverURL:        strings.TrimSpace(stringField(meta, "cover_url", "coverUrl")),
		IsPinned:        boolField(meta, "pinned", "is_pinned", "isPinned"),
		DisplayPriority: intField(meta, "priority", "display_priority", "displayPriority"),
		SEOTitle:        strings.TrimSpace(stringField(meta, "seo_title", "seoTitle")),
		SEODescription:  strings.TrimSpace(stringField(meta, "seo_description", "seoDescription")),
		SEOKeywords:     seoKeywordsField(meta["seo_keywords"], meta["seoKeywords"]),
	}, nil
}

func splitFrontMatter(raw string) (map[string]interface{}, string, error) {
	if !strings.HasPrefix(raw, "---\n") {
		return map[string]interface{}{}, raw, nil
	}
	end := strings.Index(raw[4:], "\n---\n")
	if end < 0 {
		return nil, "", apperrors.BadRequest("front matter is invalid")
	}
	yamlRaw := raw[4 : 4+end]
	body := raw[4+end+5:]
	meta := map[string]interface{}{}
	if strings.TrimSpace(yamlRaw) == "" {
		return meta, body, nil
	}
	if err := yaml.Unmarshal([]byte(yamlRaw), &meta); err != nil {
		return nil, "", apperrors.BadRequest("front matter is invalid")
	}
	return meta, body, nil
}

func firstH1Title(content string) (string, string, bool) {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			nextLines := append([]string{}, lines[:i]...)
			nextLines = append(nextLines, lines[i+1:]...)
			return title, strings.Join(nextLines, "\n"), true
		}
	}
	return "", content, false
}

func stringField(meta map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := meta[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func taxonomyField(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[interface{}]interface{}:
		if name, ok := v["name"].(string); ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
		if slug, ok := v["slug"].(string); ok {
			return strings.TrimSpace(slug)
		}
	case map[string]interface{}:
		if name, ok := v["name"].(string); ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
		if slug, ok := v["slug"].(string); ok {
			return strings.TrimSpace(slug)
		}
	}
	return ""
}

func tagsField(value interface{}) []string {
	switch v := value.(type) {
	case []interface{}:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				items = append(items, strings.TrimSpace(text))
			}
		}
		return compactStrings(items)
	case []string:
		return compactStrings(v)
	case string:
		return compactStrings(strings.Split(v, ","))
	default:
		return nil
	}
}

func seoKeywordsField(values ...interface{}) string {
	for _, value := range values {
		switch v := value.(type) {
		case string:
			return strings.TrimSpace(v)
		case []interface{}:
			keywords := make([]string, 0, len(v))
			for _, item := range v {
				if text, ok := item.(string); ok {
					keywords = append(keywords, strings.TrimSpace(text))
				}
			}
			return strings.Join(compactStrings(keywords), ", ")
		case []string:
			return strings.Join(compactStrings(v), ", ")
		}
	}
	return ""
}

func timeField(meta map[string]interface{}, keys ...string) *time.Time {
	for _, key := range keys {
		value, ok := meta[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case time.Time:
			return &v
		case string:
			if parsed := parseImportTime(v); parsed != nil {
				return parsed
			}
		}
	}
	return nil
}

func boolField(meta map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		value, ok := meta[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v
		case string:
			parsed, err := strconv.ParseBool(strings.TrimSpace(v))
			if err == nil {
				return parsed
			}
		}
	}
	return false
}

func intField(meta map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		value, ok := meta[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func parseImportTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed
		}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func importTaxonomies(userID uint64, categoryName string, tagNames []string) (*model.TaxonomyCreate, []model.TaxonomyCreate, error) {
	var category *model.TaxonomyCreate
	name := strings.TrimSpace(categoryName)
	if name != "" {
		if len([]rune(name)) > 64 {
			return nil, nil, apperrors.BadRequest("category is too long")
		}
		slug := slugify(name)
		if slug == "" {
			slug = fmt.Sprintf("category-%d", time.Now().UnixNano())
		}
		category = &model.TaxonomyCreate{Name: name, Slug: slug, CreatedBy: userID}
	}

	names := compactStrings(tagNames)
	tags := make([]model.TaxonomyCreate, 0, len(names))
	seen := map[string]struct{}{}
	for _, name := range names {
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if len([]rune(name)) > 64 {
			return nil, nil, apperrors.BadRequest("tag is too long")
		}
		slug := slugify(name)
		if slug == "" {
			slug = fmt.Sprintf("tag-%d", time.Now().UnixNano())
		}
		tags = append(tags, model.TaxonomyCreate{Name: name, Slug: slug, CreatedBy: userID})
	}
	return category, tags, nil
}

func uniqueArticleSlug(ctx context.Context, svcCtx *svc.ServiceContext, slug string) (string, error) {
	base := trimSlug(slug)
	if base == "" {
		base = fmt.Sprintf("article-%d", time.Now().Unix())
	}
	// 一次性查询 base 本身与 base-{n} 前缀族已占用的 slug，避免逐次 DB 往返。
	// 并发场景下仍由 articles.slug 唯一约束兜底（CreateArticle 返回冲突错误）。
	taken, err := svcCtx.Store.ArticleSlugsTakenByPrefix(ctx, base)
	if err != nil {
		return "", err
	}
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			suffix := fmt.Sprintf("-%d", i+1)
			candidate = trimSlug(base)
			if len([]rune(candidate))+len([]rune(suffix)) > 180 {
				candidate = string([]rune(candidate)[:180-len([]rune(suffix))])
			}
			candidate += suffix
		}
		if _, exists := taken[candidate]; !exists {
			return candidate, nil
		}
	}
	return "", apperrors.Conflict("article slug already exists")
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastHyphen := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				builder.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return trimSlug(builder.String())
}

func trimSlug(value string) string {
	value = strings.Trim(strings.ToLower(strings.TrimSpace(value)), "-")
	runes := []rune(value)
	if len(runes) > 180 {
		value = strings.Trim(string(runes[:180]), "-")
	}
	return value
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func safeMarkdownFilename(slug, title string) string {
	base := slugify(slug)
	if base == "" {
		base = slugify(title)
	}
	if base == "" {
		base = "article"
	}
	return base + ".md"
}
