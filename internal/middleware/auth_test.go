package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"notes-of-ashen/internal/authutil"
	"notes-of-ashen/model"
)

type fakeUserFinder struct {
	user *model.User
	err  error
}

func (f fakeUserFinder) FindUserByID(context.Context, uint64) (*model.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func (f fakeUserFinder) UserTokenVersion(context.Context, uint64) (uint64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.user.TokenVersion, nil
}

func TestAuthMiddlewareAllowsActiveUserWithLatestRole(t *testing.T) {
	manager := authutil.NewManager("test-secret", 3600, 3600)
	token := mustAccessToken(t, manager, 12, authutil.RoleUser)
	middleware := NewAuthMiddleware(manager, fakeUserFinder{
		user: &model.User{ID: 12, Role: authutil.RoleAdmin, Status: "active"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	middleware.Handle(func(w http.ResponseWriter, r *http.Request) {
		userID, err := authutil.UserID(r.Context())
		if err != nil {
			t.Fatalf("UserID() error = %v", err)
		}
		if userID != 12 {
			t.Fatalf("userID = %d, want 12", userID)
		}
		if got := authutil.Role(r.Context()); got != authutil.RoleAdmin {
			t.Fatalf("role = %q, want %q", got, authutil.RoleAdmin)
		}
		w.WriteHeader(http.StatusNoContent)
	}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
}

func TestAuthMiddlewareRejectsDisabledUser(t *testing.T) {
	manager := authutil.NewManager("test-secret", 3600, 3600)
	token := mustAccessToken(t, manager, 12, authutil.RoleUser)
	middleware := NewAuthMiddleware(manager, fakeUserFinder{
		user: &model.User{ID: 12, Role: authutil.RoleUser, Status: "disabled"},
	})

	rec := serveWithToken(middleware, token)

	assertErrorResponse(t, rec, http.StatusForbidden, 40300)
}

func TestAuthMiddlewareRejectsOldTokenVersion(t *testing.T) {
	manager := authutil.NewManager("test-secret", 3600, 3600)
	token, err := manager.CreateAccessToken(12, authutil.RoleUser, 1)
	if err != nil {
		t.Fatal(err)
	}
	middleware := NewAuthMiddleware(manager, fakeUserFinder{
		user: &model.User{ID: 12, Role: authutil.RoleUser, Status: "active", TokenVersion: 2},
	})
	assertErrorResponse(t, serveWithToken(middleware, token), http.StatusUnauthorized, 40100)
}

func TestAuthMiddlewareRejectsMissingUser(t *testing.T) {
	manager := authutil.NewManager("test-secret", 3600, 3600)
	token := mustAccessToken(t, manager, 12, authutil.RoleUser)
	middleware := NewAuthMiddleware(manager, fakeUserFinder{err: model.ErrNotFound})

	rec := serveWithToken(middleware, token)

	assertErrorResponse(t, rec, http.StatusUnauthorized, 40100)
}

func TestAuthMiddlewareReturnsInternalErrorForLookupFailure(t *testing.T) {
	manager := authutil.NewManager("test-secret", 3600, 3600)
	token := mustAccessToken(t, manager, 12, authutil.RoleUser)
	middleware := NewAuthMiddleware(manager, fakeUserFinder{err: errors.New("database unavailable")})

	rec := serveWithToken(middleware, token)

	assertErrorResponse(t, rec, http.StatusInternalServerError, 50000)
}

func serveWithToken(middleware *AuthMiddleware, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	middleware.Handle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}).ServeHTTP(rec, req)

	return rec
}

func mustAccessToken(t *testing.T, manager *authutil.Manager, userID uint64, role string) string {
	t.Helper()

	token, err := manager.CreateAccessToken(userID, role, 0)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	return token
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, code int) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, status, rec.Body.String())
	}

	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response body: %v", err)
	}
	if body.Code != code {
		t.Fatalf("code = %d, want %d", body.Code, code)
	}
}
