package admin

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	adminlogic "notes-of-ashen/internal/logic/admin"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := adminlogic.ListUsers(r.Context(), svcCtx, page, size)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateUserStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		var req types.UserStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		if err := adminlogic.UpdateUserStatus(r.Context(), svcCtx, id, req); err != nil {
			response.Error(w, err)
			return
		}
		response.NoData(w)
	}
}

func ListLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := adminlogic.ListLogs(r.Context(), svcCtx, page, size)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}
