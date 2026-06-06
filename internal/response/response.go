package response

import (
	"errors"
	"net/http"

	apperrors "notes-of-ashen/internal/errors"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Ok(w http.ResponseWriter, data interface{}) {
	httpx.OkJson(w, Body{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func NoData(w http.ResponseWriter) {
	httpx.OkJson(w, Body{
		Code:    0,
		Message: "success",
	})
}

func Error(w http.ResponseWriter, err error) {
	var codeErr *apperrors.CodeError
	if errors.As(err, &codeErr) {
		httpx.WriteJson(w, codeErr.StatusCode, Body{
			Code:    codeErr.Code,
			Message: codeErr.Message,
		})
		return
	}

	httpx.WriteJson(w, http.StatusInternalServerError, Body{
		Code:    50000,
		Message: "internal server error",
	})
}
