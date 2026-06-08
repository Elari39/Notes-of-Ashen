package article

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
)

func TestValidateDisplayPriority(t *testing.T) {
	tests := []struct {
		name    string
		value   *int
		wantErr bool
	}{
		{name: "nil keeps default", value: nil, wantErr: false},
		{name: "zero is valid", value: intPtr(0), wantErr: false},
		{name: "max is valid", value: intPtr(9999), wantErr: false},
		{name: "negative is invalid", value: intPtr(-1), wantErr: true},
		{name: "over max is invalid", value: intPtr(10000), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDisplayPriority(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateDisplayPriority() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequireArticleCreatePermission(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		wantStatus int
	}{
		{name: "user cannot create article", role: authutil.RoleUser, wantStatus: http.StatusForbidden},
		{name: "editor can create article", role: authutil.RoleEditor},
		{name: "admin can create article", role: authutil.RoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := authutil.WithUser(context.Background(), 1, tt.role)
			err := requireArticleCreatePermission(ctx)
			if tt.wantStatus == 0 {
				if err != nil {
					t.Fatalf("requireArticleCreatePermission() error = %v, want nil", err)
				}
				return
			}

			var codeErr *apperrors.CodeError
			if !errors.As(err, &codeErr) {
				t.Fatalf("requireArticleCreatePermission() error = %T %[1]v, want CodeError", err)
			}
			if codeErr.StatusCode != tt.wantStatus || codeErr.Code != 40300 {
				t.Fatalf("error = code %d status %d, want code 40300 status %d", codeErr.Code, codeErr.StatusCode, tt.wantStatus)
			}
		})
	}
}
