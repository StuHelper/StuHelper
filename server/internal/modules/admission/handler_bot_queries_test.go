package admission

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotPendingActionFilterRejectsOverlongValues(t *testing.T) {
	c, recorder := newBotPendingActionFilterContext("platform=" + strings.Repeat("x", maxBotPendingActionFilterLength+1))

	filter, ok := botPendingActionFilter(c)

	require.False(t, ok)
	assert.Equal(t, AdmissionPendingActionFilter{}, filter)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestBotPendingActionFilterRequiresBotIdentity(t *testing.T) {
	c, recorder := newBotPendingActionFilterContext("platform=qq")

	filter, ok := botPendingActionFilter(c)

	require.False(t, ok)
	assert.Equal(t, AdmissionPendingActionFilter{}, filter)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestBotPendingActionFilterTrimsValues(t *testing.T) {
	const pendingActionLimit = 2
	c, _ := newBotPendingActionFilterContext("platform=%20qq%20&botSelfID=%20bot-1%20&limit=2")

	filter, ok := botPendingActionFilter(c)

	require.True(t, ok)
	assert.Equal(t, AdmissionPendingActionFilter{Platform: "qq", BotSelfID: "bot-1", Limit: pendingActionLimit}, filter)
}

func newBotPendingActionFilterContext(rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/v1/bot/admission/sessions/pending?"+rawQuery, nil)
	return c, recorder
}
