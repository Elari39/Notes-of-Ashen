package user

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	userlogic "notes-of-ashen/internal/logic/user"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func MeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := userlogic.Me(r.Context(), svcCtx)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateMeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateMeReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := userlogic.UpdateMe(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func SendVerifyCodeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UserVerifyCodeReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		if err := userlogic.SendVerifyCode(r.Context(), svcCtx, req, basehandler.Meta(r)); err != nil {
			response.Error(w, err)
			return
		}
		response.NoData(w)
	}
}

func ChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChangePasswordReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		if err := userlogic.ChangePassword(r.Context(), svcCtx, req); err != nil {
			response.Error(w, err)
			return
		}
		response.NoData(w)
	}
}
