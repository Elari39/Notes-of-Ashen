package types

import "time"

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type RegisterReq struct {
	Account   string `json:"account"`
	Password  string `json:"password"`
	Email     string `json:"email"`
	EmailCode string `json:"emailCode,optional"`
	Nickname  string `json:"nickname,optional"`
	AvatarURL string `json:"avatarUrl,optional"`
}

type LoginReq struct {
	Account     string `json:"account"`
	Password    string `json:"password"`
	CaptchaID   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken,optional"`
}

type CaptchaReq struct {
	Purpose string `json:"purpose,optional"`
}

type CaptchaResp struct {
	CaptchaID string `json:"captchaId"`
	ImageData string `json:"imageData"`
	ExpiresIn int64  `json:"expiresIn"`
}

type SendVerifyCodeReq struct {
	Email       string `json:"email"`
	Purpose     string `json:"purpose"`
	CaptchaID   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

type ResetPasswordReq struct {
	Email       string `json:"email"`
	EmailCode   string `json:"emailCode"`
	NewPassword string `json:"newPassword"`
}

type SiteSettingsResp struct {
	RegistrationEnabled           bool   `json:"registrationEnabled"`
	RegistrationEmailCodeRequired bool   `json:"registrationEmailCodeRequired"`
	HomeArticleLayout             string `json:"homeArticleLayout"`
	HomeCtaHidden                 bool   `json:"homeCtaHidden"`
	SiteTitle                     string `json:"siteTitle"`
	SiteDescription               string `json:"siteDescription"`
	SiteKeywords                  string `json:"siteKeywords"`
	SiteBaseURL                   string `json:"siteBaseUrl"`
	ProjectsPageEnabled           bool   `json:"projectsPageEnabled"`
	ProjectsNavHidden             bool   `json:"projectsNavHidden"`
}

type UpdateSiteSettingsReq struct {
	RegistrationEnabled *bool   `json:"registrationEnabled,optional"`
	HomeArticleLayout   *string `json:"homeArticleLayout,optional"`
	HomeCtaHidden       *bool   `json:"homeCtaHidden,optional"`
	SiteTitle           *string `json:"siteTitle,optional"`
	SiteDescription     *string `json:"siteDescription,optional"`
	SiteKeywords        *string `json:"siteKeywords,optional"`
	SiteBaseURL         *string `json:"siteBaseUrl,optional"`
	ProjectsPageEnabled *bool   `json:"projectsPageEnabled,optional"`
	ProjectsNavHidden   *bool   `json:"projectsNavHidden,optional"`
}

type ProjectItem struct {
	ID              string   `json:"id"`
	TagIDs          []uint64 `json:"tagIds"`
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

type ProjectsPageResp struct {
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle"`
	Items    []ProjectItem `json:"items"`
}

type UpdateProjectsPageReq struct {
	Title    string        `json:"title"`
	Subtitle string        `json:"subtitle,optional"`
	Items    []ProjectItem `json:"items"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
	Referrer  string
	Host      string
	VisitorID string
}

type UserResp struct {
	ID        uint64    `json:"id"`
	Account   string    `json:"account"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatarUrl"`
	Nickname  string    `json:"nickname"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type UpdateMeReq struct {
	Email     *string `json:"email,optional"`
	EmailCode string  `json:"emailCode,optional"`
	AvatarURL *string `json:"avatarUrl,optional"`
	Nickname  *string `json:"nickname,optional"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
	EmailCode   string `json:"emailCode"`
}

type UserVerifyCodeReq struct {
	Email       string `json:"email,optional"`
	Purpose     string `json:"purpose"`
	CaptchaID   string `json:"captchaId"`
	CaptchaCode string `json:"captchaCode"`
}

type ArticleReq struct {
	CategoryID      uint64     `json:"categoryId,optional"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Summary         string     `json:"summary,optional"`
	Content         string     `json:"content"`
	CoverURL        string     `json:"coverUrl,optional"`
	Status          string     `json:"status,optional"`
	ScheduledAt     *time.Time `json:"scheduledAt,optional"`
	IsPinned        *bool      `json:"isPinned,optional"`
	DisplayPriority *int       `json:"displayPriority,optional"`
	SEOTitle        string     `json:"seoTitle,optional"`
	SEODescription  string     `json:"seoDescription,optional"`
	SEOKeywords     string     `json:"seoKeywords,optional"`
	TagIDs          []uint64   `json:"tagIds,optional"`
}

type ArticleStatusReq struct {
	Status string `json:"status"`
}

type AIAssistReq struct {
	Action  string `json:"action"`
	Title   string `json:"title,optional"`
	Content string `json:"content"`
}

type AIAssistResp struct {
	Title              string   `json:"title,omitempty"`
	Slug               string   `json:"slug,omitempty"`
	Summary            string   `json:"summary,omitempty"`
	SEOTitle           string   `json:"seoTitle,omitempty"`
	SEODescription     string   `json:"seoDescription,omitempty"`
	SEOKeywords        string   `json:"seoKeywords,omitempty"`
	CategorySuggestion string   `json:"categorySuggestion,omitempty"`
	TagSuggestions     []string `json:"tagSuggestions,omitempty"`
	RevisedContent     string   `json:"revisedContent,omitempty"`
	Suggestions        []string `json:"suggestions,omitempty"`
}

type AISettingsResp struct {
	Enabled                 bool   `json:"enabled"`
	APIFormat               string `json:"apiFormat"`
	BaseURL                 string `json:"baseUrl"`
	Model                   string `json:"model"`
	APIKeyConfigured        bool   `json:"apiKeyConfigured"`
	APIKeyNeedsUpdate       bool   `json:"apiKeyNeedsUpdate"`
	FirstByteTimeoutSeconds int    `json:"firstByteTimeoutSeconds"`
	NonStreamTimeoutSeconds int    `json:"nonStreamTimeoutSeconds"`
}

type UpdateAISettingsReq struct {
	Enabled                 bool    `json:"enabled"`
	APIFormat               *string `json:"apiFormat,optional"`
	BaseURL                 *string `json:"baseUrl,optional"`
	APIKey                  string  `json:"apiKey,optional"`
	ClearAPIKey             bool    `json:"clearApiKey,optional"`
	Model                   *string `json:"model,optional"`
	FirstByteTimeoutSeconds *int    `json:"firstByteTimeoutSeconds,optional"`
	NonStreamTimeoutSeconds *int    `json:"nonStreamTimeoutSeconds,optional"`
}

type AIConnectionReq struct {
	APIFormat               string `json:"apiFormat,optional"`
	BaseURL                 string `json:"baseUrl"`
	APIKey                  string `json:"apiKey,optional"`
	FirstByteTimeoutSeconds int    `json:"firstByteTimeoutSeconds,optional"`
	NonStreamTimeoutSeconds int    `json:"nonStreamTimeoutSeconds,optional"`
}

type AIModelsResp struct {
	Models []string `json:"models"`
}

type AIModelTestReq struct {
	APIFormat               string `json:"apiFormat,optional"`
	BaseURL                 string `json:"baseUrl"`
	APIKey                  string `json:"apiKey,optional"`
	Model                   string `json:"model"`
	FirstByteTimeoutSeconds int    `json:"firstByteTimeoutSeconds,optional"`
	NonStreamTimeoutSeconds int    `json:"nonStreamTimeoutSeconds,optional"`
}

type AIModelTestResp struct {
	Model     string `json:"model"`
	LatencyMs int64  `json:"latencyMs"`
}

type ArticleListReq struct {
	Page       int    `json:"page,optional"`
	Size       int    `json:"size,optional"`
	Status     string `json:"status,optional"`
	Query      string `json:"q,optional"`
	CategoryID uint64 `json:"categoryId,optional"`
	TagID      uint64 `json:"tagId,optional"`
}

type ArticleResp struct {
	ID                 uint64                   `json:"id"`
	AuthorID           uint64                   `json:"authorId"`
	CategoryID         uint64                   `json:"categoryId,omitempty"`
	Title              string                   `json:"title"`
	Slug               string                   `json:"slug"`
	Summary            string                   `json:"summary"`
	Content            string                   `json:"content,omitempty"`
	CoverURL           string                   `json:"coverUrl"`
	Status             string                   `json:"status"`
	ViewCount          uint64                   `json:"viewCount"`
	LikeCount          uint64                   `json:"likeCount"`
	WordCount          int                      `json:"wordCount"`
	ReadingTimeMinutes int                      `json:"readingTimeMinutes"`
	ScheduledAt        *time.Time               `json:"scheduledAt,omitempty"`
	PublishedAt        *time.Time               `json:"publishedAt,omitempty"`
	IsPinned           bool                     `json:"isPinned"`
	DisplayPriority    int                      `json:"displayPriority"`
	SEOTitle           string                   `json:"seoTitle"`
	SEODescription     string                   `json:"seoDescription"`
	SEOKeywords        string                   `json:"seoKeywords"`
	CreatedAt          time.Time                `json:"createdAt"`
	UpdatedAt          time.Time                `json:"updatedAt"`
	Tags               []TagResp                `json:"tags"`
	Category           *CategoryResp            `json:"category,omitempty"`
	SearchHighlights   *ArticleSearchHighlights `json:"searchHighlights,omitempty"`
}

type ArticleSearchHighlights struct {
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Content string `json:"content,omitempty"`
}

type ArticleLikeResp struct {
	Liked     bool   `json:"liked"`
	LikeCount uint64 `json:"likeCount"`
}

type ArticleListResp struct {
	Items []ArticleResp `json:"items"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

type SearchReindexResp struct {
	Indexed int  `json:"indexed"`
	Enabled bool `json:"enabled"`
}

type TaxonomyReq struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,optional"`
}

type CategoryResp struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	CreatedBy    uint64    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ArticleCount int64     `json:"articleCount"`
}

type TagResp struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description"`
	CreatedBy    uint64    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	ArticleCount int64     `json:"articleCount"`
}

