package admission

import (
	"context"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/audit"
)

func memberBlacklistCreatedAuditEvent(ctx context.Context, entry *MemberBlacklistEntry) audit.Event {
	return audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("member_blacklist.created"),
		Category:     "admin_operation",
		ActorType:    memberBlacklistAuditActorType(entry.CreatedByType),
		UserID:       entry.CreatedByID,
		ResourceType: "member_blacklist.entry",
		ResourceID:   entry.ID,
		Action:       "create",
		Result:       "success",
		Reason:       string(entry.ReasonCode),
		Details:      memberBlacklistAuditDetails(entry),
	})
}

func memberBlacklistReleasedAuditEvent(ctx context.Context, entry *MemberBlacklistEntry) audit.Event {
	reason := ""
	if entry.ReleaseReasonCode != nil {
		reason = string(*entry.ReleaseReasonCode)
	}
	return audit.EventFromContext(ctx, audit.Event{
		Type:         audit.EventType("member_blacklist.released"),
		Category:     "admin_operation",
		ActorType:    memberBlacklistReleasedAuditActorType(entry),
		UserID:       memberBlacklistReleasedAuditUserID(entry),
		ResourceType: "member_blacklist.entry",
		ResourceID:   entry.ID,
		Action:       "release",
		Result:       "success",
		Reason:       reason,
		Details:      memberBlacklistAuditDetails(entry),
	})
}

func memberBlacklistAuditActorType(actor MemberBlacklistActorType) string {
	switch actor {
	case BlacklistActorSystem:
		return "system"
	case BlacklistActorServiceAccount, BlacklistActorQQOperator:
		return "service_account"
	default:
		return "admin"
	}
}

func memberBlacklistReleasedAuditActorType(entry *MemberBlacklistEntry) string {
	if entry.ReleasedByType == nil {
		return "system"
	}
	return memberBlacklistAuditActorType(*entry.ReleasedByType)
}

func memberBlacklistReleasedAuditUserID(entry *MemberBlacklistEntry) string {
	if entry.ReleasedByID == nil {
		return "system"
	}
	return *entry.ReleasedByID
}

func memberBlacklistAuditDetails(entry *MemberBlacklistEntry) map[string]any {
	return map[string]any{
		"platform":       entry.Platform,
		"subject_type":   entry.SubjectType,
		"subject_id":     entry.SubjectID,
		"scope_type":     entry.ScopeType,
		"guild_id":       entry.GuildID,
		"source":         entry.Source,
		"reason_code":    entry.ReasonCode,
		"reason_text":    entry.ReasonText,
		"created_from":   entry.CreatedFrom,
		"expires_at":     entry.ExpiresAt,
		"released_at":    entry.ReleasedAt,
		"release_code":   entry.ReleaseReasonCode,
		"release_text":   entry.ReleaseReason,
		"entry_metadata": entry.Metadata,
	}
}
