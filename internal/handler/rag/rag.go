package rag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	apperrors "notes-of-ashen/internal/errors"
	basehandler "notes-of-ashen/internal/httphelper"
	raglogic "notes-of-ashen/internal/logic/rag"
	"notes-of-ashen/internal/rag"
	"notes-of-ashen/internal/ragclient"
	"notes-of-ashen/internal/response"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const maxChatRequestBytes = 16 << 10

// errSSEWrite 仅表示客户端连接已无法继续接收事件。不能把底层网络错误向
// 上游或日志传播，避免意外带出请求 URL、代理信息或私有会话内容。
var errSSEWrite = errors.New("rag sse response write failed")

func SettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := raglogic.Settings(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func UpdateSettingsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateRAGSettingsReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.StandardJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := raglogic.UpdateSettings(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func StatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := raglogic.Status(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func TestHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RAGTestReq
		if err := basehandler.ParseLimited(w, r, &req, basehandler.SmallJSONBodyLimit); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := raglogic.Test(r.Context(), svcCtx, req)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func RebuildHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := raglogic.Rebuild(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func SessionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := raglogic.ListSessions(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, map[string]any{"items": resp})
	}
}
func SessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := sessionIDPath(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		resp, err := raglogic.GetSession(r.Context(), svcCtx, id)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.Ok(w, resp)
	}
}
func DeleteSessionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := sessionIDPath(r)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		if err := raglogic.DeleteSession(r.Context(), svcCtx, id); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		response.NoData(w)
	}
}

func StreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RAGChatStreamReq
		if err := basehandler.ParseLimited(w, r, &req, maxChatRequestBytes); err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		req.Question = strings.TrimSpace(req.Question)
		if req.Question == "" || len([]rune(req.Question)) > 4000 {
			response.ErrorCtx(r.Context(), w, apperrors.BadRequest("question is invalid"))
			return
		}
		settings, state, err := raglogic.CheckChatAvailable(r.Context(), svcCtx)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		session, err := raglogic.CreateOrLoadSession(r.Context(), svcCtx, req.SessionID, req.Question, state.Epoch)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		history, err := raglogic.SessionHistory(r.Context(), svcCtx, session)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		evidence, err := raglogic.Retrieve(r.Context(), svcCtx, settings, state, req.Question)
		if err != nil {
			response.ErrorCtx(r.Context(), w, err)
			return
		}
		flusher, ok := setupSSE(w)
		if !ok {
			response.ErrorCtx(r.Context(), w, apperrors.ServiceUnavailable("streaming is unavailable"))
			return
		}
		if session != nil {
			if err := writeEvent(w, flusher, "meta", map[string]string{"sessionId": session.ID}); err != nil {
				return
			}
		}
		sources := make([]raglogic.Source, 0, len(evidence))
		for _, item := range evidence {
			sources = append(sources, item.Source)
		}
		if err := writeEvent(w, flusher, "sources", map[string]any{"sources": sources}); err != nil {
			return
		}
		if len(evidence) == 0 {
			answer := "现有文章中没有足够依据。"
			if err := writeEvent(w, flusher, "delta", map[string]string{"delta": answer}); err != nil {
				return
			}
			messageID, saveErr := raglogic.SaveTurn(r.Context(), svcCtx, session, req.Question, answer, sources)
			if saveErr != nil {
				// 问题和答案均属于私有会话内容；数据库/驱动错误不得被原样记入
				// 日志，以免未来实现或代理在错误中回显参数。
				logx.Error("rag save empty-evidence turn failed")
			}
			done := map[string]string{"messageId": messageID}
			if session != nil {
				done["sessionId"] = session.ID
			}
			_ = writeEvent(w, flusher, "done", done)
			return
		}
		conf, err := rag.ProviderConfig(settings, svcCtx.Config.Auth.AccessSecret)
		if err != nil {
			if writeErr := writeEvent(w, flusher, "error", map[string]string{"message": "rag provider is unavailable"}); writeErr != nil {
				return
			}
			return
		}
		messages := raglogic.BuildMessages(req.Question, evidence, history)
		var answer strings.Builder
		var writeMu sync.Mutex
		err = ragclient.StreamChat(r.Context(), conf, messages, func(delta string) error {
			writeMu.Lock()
			defer writeMu.Unlock()
			if err := writeEvent(w, flusher, "delta", map[string]string{"delta": delta}); err != nil {
				return err
			}
			answer.WriteString(delta)
			return nil
		})
		if err != nil {
			if errors.Is(err, errSSEWrite) {
				// 浏览器停止生成或连接断开时，立即取消上游请求且不保存局部回答。
				return
			}
			logx.Errorf("rag stream upstream failed: %s", ragclient.SafeErrorSummary(err))
			if writeErr := writeEvent(w, flusher, "error", map[string]string{"message": "rag provider request failed"}); writeErr != nil {
				return
			}
			return
		}
		messageID, saveErr := raglogic.SaveTurn(context.Background(), svcCtx, session, req.Question, answer.String(), sources)
		if saveErr != nil {
			logx.Error("rag save chat turn failed")
		}
		done := map[string]string{"messageId": messageID}
		if session != nil {
			done["sessionId"] = session.ID
		}
		_ = writeEvent(w, flusher, "done", done)
	}
}

// setupSSE 只在确认 ResponseWriter 支持 Flush 后写入 SSE 响应头。否则错误
// 响应仍应保持标准 JSON Content-Type，避免中间件或客户端把 503 误解析为事件流。
func setupSSE(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return flusher, true
}
func writeEvent(w http.ResponseWriter, f http.Flusher, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw); err != nil {
		return errSSEWrite
	}
	f.Flush()
	return nil
}
func sessionIDPath(r *http.Request) (string, error) {
	var path struct {
		ID string `path:"id"`
	}
	if err := httpx.ParsePath(r, &path); err != nil || strings.TrimSpace(path.ID) == "" {
		return "", apperrors.BadRequest("invalid session id")
	}
	return strings.TrimSpace(path.ID), nil
}
