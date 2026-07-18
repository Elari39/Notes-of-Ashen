package user

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	userlogic "notes-of-ashen/internal/logic/user"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func forwardedOptions(svcCtx *svc.ServiceContext) basehandler.ForwardedOptions {
	return basehandler.ForwardedOptions{TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs}
}

func MeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := userlogic.Me(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateMeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateMeReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := userlogic.UpdateMe(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func SendVerifyCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UserVerifyCodeReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := userlogic.SendVerifyCode(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func ChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChangePasswordReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := userlogic.ChangePassword(r.Context(), svcCtx, req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}
