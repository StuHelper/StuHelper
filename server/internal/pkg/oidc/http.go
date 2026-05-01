package oidc

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/observability"
)

func newOIDCHTTPClient(cfg config.CasdoorConfig, clientName string) *http.Client {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   5,
	}

	internalAddr := strings.TrimSpace(cfg.InternalAddress)
	issuerAddr := issuerDialAddress(cfg.Issuer)
	if internalAddr != "" && issuerAddr != "" {
		if !strings.Contains(internalAddr, ":") {
			if _, port, err := net.SplitHostPort(issuerAddr); err == nil && port != "" {
				internalAddr = net.JoinHostPort(internalAddr, port)
			}
		}

		dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			if address == issuerAddr {
				address = internalAddr
			}
			return dialer.DialContext(ctx, network, address)
		}
	}

	return observability.WrapHTTPClient(&http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
	}, clientName)
}

func issuerDialAddress(issuer string) string {
	if issuer == "" {
		return ""
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(u.Host); err == nil {
		return u.Host
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return net.JoinHostPort(u.Host, "443")
	default:
		return net.JoinHostPort(u.Host, "80")
	}
}
