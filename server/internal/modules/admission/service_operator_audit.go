package admission

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

func freshmanReviewAuditEvent(
	ctx context.Context,
	app *FreshmanApplication,
	command freshmanReviewCommand,
) audit.Event {
	return audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("admission.freshman.review"),
		Category:     "admin_operation",
		ActorType:    "user",
		UserID:       reviewActorUserID(command.ReviewerUserID),
		ResourceType: "admission.freshman_application",
		ResourceID:   app.ID,
		Action:       string(command.Action),
		Result:       "success",
		Reason:       stringValue(command.Reason),
		Details:      freshmanReviewAuditDetails(command),
	})
}

func joinRequestAuditEvent(ctx context.Context, input AdmissionJoinRequestEventInput) audit.Event {
	result := "failure"
	if input.Success {
		result = "success"
	}
	return audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("admission.join_request"),
		Category:     "domain_event",
		ActorType:    "system",
		ResourceType: "admission.join_request",
		ResourceID:   input.RequestID,
		Action:       string(input.Decision),
		Result:       result,
		Reason:       input.Error,
		Details: map[string]any{
			"platform":  input.Platform,
			"guild_id":  input.GuildID,
			"qq_id":     input.QQID,
			"raw_event": input.RawEvent,
		},
	})
}

func freshmanReviewAuditDetails(command freshmanReviewCommand) map[string]any {
	return map[string]any{
		"operator_qq_id":  stringValue(command.OperatorQQID),
		"guild_id":        command.GuildID,
		"raw_command":     command.RawCommand,
		"expires_in_days": command.ExpiresInDays,
	}
}

func reviewActorUserID(userID *int64) string {
	if userID == nil {
		return ""
	}
	return fmt.Sprint(*userID)
}
