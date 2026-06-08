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
	EmailCode string `json:"emailCode"`
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
	RefreshToken string `json:"refreshToken"`
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
	RegistrationEnabled bool   `json:"registrationEnabled"`
	HomeArticleLayout   string `json:"homeArticleLayout"`
	SiteTitle           string `json:"siteTitle"`
	SiteDescription     string `json:"siteDescription"`
	SiteKeywords        string `json:"siteKeywords"`
	SiteBaseURL         string `json:"siteBaseUrl"`
}

type UpdateSiteSettingsReq struct {
	RegistrationEnabled *bool  `json:"registrationEnabled,optional"`
	HomeArticleLayout   string `json:"homeArticleLayout"`
	SiteTitle           string `json:"siteTitle,optional"`
	SiteDescription     string `json:"siteDescription,optional"`
	SiteKeywords        string `json:"siteKeywords,optional"`
	SiteBaseURL         string `json:"siteBaseUrl,optional"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
	Referrer  string
	Host      string
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
	Email     string `json:"email,optional"`
	EmailCode string `json:"emailCode,optional"`
	AvatarURL string `json:"avatarUrl,optional"`
	Nickname  string `json:"nickname,optional"`
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
	Summary        string   `json:"summary,omitempty"`
	SEODescription string   `json:"seoDescription,omitempty"`
	SEOKeywords    string   `json:"seoKeywords,omitempty"`
	RevisedContent string   `json:"revisedContent,omitempty"`
	Suggestions    []string `json:"suggestions,omitempty"`
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
	ID              uint64        `json:"id"`
	AuthorID        uint64        `json:"authorId"`
	CategoryID      uint64        `json:"categoryId,omitempty"`
	Title           string        `json:"title"`
	Slug            string        `json:"slug"`
	Summary         string        `json:"summary"`
	Content         string        `json:"content,omitempty"`
	CoverURL        string        `json:"coverUrl"`
	Status          string        `json:"status"`
	ViewCount       uint64        `json:"viewCount"`
	ScheduledAt     *time.Time    `json:"scheduledAt,omitempty"`
	PublishedAt     *time.Time    `json:"publishedAt,omitempty"`
	IsPinned        bool          `json:"isPinned"`
	DisplayPriority int           `json:"displayPriority"`
	SEOTitle        string        `json:"seoTitle"`
	SEODescription  string        `json:"seoDescription"`
	SEOKeywords     string        `json:"seoKeywords"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	Tags            []TagResp     `json:"tags,omitempty"`
	Category        *CategoryResp `json:"category,omitempty"`
}

type ArticleListResp struct {
	Items []ArticleResp `json:"items"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

type TaxonomyReq struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,optional"`
}

type CategoryResp struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedBy   uint64    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TagResp struct {
	ID          uint64    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedBy   uint64    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
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
	EventType    string    `json:"eventType"`
	ResourceType string    `json:"resourceType"`
	ResourceID   uint64    `json:"resourceId,omitempty"`
	Metadata     string    `json:"metadata,omitempty"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"userAgent"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ListResp[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Size  int   `json:"size"`
}
