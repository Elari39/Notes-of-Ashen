package media

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	medialogic "notes-of-ashen/internal/logic/media"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

var storageKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}\.(jpg|png|gif|webp)$`)

func ListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, size := basehandler.PageSize(r)
		resp, err := medialogic.List(r.Context(), svcCtx, page, size, basehandler.Query(r, "q"))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxBytes := svcCtx.Config.Media.EffectiveMaxUploadBytes()
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("media upload is invalid or too large"))
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := medialogic.Upload(r.Context(), svcCtx, header.Filename, r.FormValue("altText"), data)
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
		var req types.UpdateMediaReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := medialogic.Update(r.Context(), svcCtx, id, req)
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
		if err := medialogic.Delete(r.Context(), svcCtx, id); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func FileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var path struct {
			Key string `path:"key"`
		}
		if err := httpx.ParsePath(r, &path); err != nil || !storageKeyPattern.MatchString(path.Key) || strings.Contains(path.Key, "..") {
			http.NotFound(w, r)
			return
		}
		root, err := medialogic.Root(svcCtx)
		if err != nil {
			http.Error(w, "media unavailable", http.StatusServiceUnavailable)
			return
		}
		filePath := filepath.Join(root, path.Key)
		if _, err := os.Stat(filePath); err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeFile(w, r, filePath)
	}
}
