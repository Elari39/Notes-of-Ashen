package article

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	articlelogic "notes-of-ashen/internal/logic/article"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

func forwardedOptions(svcCtx *svc.ServiceContext) basehandler.ForwardedOptions {
	return basehandler.ForwardedOptions{TrustedProxyCIDRs: svcCtx.Config.Proxy.TrustedCIDRs}
}

const (
	maxMarkdownUploadBytes     = 2 << 20
	maxArticleRequestBodySize  = 6 << 20
	maxAIAssistRequestBodySize = 128 << 10
)

func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := listReq(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.ListByFilter(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func AdminListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := listReq(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.AdminList(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
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
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Detail(r.Context(), svcCtx, id, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func PreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Preview(r.Context(), svcCtx, id)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ContextHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Context(r.Context(), svcCtx, id)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func LikeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Like(r.Context(), svcCtx, id, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func CreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ArticleReq
		if err := basehandler.ParseLimited(w, r, &req, maxArticleRequestBodySize); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Create(r.Context(), svcCtx, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
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
		var req types.ArticleReq
		if err := basehandler.ParseLimited(w, r, &req, maxArticleRequestBodySize); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.Update(r.Context(), svcCtx, id, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
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
		if err := articlelogic.Delete(r.Context(), svcCtx, id, basehandler.Meta(r, forwardedOptions(svcCtx))); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func ListVersionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		page, size := basehandler.PageSize(r)
		resp, err := articlelogic.ListVersions(r.Context(), svcCtx, id, page, size)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func VersionDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		versionNo, err := basehandler.PathVersionNo(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.VersionDetail(r.Context(), svcCtx, id, versionNo)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func RestoreVersionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		versionNo, err := basehandler.PathVersionNo(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.RestoreVersion(r.Context(), svcCtx, id, versionNo, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UpdateStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		var req types.ArticleStatusReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.UpdateStatus(r.Context(), svcCtx, id, req, basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func AIAssistHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AIAssistReq
		if err := basehandler.ParseLimited(w, r, &req, maxAIAssistRequestBodySize); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := articlelogic.AIAssist(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ImportMarkdownHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxMarkdownUploadBytes+1024)
		if err := r.ParseMultipartForm(maxMarkdownUploadBytes); err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("markdown file is invalid"))
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("markdown file is required"))
			return
		}
		defer file.Close()
		filename := strings.TrimSpace(header.Filename)
		if strings.ToLower(filepath.Ext(filename)) != ".md" {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("markdown file must be .md"))
			return
		}
		raw, err := io.ReadAll(io.LimitReader(file, maxMarkdownUploadBytes+1))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if len(raw) > maxMarkdownUploadBytes {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("markdown file is too large"))
			return
		}
		resp, err := articlelogic.ImportMarkdown(r.Context(), svcCtx, filename, string(raw), basehandler.Meta(r, forwardedOptions(svcCtx)))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func ExportMarkdownHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := basehandler.PathID(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		filename, content, err := articlelogic.ExportMarkdown(r.Context(), svcCtx, id)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		if _, err := w.Write([]byte(content)); err != nil {
			logx.Errorf("write markdown export failed: %v", err)
		}
	}
}
