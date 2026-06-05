package httputil

import "github.com/gin-gonic/gin"

// RouteHandlers returns gin handlers with nil middlewares skipped and handler last.
func RouteHandlers(handler gin.HandlerFunc, middlewares ...gin.HandlerFunc) []gin.HandlerFunc {
	handlers := make([]gin.HandlerFunc, 0, len(middlewares)+1)
	handlers = AppendRouteMiddlewares(handlers, middlewares...)
	handlers = append(handlers, handler)
	return handlers
}

// AppendRouteMiddlewares appends non-nil middlewares while preserving order.
func AppendRouteMiddlewares(existing []gin.HandlerFunc, middlewares ...gin.HandlerFunc) []gin.HandlerFunc {
	for _, mw := range middlewares {
		if mw != nil {
			existing = append(existing, mw)
		}
	}
	return existing
}
