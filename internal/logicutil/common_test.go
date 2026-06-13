package logicutil

import "testing"

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
