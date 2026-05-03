package admission

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

func auditFreshmanReview(ctx context.Context, app *FreshmanApplication, command freshmanReviewCommand) {
	if app == nil {
		return
	}
	audit.Log(audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("admission.freshman.review"),
		Category:     "admin_operation",
		ActorType:    "user",
		UserID:       reviewActorUserID(command.ReviewerUserID),
		ResourceType: "admission.freshman_application",
		ResourceID:   app.ID,
		Action:       string(command.Action),
		Result:       "success",
		Reason:       stringValue(command.Reason),
		Details: map[string]any{
			"operator_qq_id": stringValue(command.OperatorQQID),
			"guild_id":       command.GuildID,
			"raw_command":    command.RawCommand,
		},
	}))
}

func reviewActorUserID(userID *int64) string {
	if userID == nil {
		return ""
	}
	return fmt.Sprint(*userID)
}
