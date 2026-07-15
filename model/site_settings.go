package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const (
	RegistrationEnabledKey = "registration_enabled"
	HomeArticleLayoutKey   = "home_article_layout"
	SiteTitleKey           = "site_title"
	SiteDescriptionKey     = "site_description"
	SiteKeywordsKey        = "site_keywords"
	SiteBaseURLKey         = "site_base_url"
	ResumePageEnabledKey   = "resume_page_enabled"
	ResumeNavHiddenKey     = "resume_nav_hidden"
	ProjectsPageEnabledKey = "projects_page_enabled"
	ProjectsNavHiddenKey   = "projects_nav_hidden"
	ResumeTitleKey         = "resume_title"
	ResumeSubtitleKey      = "resume_subtitle"
	ResumeContentKey       = "resume_content_markdown"
	ProjectsTitleKey       = "projects_title"
	ProjectsSubtitleKey    = "projects_subtitle"
	ProjectsItemsKey       = "projects_items_json"
	AIEnabledKey           = "ai_enabled"
	AIAPIFormatKey         = "ai_api_format"
	AIBaseURLKey           = "ai_base_url"
	AIAPIKeyCipherKey      = "ai_api_key_cipher"
	AIModelKey             = "ai_model"
	AIFirstByteTimeoutKey  = "ai_first_byte_timeout_seconds"
	AINonStreamTimeoutKey  = "ai_non_stream_timeout_seconds"

	HomeArticleLayoutStandard    = "standard"
	HomeArticleLayoutAlternating = "alternating"
	AIAPIFormatOpenAI            = "openai"
	AIAPIFormatAnthropic         = "anthropic"

	DefaultSiteTitle       = "Notes of Ashen"
	DefaultSiteDescription = "A personal blog written slowly by the lamp of ink."
	DefaultSiteKeywords    = "blog,notes,writing"
	DefaultResumeTitle     = "简介"
	DefaultProjectsTitle   = "项目"
	DefaultAIAPIFormat     = AIAPIFormatOpenAI
	DefaultAIFirstByteWait = 60
	DefaultAINonStreamWait = 600
)

type SiteSettings struct {
	RegistrationEnabled bool
	HomeArticleLayout   string
	SiteTitle           string
	SiteDescription     string
	SiteKeywords        string
	SiteBaseURL         string
	ResumePageEnabled   bool
	ResumeNavHidden     bool
	ProjectsPageEnabled bool
	ProjectsNavHidden   bool
}

type AISettings struct {
	Enabled                 bool
	APIFormat               string
	BaseURL                 string
	APIKeyCipher            string
	Model                   string
	FirstByteTimeoutSeconds int
	NonStreamTimeoutSeconds int
}

type ResumePageContent struct {
	Title           string
	Subtitle        string
	ContentMarkdown string
	Experiences     []ResumeExperience
	Educations      []ResumeEducation
	Skills          []ResumeSkill
}

type ProjectItem struct {
	ID              string   `json:"id"`
	TagIDs          []uint64 `json:"tagIds,omitempty"`
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Role            string   `json:"role"`
	Period          string   `json:"period"`
	Tags            []string `json:"tags"`
	CoverURL        string   `json:"coverUrl"`
	DemoURL         string   `json:"demoUrl"`
	RepoURL         string   `json:"repoUrl"`
	ContentMarkdown string   `json:"contentMarkdown"`
	Featured        bool     `json:"featured"`
}

type ProjectsPageContent struct {
	Title    string
	Subtitle string
	Items    []ProjectItem
}

// GetSettingsBatch 一次性加载多个 setting_key，返回 key->value 映射。
// 缺失的 key 不在结果中，由调用方按默认值回退。表不存在（1146）时返回空映射，
// 与单条 Get*Setting 的降级语义保持一致。
func (s *Store) GetSettingsBatch(ctx context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	if len(keys) == 0 {
		return result, nil
	}
	placeholders := make([]string, len(keys))
	args := make([]any, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	query := "SELECT setting_key, setting_value FROM site_settings WHERE setting_key IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return result, nil
		}
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) SiteSettings(ctx context.Context) (*SiteSettings, error) {
	keys := []string{
		RegistrationEnabledKey, HomeArticleLayoutKey, SiteTitleKey, SiteDescriptionKey,
		SiteKeywordsKey, SiteBaseURLKey, ResumePageEnabledKey, ResumeNavHiddenKey,
		ProjectsPageEnabledKey, ProjectsNavHiddenKey,
	}
	values, err := s.GetSettingsBatch(ctx, keys)
	if err != nil {
		return nil, err
	}
	getString := func(key, def string) string {
		if v, ok := values[key]; ok {
			return v
		}
		return def
	}
	getBool := func(key string, def bool) bool {
		if v, ok := values[key]; ok {
			return v == "true" || v == "1"
		}
		return def
	}
	return &SiteSettings{
		RegistrationEnabled: getBool(RegistrationEnabledKey, true),
		HomeArticleLayout:   NormalizeHomeArticleLayout(getString(HomeArticleLayoutKey, HomeArticleLayoutStandard)),
		SiteTitle:           getString(SiteTitleKey, DefaultSiteTitle),
		SiteDescription:     getString(SiteDescriptionKey, DefaultSiteDescription),
		SiteKeywords:        getString(SiteKeywordsKey, DefaultSiteKeywords),
		SiteBaseURL:         getString(SiteBaseURLKey, ""),
		ResumePageEnabled:   getBool(ResumePageEnabledKey, false),
		ResumeNavHidden:     getBool(ResumeNavHiddenKey, true),
		ProjectsPageEnabled: getBool(ProjectsPageEnabledKey, false),
		ProjectsNavHidden:   getBool(ProjectsNavHiddenKey, true),
	}, nil
}

