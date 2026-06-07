package article

import "testing"

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
