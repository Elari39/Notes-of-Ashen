package validator

import (
	"net"
	"testing"
)

func TestOptionalHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty is allowed", value: "", wantErr: false},
		{name: "space is allowed", value: "   ", wantErr: false},
		{name: "https url is allowed", value: "https://1.1.1.1/a.png", wantErr: false},
		{name: "http url is allowed", value: "http://1.1.1.1/a.png", wantErr: false},
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

func TestIsBlockedHostIP(t *testing.T) {
	for _, value := range []string{
		"127.0.0.1",
		"10.0.0.1",
		"100.64.0.1",
		"169.254.169.254",
		"192.0.2.1",
		"198.18.0.1",
		"203.0.113.1",
		"::1",
		"64:ff9b::10.0.0.1",
		"fc00::1",
		"fe80::1",
		"2001:2::1",
		"2001:db8::1",
		"2002:0a00:0001::1",
	} {
		if !IsBlockedHostIP(net.ParseIP(value)) {
			t.Errorf("IsBlockedHostIP(%q) = false, want true", value)
		}
	}
	for _, value := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if IsBlockedHostIP(net.ParseIP(value)) {
			t.Errorf("IsBlockedHostIP(%q) = true, want false", value)
		}
	}
}
