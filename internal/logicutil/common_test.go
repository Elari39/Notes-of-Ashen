package logicutil

import (
	"encoding/json"
	"strings"
	"testing"

	"notes-of-ashen/internal/types"
	"notes-of-ashen/model"
)

func TestRegistrationEmailCodeRequired(t *testing.T) {
	tests := []struct {
		name         string
		isFirstUser  bool
		emailEnabled bool
		want         bool
	}{
		{name: "first user without email service", isFirstUser: true, emailEnabled: false, want: false},
		{name: "first user with email service", isFirstUser: true, emailEnabled: true, want: true},
		{name: "later registration without email service", isFirstUser: false, emailEnabled: false, want: true},
		{name: "later registration with email service", isFirstUser: false, emailEnabled: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RegistrationEmailCodeRequired(tt.isFirstUser, tt.emailEnabled); got != tt.want {
				t.Fatalf("RegistrationEmailCodeRequired(%v, %v) = %v, want %v", tt.isFirstUser, tt.emailEnabled, got, tt.want)
			}
		})
	}
}

// TestEmptySliceSerialization 保证空标签切片序列化为 `[]` 而非被 omitempty 省略，
// 否则前端非可选数组类型会收到 undefined。覆盖 ArticleResp.Tags、
// ArticleVersionResp.TagIDs、ProjectItem.TagIDs 三个字段（P2-2）。
func TestEmptySliceSerialization(t *testing.T) {
	t.Run("ArticleResp.Tags", func(t *testing.T) {
		resp := ArticleResp(model.Article{}, nil, nil, false)
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(b), `"tags":[]`) {
			t.Fatalf("expect tags:[] in %s", b)
		}
	})

	t.Run("ArticleVersionResp.TagIDs", func(t *testing.T) {
		resp := ArticleVersionResp(model.ArticleVersion{TagIDs: nil}, false)
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(b), `"tagIds":[]`) {
			t.Fatalf("expect tagIds:[] in %s", b)
		}
	})

	t.Run("ProjectItem.TagIDs", func(t *testing.T) {
		// model 层（nonNilUint64s/uniqueUint64）已保证运行时为 non-nil 空 slice，
		// 这里用 []uint64{} 模拟该前提，验证移除 omitempty 后输出 [] 而非被省略。
		resp := types.ProjectItem{TagIDs: []uint64{}}
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(b), `"tagIds":[]`) {
			t.Fatalf("expect tagIds:[] in %s", b)
		}
	})
}
