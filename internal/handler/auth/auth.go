package auth

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	authlogic "notes-of-ashen/internal/logic/auth"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func forwardedOptions(svcCtx *svc.ServiceContext) basehandler.ForwardedOptions {
	return basehandler.ForwardedOptions{TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs}
}

// issueRefreshCookie 将 TokenPair 中的 refreshToken 写入 HttpOnly Cookie，
// 并把响应体内的 refreshToken 置空——omitempty 会省略该字段，长期凭证仅存于 Cookie。
func issueRefreshCookie(w http.ResponseWriter, svcCtx *svc.ServiceContext, resp *types.TokenPair) {
	if resp != nil {
		authlogic.SetRefreshCookie(w, svcCtx, resp.RefreshToken)
		resp.RefreshToken = ""
	}
}

// resolveRefreshToken 优先从 Cookie 读取 refreshToken，缺失时回退请求体（兼容 API 客户端）。
func resolveRefreshToken(r *http.Request, req types.RefreshReq) types.RefreshReq {
	if req.RefreshToken == "" {
		if token := authlogic.RefreshTokenFromCookie(r); token != "" {
			req.RefreshToken = token
		}
	}
	return req
}

func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Register(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		issueRefreshCookie(w, svcCtx, resp)
		response.Ok(w, resp)
	}
}

func CaptchaHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CaptchaReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Captcha(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func SendVerifyCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.SendVerifyCodeReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := authlogic.SendVerifyCode(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Login(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		issueRefreshCookie(w, svcCtx, resp)
		response.Ok(w, resp)
	}
}

func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetPasswordReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := authlogic.ResetPassword(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func RefreshHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		req = resolveRefreshToken(r, req)
		resp, err := authlogic.Refresh(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		issueRefreshCookie(w, svcCtx, resp)
		response.Ok(w, resp)
	}
}

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		req = resolveRefreshToken(r, req)
		if err := authlogic.Logout(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		authlogic.ClearRefreshCookie(w, svcCtx)
		response.NoData(w)
	}
}
