package tag

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"notes-of-ashen/internal/authutil"
	apperrors "notes-of-ashen/internal/errors"
	"notes-of-ashen/internal/svc"
	"notes-of-ashen/internal/types"
)

func TestProtectedOperationsRejectRegularUser(t *testing.T) {
	ctx := authutil.WithUser(context.Background(), 1, authutil.RoleUser)
	svcCtx := &svc.ServiceContext{}
	req := types.TaxonomyReq{Name: "Go", Slug: "go"}

	tests := []struct {
		name string
		call func() error
	}{
		{name: "admin list", call: func() error { _, err := AdminList(ctx, svcCtx, 1, 10); return err }},
		{name: "create", call: func() error { _, err := Create(ctx, svcCtx, req); return err }},
		{name: "update", call: func() error { _, err := Update(ctx, svcCtx, 1, req); return err }},
		{name: "delete", call: func() error { return Delete(ctx, svcCtx, 1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertForbidden(t, tt.call())
		})
	}
}

func TestValidateTaxonomyRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     types.TaxonomyReq
		wantErr bool
	}{
		{name: "trims valid values", req: types.TaxonomyReq{Name: " Go ", Slug: " go "}},
		{name: "accepts maximum lengths", req: types.TaxonomyReq{Name: strings.Repeat("n", 64), Slug: strings.Repeat("s", 96)}},
		{name: "rejects blank name", req: types.TaxonomyReq{Name: "  ", Slug: "go"}, wantErr: true},
		{name: "rejects blank slug", req: types.TaxonomyReq{Name: "Go", Slug: "  "}, wantErr: true},
		{name: "rejects long name", req: types.TaxonomyReq{Name: strings.Repeat("n", 65), Slug: "go"}, wantErr: true},
		{name: "rejects long slug", req: types.TaxonomyReq{Name: "Go", Slug: strings.Repeat("s", 97)}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func assertForbidden(t *testing.T, err error) {
	t.Helper()
	var codeErr *apperrors.CodeError
	if !errors.As(err, &codeErr) {
		t.Fatalf("error = %T %[1]v, want CodeError", err)
	}
	if codeErr.Code != 40300 || codeErr.StatusCode != http.StatusForbidden {
		t.Fatalf("error = code %d status %d, want code 40300 status %d", codeErr.Code, codeErr.StatusCode, http.StatusForbidden)
	}
}
