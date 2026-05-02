package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "无敏感参数",
			input:    "page=1&size=10",
			expected: "page=1&size=10",
		},
		{
			name:     "包含 code 和 state 参数",
			input:    "code=abc123&state=xyz",
			expected: "code=%5BREDACTED%5D&state=%5BREDACTED%5D",
		},
		{
			name:     "包含 token 参数",
			input:    "token=secret123",
			expected: "token=%5BREDACTED%5D",
		},
		{
			name:     "包含 access_token 参数",
			input:    "access_token=jwt.token.here",
			expected: "access_token=%5BREDACTED%5D",
		},
		{
			name:     "包含 password 参数",
			input:    "username=admin&password=secret",
			expected: "password=%5BREDACTED%5D&username=admin",
		},
		{
			name:     "大小写不敏感",
			input:    "CODE=abc&Token=xyz&PASSWORD=123",
			expected: "CODE=%5BREDACTED%5D&PASSWORD=%5BREDACTED%5D&Token=%5BREDACTED%5D",
		},
		{
			name:     "多个敏感参数",
			input:    "code=abc&token=xyz&api_key=123&internal_key=secret",
			expected: "api_key=%5BREDACTED%5D&code=%5BREDACTED%5D&internal_key=%5BREDACTED%5D&token=%5BREDACTED%5D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveQueryParams(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskSensitiveQueryParams_InvalidQuery(t *testing.T) {
	// 无效的 query string 应该返回错误标记
	result := maskSensitiveQueryParams("%invalid")
	assert.Equal(t, "[parse_error]", result)
}
