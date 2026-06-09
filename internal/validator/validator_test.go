package validator

import "testing"

func TestOptionalHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty is allowed", value: "", wantErr: false},
		{name: "space is allowed", value: "   ", wantErr: false},
		{name: "https url is allowed", value: "https://example.com/a.png", wantErr: false},
		{name: "http url is allowed", value: "http://example.com/a.png", wantErr: false},
		{name: "account is rejected", value: "Elari39", wantErr: true},
		{name: "anonymous text is rejected", value: "匿名", wantErr: true},
		{name: "missing scheme is rejected", value: "example.com/a.png", wantErr: true},
		{name: "unsupported scheme is rejected", value: "ftp://example.com/a.png", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OptionalHTTPURL(tt.value, "avatarUrl")
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestEmailRequiresPlainAddress(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "plain email", value: "user@example.com", wantErr: false},
		{name: "trimmed plain email", value: " user@example.com ", wantErr: false},
		{name: "display name is rejected", value: "User <user@example.com>", wantErr: true},
		{name: "invalid email is rejected", value: "not-an-email", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
