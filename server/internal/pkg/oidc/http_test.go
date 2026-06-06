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
		{name: "ipv6 default port", issuer: "https://[::1]", want: "[::1]:443"},
		{name: "ipv6 explicit port", issuer: "http://[fd00::5]:8080", want: "[fd00::5]:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, issuerDialAddress(tt.issuer))
		})
	}
}

func TestInternalDialAddress(t *testing.T) {
	tests := []struct {
		name         string
		internalAddr string
		issuerAddr   string
		want         string
	}{
		{name: "empty", internalAddr: "", issuerAddr: "issuer.example.com:443", want: ""},
		{name: "hostname inherits issuer port", internalAddr: "casdoor", issuerAddr: "issuer.example.com:8443", want: "casdoor:8443"},
		{name: "ipv4 inherits issuer port", internalAddr: "127.0.0.1", issuerAddr: "issuer.example.com:8080", want: "127.0.0.1:8080"},
		{name: "bare ipv6 inherits issuer port", internalAddr: "::1", issuerAddr: "issuer.example.com:8080", want: "[::1]:8080"},
		{name: "bracketed ipv6 inherits issuer port", internalAddr: "[fd00::5]", issuerAddr: "issuer.example.com:9443", want: "[fd00::5]:9443"},
		{name: "explicit hostname port is preserved", internalAddr: "casdoor:18085", issuerAddr: "issuer.example.com:8443", want: "casdoor:18085"},
		{name: "explicit ipv6 port is preserved", internalAddr: "[::1]:18085", issuerAddr: "issuer.example.com:8443", want: "[::1]:18085"},
		{name: "issuer without port keeps original", internalAddr: "casdoor", issuerAddr: "issuer.example.com", want: "casdoor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, internalDialAddress(tt.internalAddr, tt.issuerAddr))
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

	cfg := config.CasdoorConfig{
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