type SearchSuggestionResp struct {
	Kind         string `json:"kind"`
	ID           uint64 `json:"id"`
	Label        string `json:"label"`
	ArticleCount int64  `json:"articleCount,omitempty"`
}

type SearchSuggestionsResp struct {
	Items []SearchSuggestionResp `json:"items"`
}

type MediaAssetResp struct {
	ID           uint64    `json:"id"`
	StorageKey   string    `json:"storageKey"`
	URL          string    `json:"url"`
	OriginalName string    `json:"originalName"`
	MIMEType     string    `json:"mimeType"`
	SizeBytes    uint64    `json:"sizeBytes"`
	Width        uint      `json:"width"`
	Height       uint      `json:"height"`
	AltText      string    `json:"altText"`
	SHA256       string    `json:"sha256"`
	CreatedBy    uint64    `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type UpdateMediaReq struct {
	AltText string `json:"altText"`
}

type AnalyticsRangeReq struct {
	From string `json:"from,optional"`
	To   string `json:"to,optional"`
}

type AnalyticsSummaryResp struct {
	PV            int64    `json:"pv"`
	UV            int64    `json:"uv"`
	Likes         int64    `json:"likes"`
	PreviousPV    int64    `json:"previousPv"`
	PreviousUV    int64    `json:"previousUv"`
	PreviousLikes int64    `json:"previousLikes"`
	PVChange      *float64 `json:"pvChange,omitempty"`
	UVChange      *float64 `json:"uvChange,omitempty"`
	LikesChange   *float64 `json:"likesChange,omitempty"`
}

type PageAnalyticsResp struct {
	RouteType string `json:"routeType"`
	Path      string `json:"path"`
	ArticleID uint64 `json:"articleId,omitempty"`
	Title     string `json:"title,omitempty"`
	PV        int64  `json:"pv"`
	UV        int64  `json:"uv"`
}

type AnalyticsOverviewResp struct {
	From        string                  `json:"from"`
	To          string                  `json:"to"`
	Summary     AnalyticsSummaryResp    `json:"summary"`
	Trend       []TrafficTrendPointResp `json:"trend"`
	TopPages    []PageAnalyticsResp     `json:"topPages"`
	TopReferers []RefererStatResp       `json:"topReferers"`
}

type ArticleAnalyticsResp struct {
	ArticleID  uint64 `json:"articleId"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	PV         int64  `json:"pv"`
	UV         int64  `json:"uv"`
	Likes      int64  `json:"likes"`
	TotalViews uint64 `json:"totalViews"`
	TotalLikes uint64 `json:"totalLikes"`
}

