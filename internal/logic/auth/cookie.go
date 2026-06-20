package auth

import (
	"net/http"

	"notes-of-ashen/internal/svc"
)

// refreshTokenCookieName 是存放 refreshToken 的 HttpOnly Cookie 名称。
// 前端无法通过 JS 读取，降低 XSS 窃取长期续签凭证的风险。
const refreshTokenCookieName = "noa_refresh_token"

// SetRefreshCookie 将 refreshToken 写入 HttpOnly+SameSite=Strict Cookie。
// secure 由配置 APP_AUTH_COOKIE_SECURE 决定：生产 HTTPS 为 true，本机 dev 可设 false。
func SetRefreshCookie(w http.ResponseWriter, svcCtx *svc.ServiceContext, token string) {
	maxAge := int(svcCtx.Tokens.RefreshTTL().Seconds())
	if maxAge <= 0 {
		maxAge = 0
	}
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   svcCtx.Config.Auth.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearRefreshCookie 清除 refreshToken Cookie。
func ClearRefreshCookie(w http.ResponseWriter, svcCtx *svc.ServiceContext) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   svcCtx.Config.Auth.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// RefreshTokenFromCookie 优先从 Cookie 读取 refreshToken，缺失返回空串。
// 兼容仍通过 body 传 refreshToken 的 API 客户端：调用方在 cookie 为空时回退 body 值。
func RefreshTokenFromCookie(r *http.Request) string {
	if c, err := r.Cookie(refreshTokenCookieName); err == nil {
		return c.Value
	}
	return ""
}
