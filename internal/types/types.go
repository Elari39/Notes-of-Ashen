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
	Nickname  string `json:"nickname,optional"`
	AvatarURL string `json:"avatarUrl,optional"`
}

type LoginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type RefreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

type RequestMeta struct {
	IP        string
	UserAgent string
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
	AvatarURL string `json:"avatarUrl,optional"`
	Nickname  string `json:"nickname,optional"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type ArticleReq struct {
	CategoryID uint64   `json:"categoryId,optional"`
	Title      string   `json:"title"`
	Slug       string   `json:"slug"`
	Summary    string   `json:"summary,optional"`
	Content    string   `json:"content"`
	CoverURL   string   `json:"coverUrl,optional"`
	Status     string   `json:"status,optional"`
	TagIDs     []uint64 `json:"tagIds,optional"`
}

type ArticleStatusReq struct {
	Status string `json:"status"`
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
	ID          uint64        `json:"id"`
	AuthorID    uint64        `json:"authorId"`
	CategoryID  uint64        `json:"categoryId,omitempty"`
	Title       string        `json:"title"`
	Slug        string        `json:"slug"`
	Summary     string        `json:"summary"`
	Content     string        `json:"content,omitempty"`
	CoverURL    string        `json:"coverUrl"`
	Status      string        `json:"status"`
	ViewCount   uint64        `json:"viewCount"`
	PublishedAt *time.Time    `json:"publishedAt,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	Tags        []TagResp     `json:"tags,omitempty"`
	Category    *CategoryResp `json:"category,omitempty"`
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
