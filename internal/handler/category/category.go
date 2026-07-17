package category

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	categorylogic "notes-of-ashen/internal/logic/category"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

const maxTaxonomyRequestBodySize = 256 << 10

func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := categorylogic.List(r.Context(), svcCtx, page, size)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func AdminListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := categorylogic.AdminList(r.Context(), svcCtx, page, size)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func CreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TaxonomyReq
		if err := basehandler.ParseLimited(w, r, &req, maxTaxonomyRequestBodySize); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := categorylogic.Create(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		var req types.TaxonomyReq
		if err := basehandler.ParseLimited(w, r, &req, maxTaxonomyRequestBodySize); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := categorylogic.Update(r.Context(), svcCtx, id, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func DeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := categorylogic.Delete(r.Context(), svcCtx, id); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}
