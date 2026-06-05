package httputil

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRouteHandlersSkipsNilMiddlewaresAndKeepsOrder(t *testing.T) {
	var calls []string
	first := namedHandler("first", &calls)
	second := namedHandler("second", &calls)
	handler := namedHandler("handler", &calls)

	handlers := RouteHandlers(handler, first, nil, second)

	assert.Len(t, handlers, 3)
	for _, h := range handlers {
		h(&gin.Context{})
	}
	assert.Equal(t, []string{"first", "second", "handler"}, calls)
}

func TestAppendRouteMiddlewaresPreservesExistingHandlers(t *testing.T) {
	var calls []string
	existing := []gin.HandlerFunc{namedHandler("existing", &calls)}

	handlers := AppendRouteMiddlewares(existing, nil, namedHandler("added", &calls))

	assert.Len(t, handlers, 2)
	for _, h := range handlers {
		h(&gin.Context{})
	}
	assert.Equal(t, []string{"existing", "added"}, calls)
}

func namedHandler(name string, calls *[]string) gin.HandlerFunc {
	return func(*gin.Context) {
		*calls = append(*calls, name)
	}
}