type ArticleAnalyticsPointResp struct {
	Date  string `json:"date"`
	PV    int64  `json:"pv"`
	UV    int64  `json:"uv"`
	Likes int64  `json:"likes"`
}

type ArticleAnalyticsDetailResp struct {
	Article  ArticleAnalyticsResp        `json:"article"`
	From     string                      `json:"from"`
	To       string                      `json:"to"`
	Trend    []ArticleAnalyticsPointResp `json:"trend"`
	Referers []RefererStatResp           `json:"referers"`
}

type DependencyCheckResp struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMs int64  `json:"latencyMs"`
	Message   string `json:"message,omitempty"`
}

type SystemHealthResp struct {
	Status    string                `json:"status"`
	CheckedAt time.Time             `json:"checkedAt"`
	Checks    []DependencyCheckResp `json:"checks"`
}

type BackupExportReq struct {
	CurrentPassword string `json:"currentPassword"`
	Passphrase      string `json:"passphrase"`
}

type BackupRestoreResp struct {
	Users    int      `json:"users"`
	Articles int      `json:"articles"`
	Media    int      `json:"media"`
	Warnings []string `json:"warnings"`
}

type UserStatusReq struct {
	Status string `json:"status"`
}

type UserRoleReq struct {
	Role string `json:"role"`
}

