package admission

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondAdmissionErrorDistinguishesInvalidInputAndForbiddenSource(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid input", err: ErrMemberBlacklistInvalidInput, want: http.StatusBadRequest},
		{name: "forbidden source", err: ErrMemberBlacklistSourceForbidden, want: http.StatusForbidden},
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
