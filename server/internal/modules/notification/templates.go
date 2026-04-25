package notification

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	TypeReply            = "reply"
	TypeLike             = "like"
	TypeVote             = "vote" // legacy compatibility
	TypeReviewHidden     = "review_hidden"
	TypeReviewRestored   = "review_restored"
	TypeReportResolved   = "report_resolved"
	TypeIdentityApproved = "identity_approved"
	TypeIdentityRejected = "identity_rejected"
	TypeStudentApproved  = "student_approved"
	TypeStudentRejected  = "student_rejected"
	TypeSystem           = "system"
)

const (
	ReportActionReject = "reject"
	ReportActionHide   = "hide"
	ReportActionDelete = "delete"
)

func NewReplyNotification(userID int64, reviewID string, courseID int64) SendParams {
	return newSendParams(userID, TypeReply, "你的评价收到了新回复", "有人回复了你对课程的评价", nil, "review", reviewID, reviewSourceURL(courseID), courseID)
}

func NewLikeNotification(userID int64, reviewID string, courseID int64) SendParams {
	return newSendParams(userID, TypeLike, "你的评价获得了一个赞", "有人赞了你的评价", nil, "review", reviewID, reviewSourceURL(courseID), courseID)
}

func NewReviewHiddenNotification(userID int64, reviewID string, courseID int64, reason string) SendParams {
	reason = strings.TrimSpace(reason)
	content := "你的评价已被隐藏。"
	if reason != "" {
		content = "你的评价已被隐藏：" + reason
	}
	return newSendParams(userID, TypeReviewHidden, "评价已隐藏", content, marshalPayload(map[string]string{"reason": reason}), "review", reviewID, reviewSourceURL(courseID), courseID)
}

func NewReviewRestoredNotification(userID int64, reviewID string, courseID int64) SendParams {
	return newSendParams(userID, TypeReviewRestored, "评价已恢复", "你的评价已恢复展示。", nil, "review", reviewID, reviewSourceURL(courseID), courseID)
}

func NewReportResolvedNotification(userID int64, reportID string, courseID int64, action string) SendParams {
	action = strings.TrimSpace(action)
	content := "你提交的举报已处理。"
	switch action {
	case ReportActionReject:
		content = "你提交的举报已处理：管理员未采纳该举报。"
	case ReportActionHide:
		content = "你提交的举报已处理：相关评价已被隐藏。"
	case ReportActionDelete:
		content = "你提交的举报已处理：相关评价已被删除。"
	}
	return newSendParams(userID, TypeReportResolved, "举报处理结果", content, marshalPayload(map[string]string{"action": action}), "review_report", reportID, reviewSourceURL(courseID), courseID)
}

func NewIdentityApprovedNotification(userID int64) SendParams {
	return newSendParams(userID, TypeIdentityApproved, "实名认证已通过", "你的实名认证已通过审核。", nil, "user", fmt.Sprintf("%d", userID), "/user/identity-verification", 0)
}

func NewIdentityRejectedNotification(userID int64, reason string) SendParams {
	reason = strings.TrimSpace(reason)
	content := "你的实名认证未通过审核。"
	if reason != "" {
		content = "你的实名认证未通过审核：" + reason
	}
	return newSendParams(userID, TypeIdentityRejected, "实名认证未通过", content, marshalPayload(map[string]string{"reason": reason}), "user", fmt.Sprintf("%d", userID), "/user/identity-verification", 0)
}

func NewStudentApprovedNotification(userID int64) SendParams {
	return newSendParams(userID, TypeStudentApproved, "学生认证已通过", "你的学生认证已通过审核。", nil, "user", fmt.Sprintf("%d", userID), "/user/student-verification", 0)
}

func NewStudentRejectedNotification(userID int64, reason string) SendParams {
	reason = strings.TrimSpace(reason)
	content := "你的学生认证未通过审核。"
	if reason != "" {
		content = "你的学生认证未通过审核：" + reason
	}
	return newSendParams(userID, TypeStudentRejected, "学生认证未通过", content, marshalPayload(map[string]string{"reason": reason}), "user", fmt.Sprintf("%d", userID), "/user/student-verification", 0)
}

func newSendParams(userID int64, typ, title, content string, payload json.RawMessage, sourceModule, sourceID, sourceURL string, courseID int64) SendParams {
	return SendParams{
		UserID:       userID,
		Type:         typ,
		Title:        title,
		Body:         content,
		Content:      content,
		Payload:      payloadOrEmptyJSON(payload),
		SourceModule: sourceModule,
		SourceID:     sourceID,
		SourceURL:    sourceURL,
		CourseID:     courseID,
	}
}

func reviewSourceURL(courseID int64) string {
	if courseID <= 0 {
		return ""
	}
	return fmt.Sprintf("/courses/%d/reviews", courseID)
}

func marshalPayload(payload map[string]string) json.RawMessage {
	if len(payload) == 0 {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
}