type ArticleContextResp struct {
	Previous *ArticleResp  `json:"previous,omitempty"`
	Next     *ArticleResp  `json:"next,omitempty"`
	Related  []ArticleResp `json:"related"`
}

type TrafficVisitReq struct {
	Path      string `json:"path"`
	RouteType string `json:"routeType"`
	ArticleID uint64 `json:"articleId,optional"`
	Referrer  string `json:"referrer,optional"`
}

type TrafficTrendPointResp struct {
	Date string `json:"date"`
	PV   int64  `json:"pv"`
	UV   int64  `json:"uv"`
}

type RefererStatResp struct {
	SourceType string `json:"sourceType"`
	SourceName string `json:"sourceName"`
	PV         int64  `json:"pv"`
}

type ArticleVersionResp struct {
	ID                uint64     `json:"id"`
	ArticleID         uint64     `json:"articleId"`
	VersionNo         int        `json:"versionNo"`
	ChangedBy         uint64     `json:"changedBy"`
	AuthorID          uint64     `json:"authorId"`
	CategoryID        uint64     `json:"categoryId,omitempty"`
	Title             string     `json:"title"`
	Slug              string     `json:"slug"`
	Summary           string     `json:"summary"`
	Content           string     `json:"content,omitempty"`
	CoverURL          string     `json:"coverUrl"`
	Status            string     `json:"status"`
	ViewCount         uint64     `json:"viewCount"`
	LikeCount         uint64     `json:"likeCount"`
	ScheduledAt       *time.Time `json:"scheduledAt,omitempty"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
	IsPinned          bool       `json:"isPinned"`
	DisplayPriority   int        `json:"displayPriority"`
	SEOTitle          string     `json:"seoTitle"`
	SEODescription    string     `json:"seoDescription"`
	SEOKeywords       string     `json:"seoKeywords"`
	TagIDs            []uint64   `json:"tagIds"`
	OriginalCreatedAt *time.Time `json:"originalCreatedAt,omitempty"`
	OriginalUpdatedAt *time.Time `json:"originalUpdatedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
}

type ArticleVersionListResp struct {
	Items []ArticleVersionResp `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Size  int                  `json:"size"`
}

type AdminStatsResp struct {
	ArticleTotal    int64                   `json:"articleTotal"`
	PublishedTotal  int64                   `json:"publishedTotal"`
	DraftTotal      int64                   `json:"draftTotal"`
	ArchivedTotal   int64                   `json:"archivedTotal"`
	ScheduledTotal  int64                   `json:"scheduledTotal"`
	ViewTotal       uint64                  `json:"viewTotal"`
	LikeTotal       uint64                  `json:"likeTotal"`
	TodayPV         int64                   `json:"todayPv"`
	TodayUV         int64                   `json:"todayUv"`
	UserTotal       int64                   `json:"userTotal"`
	CategoryTotal   int64                   `json:"categoryTotal"`
	TagTotal        int64                   `json:"tagTotal"`
	TrafficTrend    []TrafficTrendPointResp `json:"trafficTrend"`
	TopReferers     []RefererStatResp       `json:"topReferers"`
	PopularArticles []ArticleResp           `json:"popularArticles"`
	RecentArticles  []ArticleResp           `json:"recentArticles"`
	RecentLogs      []OperationLogResp      `json:"recentLogs"`
}

type OperationLogResp struct {
	ID           uint64    `json:"id"`
	UserID       uint64    `json:"userId,omitempty"`
	UserAccount  string    `json:"userAccount,omitempty"`
	EventType    string    `json:"eventType"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uint64    `json:"resourceId,omitempty"`
	Metadata     string    `json:"metadata,omitempty"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"userAgent"`
	CreatedAt    time.Time `json:"createdAt"`
}

type OperationLogListReq struct {
	Page      int    `json:"page,optional"`
	Size      int    `json:"size,optional"`
	EventType string `json:"eventType,optional"`
	Actor     string `json:"actor,optional"`
	IP        string `json:"ip,optional"`
	StartAt   string `json:"startAt,optional"`
	EndAt     string `json:"endAt,optional"`
}

type ListResp[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}
