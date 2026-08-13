package rag

import (
	"testing"

	"notes-of-ashen/model"
)

func TestValidateSettingsOnlyAllowsDefinedHistoryRetentionOptions(t *testing.T) {
	base := DefaultSettings(model.RAGSettings{})
	for _, days := range []int{0, 7, 30, 60, 90, 180, 365} {
		settings := base
		settings.HistoryRetentionDays = days
		if err := ValidateSettings(settings, false); err != nil {
			t.Fatalf("ValidateSettings(%d days) error = %v", days, err)
		}
	}
	for _, days := range []int{-1, 1, 8, 91, 366, 3650} {
		settings := base
		settings.HistoryRetentionDays = days
		if err := ValidateSettings(settings, false); err == nil {
			t.Fatalf("ValidateSettings(%d days) error = nil, want validation failure", days)
		}
	}
}

func TestDefaultSettingsNormalizesInvalidHistoryRetention(t *testing.T) {
	settings := DefaultSettings(model.RAGSettings{HistoryRetentionDays: 1})
	if settings.HistoryRetentionDays != 90 {
		t.Fatalf("HistoryRetentionDays = %d, want 90", settings.HistoryRetentionDays)
	}
}

func TestValidateSettingsRejectsProviderURLCredentialsQueryAndFragment(t *testing.T) {
	base := DefaultSettings(model.RAGSettings{})
	tests := []struct {
		name  string
		field func(*model.RAGSettings)
	}{
		{name: "credentials", field: func(settings *model.RAGSettings) { settings.ChatBaseURL = "https://token@example.com/v1" }},
		{name: "query", field: func(settings *model.RAGSettings) { settings.EmbeddingBaseURL = "https://example.com/v1?apiKey=secret" }},
		{name: "empty query", field: func(settings *model.RAGSettings) { settings.EmbeddingBaseURL = "https://example.com/v1?" }},
		{name: "fragment", field: func(settings *model.RAGSettings) { settings.RerankURL = "https://example.com/rerank#secret" }},
		{name: "empty fragment", field: func(settings *model.RAGSettings) { settings.RerankURL = "https://example.com/rerank#" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := base
			tt.field(&settings)
			if err := ValidateSettings(settings, false); err == nil {
				t.Fatal("ValidateSettings() error = nil, want URL validation failure")
			}
		})
	}
}
