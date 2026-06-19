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

func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Register(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func CaptchaHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CaptchaReq
		if err := basehandler.Parse(r, &req); err != nil {
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
		if err := basehandler.Parse(r, &req); err != nil {
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
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Login(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ResetPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetPasswordReq
		if err := basehandler.Parse(r, &req); err != nil {
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
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := authlogic.Refresh(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := authlogic.Logout(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}
