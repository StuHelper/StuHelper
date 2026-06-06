package metrics

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/response"
)

// OriginValidationMiddleware 只允许已知前端来源匿名上报指标。
// 它接受 Origin 头，或从 Referer 中提取出的 scheme://host 并与白名单匹配。
func OriginValidationMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimRight(strings.TrimSpace(origin), "/")
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}

	return func(c *gin.Context) {
		if len(allowed) == 0 {
			response.Forbidden(c, "forbidden")
			return
		}

		if origin := normalizeOrigin(c.GetHeader("Origin")); origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Next()
				return
			}
			response.Forbidden(c, "forbidden")
			return
		}

		if refererOrigin := originFromReferer(c.GetHeader("Referer")); refererOrigin != "" {
			if _, ok := allowed[refererOrigin]; ok {
				c.Next()
				return
			}
		}

		if isBrowserSameOriginFetch(c.GetHeader("Sec-Fetch-Site")) {
			if _, ok := allowed[requestOrigin(c)]; ok {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "forbidden")
	}
}

func normalizeOrigin(raw string) string {
	return strings.TrimRight(strings.TrimSpace(raw), "/")
}

func originFromReferer(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func isBrowserSameOriginFetch(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "same-origin")
}

func requestOrigin(c *gin.Context) string {
	proto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(c.Request.Host)
	}
	if proto == "" || host == "" {
		return ""
	}
	return strings.TrimRight(proto+"://"+host, "/")
}