func (s *Store) GetBoolSetting(ctx context.Context, key string, defaultValue bool) (bool, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1", key).Scan(&raw)
	if err != nil {
		if errors.Is(scanErr(err), ErrNotFound) {
			return defaultValue, nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return defaultValue, nil
		}
		return false, err
	}
	return raw == "true" || raw == "1", nil
}

func (s *Store) GetStringSetting(ctx context.Context, key string, defaultValue string) (string, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT setting_value FROM site_settings WHERE setting_key = ? LIMIT 1", key).Scan(&raw)
	if err != nil {
		if errors.Is(scanErr(err), ErrNotFound) {
			return defaultValue, nil
		}
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1146 {
			return defaultValue, nil
		}
		return "", err
	}
	return raw, nil
}

func (s *Store) GetIntSetting(ctx context.Context, key string, defaultValue int) (int, error) {
	raw, err := s.GetStringSetting(ctx, key, "")
	if err != nil {
		return 0, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
		return defaultValue, nil
	}
	return value, nil
}

func (s *Store) AISettings(ctx context.Context) (*AISettings, error) {
	keys := []string{
		AIEnabledKey, AIAPIFormatKey, AIBaseURLKey, AIAPIKeyCipherKey, AIModelKey,
		AIFirstByteTimeoutKey, AINonStreamTimeoutKey,
	}
	values, err := s.GetSettingsBatch(ctx, keys)
	if err != nil {
		return nil, err
	}
	getString := func(key, def string) string {
		if v, ok := values[key]; ok {
			return v
		}
		return def
	}
	getBool := func(key string, def bool) bool {
		if v, ok := values[key]; ok {
			return v == "true" || v == "1"
		}
		return def
	}
	getInt := func(key string, def int) int {
		raw := strings.TrimSpace(getString(key, ""))
		if raw == "" {
			return def
		}
		var value int
		if _, err := fmt.Sscanf(raw, "%d", &value); err != nil {
			return def
		}
		return value
	}
	return &AISettings{
		Enabled:                 getBool(AIEnabledKey, false),
		APIFormat:               NormalizeAIAPIFormat(getString(AIAPIFormatKey, DefaultAIAPIFormat)),
		BaseURL:                 getString(AIBaseURLKey, ""),
		APIKeyCipher:            getString(AIAPIKeyCipherKey, ""),
		Model:                   getString(AIModelKey, ""),
		FirstByteTimeoutSeconds: getInt(AIFirstByteTimeoutKey, DefaultAIFirstByteWait),
		NonStreamTimeoutSeconds: getInt(AINonStreamTimeoutKey, DefaultAINonStreamWait),
	}, nil
}

func (s *Store) UpdateAISettings(ctx context.Context, settings AISettings) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		AIEnabledKey, boolSettingValue(settings.Enabled),
		AIAPIFormatKey, NormalizeAIAPIFormat(settings.APIFormat),
		AIBaseURLKey, settings.BaseURL,
		AIAPIKeyCipherKey, settings.APIKeyCipher,
		AIModelKey, settings.Model,
		AIFirstByteTimeoutKey, fmt.Sprintf("%d", settings.FirstByteTimeoutSeconds),
		AINonStreamTimeoutKey, fmt.Sprintf("%d", settings.NonStreamTimeoutSeconds))
	return err
}

func (s *Store) UpdateSiteSettings(ctx context.Context, settings SiteSettings) error {
	value := "false"
	if settings.RegistrationEnabled {
		value = "true"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		RegistrationEnabledKey, value,
		HomeArticleLayoutKey, NormalizeHomeArticleLayout(settings.HomeArticleLayout),
		SiteTitleKey, settings.SiteTitle,
		SiteDescriptionKey, settings.SiteDescription,
		SiteKeywordsKey, settings.SiteKeywords,
		SiteBaseURLKey, settings.SiteBaseURL,
		ResumePageEnabledKey, boolSettingValue(settings.ResumePageEnabled),
		ResumeNavHiddenKey, boolSettingValue(settings.ResumeNavHidden),
		ProjectsPageEnabledKey, boolSettingValue(settings.ProjectsPageEnabled),
		ProjectsNavHiddenKey, boolSettingValue(settings.ProjectsNavHidden))
	return err
}

