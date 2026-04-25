package review

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestSensitiveWordHandlers_InvalidAndNotFoundPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	h := newReviewAdminHandler(t, svc)

	// invalid category on create
	w, c := withAdminContext(http.MethodPost, "/admin/sensitive-words", `{"word":"测试敏感词","category":"Invalid Category","level":"warn"}`)
	h.CreateSensitiveWord(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "category must match")

	// invalid level on update
	w, c = withAdminContext(http.MethodPut, "/admin/sensitive-words/missing-id", `{"level":"fatal"}`)
	c.Params = gin.Params{{Key: "sensitiveWordID", Value: "missing-id"}}
	h.UpdateSensitiveWord(c)
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "level must be one of")

	// missing row on update/delete
	w, c = withAdminContext(http.MethodPut, "/admin/sensitive-words/missing-id", `{"level":"warn"}`)
	c.Params = gin.Params{{Key: "sensitiveWordID", Value: "missing-id"}}
	h.UpdateSensitiveWord(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "sensitive word not found")

	w, c = withAdminContext(http.MethodDelete, "/admin/sensitive-words/missing-id", "")
	c.Params = gin.Params{{Key: "sensitiveWordID", Value: "missing-id"}}
	h.DeleteSensitiveWord(c)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "sensitive word not found")
}
