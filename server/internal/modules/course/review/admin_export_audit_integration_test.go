package review

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	auditpkg "github.com/StuHelper/StuHelper/server/internal/pkg/audit"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestExportReviews_PersistsOneAuditEventForSuccessAndFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(fixture.DB, repo, noopNotificationSender{}, noopReviewFGAWriter{}, failClosedReviewAccessReader{})
	h := &Handler{service: svc}
	ctx := context.Background()

	auditpkg.ConfigureRepository(auditpkg.NewRepository(fixture.DB))
	t.Cleanup(func() { auditpkg.ConfigureRepository(nil) })

	departmentID := seedDepartment(t, fixture, 4111010006, "导出审计学院")
	teacherID := seedTeacher(t, fixture, 4111010006, "导出审计教师", departmentID)
	courseID := seedCourse(t, fixture, 4111010006, departmentID, "导出审计课程")
	reviewID := "550e8400-e29b-41d4-a716-446655440901"
	seedReviewWithRatings(
		t,
		fixture,
		reviewID,
		courseID,
		teacherID,
		"export-audit-user",
		4.5,
		StatusPublished,
		ReviewRatings{"teaching": 5},
		"导出审计标题",
		"导出审计内容",
	)

	tests := []struct {
		name         string
		format       string
		cancelStream bool
		wantResult   string
		wantRows     int
	}{
		{name: "ndjson success", format: "ndjson", wantResult: "success", wantRows: 1},
		{name: "csv success", format: "csv", wantResult: "success", wantRows: 1},
		{name: "ndjson canceled stream", format: "ndjson", cancelStream: true, wantResult: "failure"},
		{name: "csv canceled stream", format: "csv", cancelStream: true, wantResult: "failure"},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requestID := "review-export-audit-" + test.format + "-" + test.wantResult
			recorder, c := withAdminContext(
				http.MethodGet,
				"/admin/export?format="+test.format+"&status=published",
				"",
			)
			c.Set(middleware.CtxKeyRequestID, requestID)

			if test.cancelStream {
				requestContext, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(requestContext)
			}

			h.ExportReviews(c)

			if test.wantResult == "success" {
				assert.Contains(t, recorder.Body.String(), reviewID)
				assert.Contains(t, recorder.Body.String(), "# EXPORT_COMPLETE")
			} else {
				assert.NotContains(t, recorder.Body.String(), "# EXPORT_COMPLETE")
			}

			var eventCount int
			require.NoError(t, fixture.Pool.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM audit_events
				WHERE request_id = $1
			`, requestID).Scan(&eventCount))
			require.Equal(t, 1, eventCount)

			var (
				eventType     string
				category      string
				actorType     string
				actorUserID   string
				actorUsername string
				action        string
				resourceType  string
				resourceID    string
				result        string
				reason        string
				detailsJSON   []byte
			)
			require.NoError(t, fixture.Pool.QueryRow(ctx, `
				SELECT event_type, category, actor_type, actor_user_id, actor_username,
				       action, resource_type, resource_id, result, COALESCE(reason, ''), details
				FROM audit_events
				WHERE request_id = $1
			`, requestID).Scan(
				&eventType,
				&category,
				&actorType,
				&actorUserID,
				&actorUsername,
				&action,
				&resourceType,
				&resourceID,
				&result,
				&reason,
				&detailsJSON,
			))

			assert.Equal(t, string(auditpkg.EventDataExport), eventType)
			assert.Equal(t, "admin_operation", category)
			assert.Equal(t, "admin", actorType)
			assert.Equal(t, "admin-user-1", actorUserID)
			assert.Equal(t, "admin-root", actorUsername)
			assert.Equal(t, "export", action)
			assert.Equal(t, "review", resourceType)
			assert.Equal(t, "bulk", resourceID)
			assert.Equal(t, test.wantResult, result)
			if test.wantResult == "failure" {
				assert.NotEmpty(t, reason)
			} else {
				assert.Empty(t, reason)
			}

			var details struct {
				Format   string `json:"format"`
				Status   string `json:"status"`
				RowCount int    `json:"row_count"`
				RowLimit int    `json:"row_limit"`
			}
			require.NoError(t, json.Unmarshal(detailsJSON, &details))
			assert.Equal(t, test.format, details.Format)
			assert.Equal(t, StatusPublished, details.Status)
			assert.Equal(t, test.wantRows, details.RowCount)
			assert.Equal(t, maxExportLimit, details.RowLimit)

			assert.Equal(t, index+1, countReviewExportAuditEvents(t, fixture, ctx))
		})
	}
}

func countReviewExportAuditEvents(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	ctx context.Context,
) int {
	t.Helper()
	var count int
	require.NoError(t, fixture.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM audit_events
		WHERE event_type = $1 AND resource_type = 'review' AND action = 'export'
	`, auditpkg.EventDataExport).Scan(&count))
	return count
}
