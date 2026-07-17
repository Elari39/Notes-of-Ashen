package tag

import (
	"strings"
	"testing"

	"notes-of-ashen/internal/types"
)

func TestValidateRejectsOversizedDescription(t *testing.T) {
	req := types.TaxonomyReq{Name: "tag", Slug: "tag", Description: strings.Repeat("你", maxTagDescriptionBytes/3+1)}
	if err := validate(req); err == nil {
		t.Fatal("validate() error = nil, want description size validation error")
	}
}
