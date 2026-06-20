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
	RegistrationEnabledKey  = "registration_enabled"
	HomeArticleLayoutKey    = "home_article_layout"
	SiteTitleKey            = "site_title"
	SiteDescriptionKey      = "site_description"
	SiteKeywordsKey         = "site_keywords"
	SiteBaseURLKey          = "site_base_url"
	ResumePageEnabledKey    = "resume_page_enabled"
	ResumeNavHiddenKey      = "resume_nav_hidden"
	ProjectsPageEnabledKey  = "projects_page_enabled"
	ProjectsNavHiddenKey    = "projects_nav_hidden"
	ResumeTitleKey          = "resume_title"
	ResumeSubtitleKey       = "resume_subtitle"
	ResumeContentKey        = "resume_content_markdown"
	ProjectsTitleKey        = "projects_title"
	ProjectsSubtitleKey     = "projects_subtitle"
	ProjectsItemsKey        = "projects_items_json"
	AIEnabledKey            = "ai_enabled"
	AIBaseURLKey            = "ai_base_url"
	AIAPIKeyCipherKey       = "ai_api_key_cipher"
	AIModelKey              = "ai_model"
	AIFirstByteTimeoutKey   = "ai_first_byte_timeout_seconds"
	AIStreamTimeoutKey      = "ai_stream_timeout_seconds"
	AINonStreamTimeoutKey   = "ai_non_stream_timeout_seconds"
	AISettingsConfiguredKey = "ai_settings_configured"

	HomeArticleLayoutStandard    = "standard"
	HomeArticleLayoutAlternating = "alternating"

	DefaultSiteTitle       = "Notes of Ashen"
	DefaultSiteDescription = "A personal blog written slowly by the lamp of ink."
	DefaultSiteKeywords    = "blog,notes,writing"
	DefaultResumeTitle     = "简介"
	DefaultProjectsTitle   = "项目"
	DefaultAIFirstByteWait = 60
	DefaultAIStreamWait    = 300
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
	BaseURL                 string
	APIKeyCipher            string
	Model                   string
	FirstByteTimeoutSeconds int
	StreamTimeoutSeconds    int
	NonStreamTimeoutSeconds int
	Configured              bool
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

func (s *Store) SiteSettings(ctx context.Context) (*SiteSettings, error) {
	enabled, err := s.GetBoolSetting(ctx, RegistrationEnabledKey, true)
	if err != nil {
		return nil, err
	}
	layout, err := s.GetStringSetting(ctx, HomeArticleLayoutKey, HomeArticleLayoutStandard)
	if err != nil {
		return nil, err
	}
	title, err := s.GetStringSetting(ctx, SiteTitleKey, DefaultSiteTitle)
	if err != nil {
		return nil, err
	}
	description, err := s.GetStringSetting(ctx, SiteDescriptionKey, DefaultSiteDescription)
	if err != nil {
		return nil, err
	}
	keywords, err := s.GetStringSetting(ctx, SiteKeywordsKey, DefaultSiteKeywords)
	if err != nil {
		return nil, err
	}
	baseURL, err := s.GetStringSetting(ctx, SiteBaseURLKey, "")
	if err != nil {
		return nil, err
	}
	resumeEnabled, err := s.GetBoolSetting(ctx, ResumePageEnabledKey, false)
	if err != nil {
		return nil, err
	}
	resumeHidden, err := s.GetBoolSetting(ctx, ResumeNavHiddenKey, true)
	if err != nil {
		return nil, err
	}
	projectsEnabled, err := s.GetBoolSetting(ctx, ProjectsPageEnabledKey, false)
	if err != nil {
		return nil, err
	}
	projectsHidden, err := s.GetBoolSetting(ctx, ProjectsNavHiddenKey, true)
	if err != nil {
		return nil, err
	}
	return &SiteSettings{
		RegistrationEnabled: enabled,
		HomeArticleLayout:   NormalizeHomeArticleLayout(layout),
		SiteTitle:           title,
		SiteDescription:     description,
		SiteKeywords:        keywords,
		SiteBaseURL:         baseURL,
		ResumePageEnabled:   resumeEnabled,
		ResumeNavHidden:     resumeHidden,
		ProjectsPageEnabled: projectsEnabled,
		ProjectsNavHidden:   projectsHidden,
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
	enabled, err := s.GetBoolSetting(ctx, AIEnabledKey, false)
	if err != nil {
		return nil, err
	}
	baseURL, err := s.GetStringSetting(ctx, AIBaseURLKey, "")
	if err != nil {
		return nil, err
	}
	keyCipher, err := s.GetStringSetting(ctx, AIAPIKeyCipherKey, "")
	if err != nil {
		return nil, err
	}
	model, err := s.GetStringSetting(ctx, AIModelKey, "")
	if err != nil {
		return nil, err
	}
	firstByteTimeout, err := s.GetIntSetting(ctx, AIFirstByteTimeoutKey, DefaultAIFirstByteWait)
	if err != nil {
		return nil, err
	}
	streamTimeout, err := s.GetIntSetting(ctx, AIStreamTimeoutKey, DefaultAIStreamWait)
	if err != nil {
		return nil, err
	}
	nonStreamTimeout, err := s.GetIntSetting(ctx, AINonStreamTimeoutKey, DefaultAINonStreamWait)
	if err != nil {
		return nil, err
	}
	configured, err := s.GetBoolSetting(ctx, AISettingsConfiguredKey, false)
	if err != nil {
		return nil, err
	}
	return &AISettings{
		Enabled:                 enabled,
		BaseURL:                 baseURL,
		APIKeyCipher:            keyCipher,
		Model:                   model,
		FirstByteTimeoutSeconds: firstByteTimeout,
		StreamTimeoutSeconds:    streamTimeout,
		NonStreamTimeoutSeconds: nonStreamTimeout,
		Configured:              configured,
	}, nil
}

func (s *Store) UpdateAISettings(ctx context.Context, settings AISettings) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO site_settings (setting_key, setting_value)
VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)
ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)`,
		AIEnabledKey, boolSettingValue(settings.Enabled),
		AIBaseURLKey, settings.BaseURL,
		AIAPIKeyCipherKey, settings.APIKeyCipher,
		AIModelKey, settings.Model,
		AIFirstByteTimeoutKey, fmt.Sprintf("%d", settings.FirstByteTimeoutSeconds),
		AIStreamTimeoutKey, fmt.Sprintf("%d", settings.StreamTimeoutSeconds),
		AINonStreamTimeoutKey, fmt.Sprintf("%d", settings.NonStreamTimeoutSeconds),
		AISettingsConfiguredKey, "true")
	return err
}

func (s *Store) UpdateRegistrationEnabled(ctx context.Context, enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	return s.UpsertSetting(ctx, RegistrationEnabledKey, value)
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
	title, err := s.GetStringSetting(ctx, ResumeTitleKey, DefaultResumeTitle)
	if err != nil {
		return nil, err
	}
	subtitle, err := s.GetStringSetting(ctx, ResumeSubtitleKey, "")
	if err != nil {
		return nil, err
	}
	content, err := s.GetStringSetting(ctx, ResumeContentKey, "")
	if err != nil {
		return nil, err
	}
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

func (s *Store) UpdateHomeArticleLayout(ctx context.Context, layout string) error {
	return s.UpsertSetting(ctx, HomeArticleLayoutKey, NormalizeHomeArticleLayout(layout))
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
