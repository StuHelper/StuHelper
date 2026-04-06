package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFallbackIntrospectionEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		tokenURL string
		want     string
	}{
		{name: "standard token path", tokenURL: "https://issuer.example.com/oauth/v2/token", want: "https://issuer.example.com/oauth/v2/introspect"},
		{name: "single path", tokenURL: "https://issuer.example.com/token", want: "https://issuer.example.com/introspect"},
		{name: "root path", tokenURL: "https://issuer.example.com", want: "https://issuer.example.com/introspect"},
		{name: "invalid url", tokenURL: "://bad", want: ""},
		{name: "empty", tokenURL: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fallbackIntrospectionEndpoint(tt.tokenURL))
		})
	}
}
