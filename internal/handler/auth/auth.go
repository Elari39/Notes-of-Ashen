package auth

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	authlogic "notes-of-ashen/internal/logic/auth"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func RegisterHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := authlogic.Register(r.Context(), svcCtx, req, basehandler.Meta(r))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := authlogic.Login(r.Context(), svcCtx, req, basehandler.Meta(r))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func RefreshHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := authlogic.Refresh(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		if err := authlogic.Logout(r.Context(), svcCtx, req, basehandler.Meta(r)); err != nil {
			response.Error(w, err)
			return
		}
		response.NoData(w)
	}
}
