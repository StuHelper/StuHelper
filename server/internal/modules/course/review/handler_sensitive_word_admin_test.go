package review

import "testing"

func TestValidateSensitiveWordCategory(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty allowed", value: "", wantErr: false},
		{name: "valid slug", value: "violence", wantErr: false},
		{name: "valid with dash", value: "politics_high", wantErr: false},
		{name: "starts with digit", value: "1bad", wantErr: true},
		{name: "contains space", value: "bad value", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSensitiveWordCategory(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSensitiveWordCategory(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSensitiveWordLevel(t *testing.T) {
	for _, level := range []string{"", "block", "warn", "review"} {
		if err := validateSensitiveWordLevel(level); err != nil {
			t.Fatalf("validateSensitiveWordLevel(%q) unexpected error: %v", level, err)
		}
	}
	if err := validateSensitiveWordLevel("critical"); err == nil {
		t.Fatal("expected invalid level to fail")
	}
}
