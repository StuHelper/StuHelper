package admission

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewHandlerRequiresDependencies(t *testing.T) {
	assert.PanicsWithValue(t, "admission.NewHandler: service must not be nil", func() {
		NewHandler(nil, nil, nil)
	})
	assert.PanicsWithValue(t, "admission.NewHandler: internal user id resolver must not be nil", func() {
		NewHandler(&Service{}, nil, nil)
	})

	resolver := func(context.Context, string) (int64, error) {
		return 0, nil
	}
	assert.NotNil(t, NewHandler(&Service{}, resolver, nil))
}

func TestRespondAdmissionErrorDistinguishesInvalidInputAndForbiddenSource(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: ErrMemberBlacklistInvalidInput, want: http.StatusBadRequest},
		{name: "forbidden source", err: ErrMemberBlacklistSourceForbidden, want: http.StatusForbidden},
		{name: "redis unavailable", err: ErrAdmissionRedisUnavailable, want: http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, recorder := newAdmissionErrorTestContext(tt.err)

			assert.Equal(t, tt.want, recorder.Code)
		})
	}
}

func newAdmissionErrorTestContext(err error) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	respondAdmissionError(c, err)
	return c, recorder
}
