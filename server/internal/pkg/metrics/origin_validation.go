package metrics

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// OriginValidationMiddleware restricts anonymous metrics ingestion to known frontend origins.
// Accept either an Origin header or a Referer whose scheme://host matches the allowlist.
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
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if origin := normalizeOrigin(c.GetHeader("Origin")); origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Next()
				return
			}
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if refererOrigin := originFromReferer(c.GetHeader("Referer")); refererOrigin != "" {
			if _, ok := allowed[refererOrigin]; ok {
				c.Next()
				return
			}
		}

		c.AbortWithStatus(http.StatusForbidden)
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
