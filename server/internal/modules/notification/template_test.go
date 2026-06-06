package notification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewReviewHiddenNotification_IncludesReasonPayload(t *testing.T) {
	params := NewReviewHiddenNotification(7, "review-1", 42, "含有违规内容")

	assert.Equal(t, TypeReviewHidden, params.Type)
	assert.Equal(t, "/courses/42/reviews", params.SourceURL)
	assert.Equal(t, int64(42), params.CourseID)
	assert.Equal(t, "评价已隐藏", params.Title)
	assert.Equal(t, "你的评价已被隐藏：含有违规内容", params.Content)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(params.Payload, &payload))
	assert.Equal(t, "含有违规内容", payload["reason"])
}

func TestNewReportResolvedNotification_EncodesAction(t *testing.T) {
	params := NewReportResolvedNotification(7, "report-1", 42, ReportActionHide)

	assert.Equal(t, TypeReportResolved, params.Type)
	assert.Equal(t, "举报处理结果", params.Title)
	assert.Equal(t, "你提交的举报已处理：相关评价已被隐藏。", params.Content)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(params.Payload, &payload))
	assert.Equal(t, ReportActionHide, payload["action"])
}

func TestNewIdentityRejectedNotification_UsesVerificationPage(t *testing.T) {
	params := NewIdentityRejectedNotification(9, "实名信息不匹配")

	assert.Equal(t, TypeIdentityRejected, params.Type)
	assert.Equal(t, "/user/identity-verification", params.SourceURL)
	assert.Equal(t, "你的实名认证未通过审核：实名信息不匹配", params.Content)
}

func TestFreshmanNotificationTemplatesEncodeReviewState(t *testing.T) {
	expiresAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	approved := NewFreshmanApprovedNotification(9, "app-1", expiresAt)
	rejected := NewFreshmanRejectedNotification(9, "app-1", "材料不清晰")
	nearExpiry := NewFreshmanNearExpiryNotification(9, "cred-1", expiresAt)

	assert.Equal(t, TypeFreshmanApproved, approved.Type)
	assert.Equal(t, "新生认证已通过", approved.Title)
	assert.Equal(t, "/admission", approved.SourceURL)
	assert.Contains(t, approved.Content, "2026-10-01 12:00")
	assert.Equal(t, TypeFreshmanRejected, rejected.Type)
	assert.Equal(t, "你的新生认证未通过审核：材料不清晰", rejected.Content)
	assert.Equal(t, TypeFreshmanNearExpiry, nearExpiry.Type)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(approved.Payload, &payload))
	assert.Equal(t, "app-1", payload["applicationID"])
	assert.Equal(t, "2026-10-01T12:00:00Z", payload["expiresAt"])
}

func TestNotificationTemplateTypesMatchOpenAPIEnum(t *testing.T) {
	openAPITypes := readOpenAPINotificationTypes(t)
	templateTypes := []string{
		TypeReply,
		TypeLike,
		TypeVote,
		TypeReviewHidden,
		TypeReviewRestored,
		TypeReportResolved,
		TypeIdentityApproved,
		TypeIdentityRejected,
		TypeStudentApproved,
		TypeStudentRejected,
		TypeFreshmanApproved,
		TypeFreshmanRejected,
		TypeFreshmanNearExpiry,
		TypeSystem,
	}

	assert.ElementsMatch(t, templateTypes, openAPITypes)
}

func readOpenAPINotificationTypes(t *testing.T) []string {
	t.Helper()

	path := filepath.Join("..", "..", "..", "api", "components", "schemas", "notification.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var schemas struct {
		NotificationType struct {
			Enum []string `yaml:"enum"`
		} `yaml:"NotificationType"`
	}
	require.NoError(t, yaml.Unmarshal(data, &schemas))

	require.NotEmpty(t, schemas.NotificationType.Enum)
	return schemas.NotificationType.Enum
}
