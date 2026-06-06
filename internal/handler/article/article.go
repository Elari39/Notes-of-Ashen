package article

import (
	"net/http"

	basehandler "notes-of-ashen/internal/httphelper"
	articlelogic "notes-of-ashen/internal/logic/article"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := listReq(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.ListByFilter(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func AdminListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := listReq(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.AdminList(r.Context(), svcCtx, req)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func listReq(r *http.Request) (types.ArticleListReq, error) {
	page, size := basehandler.PageSize(r)
	categoryID, err := basehandler.QueryUint64(r, "categoryId")
	if err != nil {
		return types.ArticleListReq{}, err
	}
	tagID, err := basehandler.QueryUint64(r, "tagId")
	if err != nil {
		return types.ArticleListReq{}, err
	}
	return types.ArticleListReq{
		Page:       page,
		Size:       size,
		Status:     basehandler.Query(r, "status"),
		Query:      basehandler.Query(r, "q"),
		CategoryID: categoryID,
		TagID:      tagID,
	}, nil
}

func DetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.Detail(r.Context(), svcCtx, id)
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func CreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ArticleReq
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.Create(r.Context(), svcCtx, req, basehandler.Meta(r))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		var req types.ArticleReq
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.Update(r.Context(), svcCtx, id, req, basehandler.Meta(r))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func DeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		if err := articlelogic.Delete(r.Context(), svcCtx, id, basehandler.Meta(r)); err != nil {
			response.Error(w, err)
			return
		}
		response.NoData(w)
	}
}

func UpdateStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.Error(w, err)
			return
		}
		var req types.ArticleStatusReq
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}
		resp, err := articlelogic.UpdateStatus(r.Context(), svcCtx, id, req, basehandler.Meta(r))
		if err != nil {
			response.Error(w, err)
			return
		}
		response.Ok(w, resp)
	}
}
