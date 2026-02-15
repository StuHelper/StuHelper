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

func TestSanitizeCacheKeyPart(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"正常字符串不变", "2024-spring", "2024-spring"},
		{"编码特殊字符", "term/2024:spring", "term%2F2024%3Aspring"},
		{"空字符串", "", ""},
		{"编码非安全字符保留可区分性", "abc_123-XYZ!@#", "abc_123-XYZ%21%40%23"},
		{"NFC 规范化：NFD 分解形式与 NFC 组合形式产生相同结果", "caf\u0065\u0301", "caf%C3%A9"}, // e + combining accent → é (NFC)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeCacheKeyPart(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeCacheKeyPart(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
