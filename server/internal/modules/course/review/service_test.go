package review

import "testing"

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		fallback string
		want     string
	}{
		{"空字符串回退", "", "all", "all"},
		{"合法 published", "published", "all", "published"},
		{"合法 hidden", "hidden", "all", "hidden"},
		{"非法值回退", "invalid_status", "all", "all"},
		{"合法 pending", "pending", "published", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateStatus(tt.status, tt.fallback)
			if got != tt.want {
				t.Errorf("validateStatus(%q, %q) = %q, want %q", tt.status, tt.fallback, got, tt.want)
			}
		})
	}
}
