package oidc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestIssuerDialAddress(t *testing.T) {
	tests := []struct {
		name   string
		issuer string
		want   string
	}{
		{name: "empty", issuer: "", want: ""},
		{name: "invalid", issuer: "://bad", want: ""},
		{name: "https default port", issuer: "https://issuer.example.com", want: "issuer.example.com:443"},
		{name: "http default port", issuer: "http://issuer.example.com", want: "issuer.example.com:80"},
		{name: "explicit port", issuer: "https://issuer.example.com:8443", want: "issuer.example.com:8443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, issuerDialAddress(tt.issuer))
		})
	}
}

func TestNewOIDCHTTPClient_RewritesIssuerToInternalAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)

	cfg := config.ZitadelConfig{
		Issuer:          "http://issuer.example.test:" + parsed.Port(),
		InternalAddress: "127.0.0.1",
	}

	client := newOIDCHTTPClient(cfg, "test_oidc")
	require.NotNil(t, client)
	assert.Equal(t, 10*time.Second, client.Timeout)

	resp, err := client.Get(cfg.Issuer)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
