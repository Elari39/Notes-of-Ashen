package backup

import (
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	backuplogic "notes-of-ashen/internal/logic/backup"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func ExportHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.BackupExportReq
		if err := basehandler.Parse(r, &req); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		filePath, err := backuplogic.Export(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		defer os.Remove(filePath)
		file, err := os.Open(filePath)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		defer file.Close()
		filename := "notes-of-ashen-" + time.Now().Format("20060102-150405") + ".noa-backup"
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
		w.Header().Set("Cache-Control", "no-store")
		if _, err := io.Copy(w, file); err != nil {
			return
		}
	}
}

func RestoreHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxBytes := svcCtx.Config.Backup.EffectiveMaxUploadBytes()
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes+1<<20)
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("backup upload is invalid"))
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("backup file is required"))
			return
		}
		defer file.Close()
		resp, err := backuplogic.Restore(r.Context(), svcCtx, r.FormValue("currentPassword"), r.FormValue("passphrase"), r.FormValue("confirmation"), io.LimitReader(file, maxBytes+1))
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
