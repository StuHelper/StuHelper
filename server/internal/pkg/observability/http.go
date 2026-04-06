package observability

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHTTPClient 为外部 HTTP 客户端增加 trace 传播与 span 采集。
func WrapHTTPClient(client *http.Client, name string) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	wrapped := *client
	wrapped.Transport = otelhttp.NewTransport(baseTransport(client.Transport), otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		if r == nil || r.URL == nil {
			return name
		}
		return name + " " + r.Method + " " + r.URL.Host
	}))
	return &wrapped
}

func baseTransport(rt http.RoundTripper) http.RoundTripper {
	if rt != nil {
		return rt
	}
	return http.DefaultTransport
}