func (s *Store) ResumePageContent(ctx context.Context) (*ResumePageContent, error) {
	values, err := s.GetSettingsBatch(ctx, []string{ResumeTitleKey, ResumeSubtitleKey, ResumeContentKey})
	if err != nil {
		return nil, err
	}
	title := values[ResumeTitleKey]
	if title == "" {
		title = DefaultResumeTitle
	}
	subtitle := values[ResumeSubtitleKey]
	content := values[ResumeContentKey]
	experiences, err := s.ListResumeExperiences(ctx)
	if err != nil {
		return nil, err
	}
	educations, err := s.ListResumeEducations(ctx)
	if err != nil {
		return nil, err
	}
	skills, err := s.ListResumeSkills(ctx)
	if err != nil {
		return nil, err
	}
	return &ResumePageContent{
		Title:           title,
		Subtitle:        subtitle,
		ContentMarkdown: content,
		Experiences:     experiences,
		Educations:      educations,
		Skills:          skills,
	}, nil
}

func (s *Store) UpdateResumePageContent(ctx context.Context, content ResumePageContent) error {
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
			ResumeTitleKey, content.Title,
			ResumeSubtitleKey, content.Subtitle,
			ResumeContentKey, content.ContentMarkdown); err != nil {
			return err
		}
		if err := replaceResumeExperiencesTx(ctx, tx, content.Experiences); err != nil {
			return err
		}
		if err := replaceResumeEducationsTx(ctx, tx, content.Educations); err != nil {
			return err
		}
		return replaceResumeSkillsTx(ctx, tx, content.Skills)
	})
}

func (s *Store) ProjectsPageContent(ctx context.Context) (*ProjectsPageContent, error) {
	title, err := s.GetStringSetting(ctx, ProjectsTitleKey, DefaultProjectsTitle)
	if err != nil {
		return nil, err
	}
	subtitle, err := s.GetStringSetting(ctx, ProjectsSubtitleKey, "")
	if err != nil {
		return nil, err
	}
	entityItems, err := s.ListProjectItems(ctx)
	if err != nil {
		return nil, err
	}
	if len(entityItems) > 0 {
		return &ProjectsPageContent{
			Title:    title,
			Subtitle: subtitle,
			Items:    entityItems,
		}, nil
	}
	rawItems, err := s.GetStringSetting(ctx, ProjectsItemsKey, "[]")
	if err != nil {
		return nil, err
	}
	items, err := DecodeProjectItems(rawItems)
	if err != nil {
		return nil, err
	}
	return &ProjectsPageContent{
		Title:    title,
		Subtitle: subtitle,
		Items:    items,
	}, nil
}

func (s *Store) UpdateProjectsPageContent(ctx context.Context, content ProjectsPageContent) error {
	items := NormalizeProjectItems(content.Items)
	// 将归一化后的 items 同步写入 projects_items_json 列作为快照，
	// 保证独立表为空时的读路径回退（见 ProjectsPageContent）仍能拿到数据。
	encoded, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
			ProjectsTitleKey, content.Title,
			ProjectsSubtitleKey, content.Subtitle,
			ProjectsItemsKey, string(encoded)); err != nil {
			return err
		}
		return replaceProjectItemsTx(ctx, tx, items)
	})
}

func (s *Store) UpsertSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		key, value)
	return err
}

func NormalizeHomeArticleLayout(layout string) string {
	if layout == HomeArticleLayoutAlternating {
		return HomeArticleLayoutAlternating
	}
	return HomeArticleLayoutStandard
}

func NormalizeAIAPIFormat(apiFormat string) string {
	switch strings.ToLower(strings.TrimSpace(apiFormat)) {
	case AIAPIFormatAnthropic:
		return AIAPIFormatAnthropic
	case AIAPIFormatOpenAI:
		return AIAPIFormatOpenAI
	default:
		return DefaultAIAPIFormat
	}
}

func boolSettingValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func DecodeProjectItems(raw string) ([]ProjectItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []ProjectItem{}, nil
	}
	var items []ProjectItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	return NormalizeProjectItems(items), nil
}

func NormalizeProjectItems(items []ProjectItem) []ProjectItem {
	normalized := make([]ProjectItem, 0, len(items))
	for index, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		if item.ID == "" {
			item.ID = fmt.Sprintf("project-%d", index+1)
		}
		item.Title = strings.TrimSpace(item.Title)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Role = strings.TrimSpace(item.Role)
		item.Period = strings.TrimSpace(item.Period)
		item.CoverURL = strings.TrimSpace(item.CoverURL)
		item.DemoURL = strings.TrimSpace(item.DemoURL)
		item.RepoURL = strings.TrimSpace(item.RepoURL)
		item.Tags = normalizeProjectTags(item.Tags)
		item.TagIDs = uniqueUint64(item.TagIDs)
		normalized = append(normalized, item)
	}
	return normalized
}

func normalizeProjectTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
}
