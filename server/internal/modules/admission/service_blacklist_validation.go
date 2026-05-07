package admission

import (
	"strings"
	"time"
)

const (
	defaultMemberBlacklistPageSize = 50
	maxMemberBlacklistPageSize     = 200
)

func normalizeMemberBlacklistCreateInput(input MemberBlacklistCreateInput) MemberBlacklistCreateInput {
	input.Platform = normalizeMemberBlacklistString(input.Platform)
	input.SubjectID = normalizeMemberBlacklistString(input.SubjectID)
	input.ReasonText = normalizeMemberBlacklistString(input.ReasonText)
	input.CreatedByID = normalizeMemberBlacklistString(input.CreatedByID)
	input.GuildID = normalizeStringPtr(input.GuildID)
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	return input
}

func validateMemberBlacklistCreateInput(input MemberBlacklistCreateInput, entryPoint memberBlacklistEntryPoint) error {
	if err := validateMemberBlacklistKey(memberBlacklistCreateKey(input)); err != nil {
		return err
	}
	if input.Source == "" || input.ReasonCode == "" || input.ReasonText == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.CreatedByType == "" || input.CreatedByID == "" || input.CreatedFrom == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistSourceReasonAllowed(input.Source, input.ReasonCode) {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistCreatedFromValidForSource(input.Source, input.CreatedFrom) {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistSourceAllowed(entryPoint, input.Source, input.CreatedFrom) {
		return ErrMemberBlacklistSourceForbidden
	}
	return validateMemberBlacklistMetadata(input)
}

func validateMemberBlacklistCreateTime(input MemberBlacklistCreateInput, now time.Time) error {
	if input.ExpiresAt == nil || input.ExpiresAt.After(now) {
		return nil
	}
	return ErrMemberBlacklistInvalidInput
}

func validateMemberBlacklistMetadata(input MemberBlacklistCreateInput) error {
	switch input.Source {
	case BlacklistSourceAdmissionFailure:
		return validateAdmissionFailureBlacklistMetadata(input.Metadata)
	case BlacklistSourceManualAdmin:
		if err := requireMetadataString(input.Metadata, "operatorInput", "scopeSelectionContext"); err != nil {
			return err
		}
		if input.CreatedFrom == BlacklistCreatedFromQQCommand {
			return requireMetadataString(input.Metadata, "operatorQQID")
		}
		return nil
	case BlacklistSourceKickBlacklist:
		return requireMetadataString(input.Metadata, "rawCommand", "targetGuildID", "operatorQQID")
	case BlacklistSourceModerationAction:
		return requireMetadataString(input.Metadata, "reviewID", "workItemID", "targetGuildID")
	default:
		return nil
	}
}

func validateAdmissionFailureBlacklistMetadata(metadata map[string]any) error {
	if err := requireMetadataString(metadata, "admissionSessionID", "platform", "guildID"); err != nil {
		return err
	}
	return requireMetadataValue(metadata, "failureCount", "failedJoinLimit", "botSelfID")
}

func requireMetadataString(metadata map[string]any, keys ...string) error {
	for _, key := range keys {
		value, ok := metadata[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return ErrMemberBlacklistInvalidInput
		}
	}
	return nil
}

func requireMetadataValue(metadata map[string]any, keys ...string) error {
	for _, key := range keys {
		if value, ok := metadata[key]; !ok || value == nil {
			return ErrMemberBlacklistInvalidInput
		}
	}
	return nil
}

func normalizeMemberBlacklistAccessQuery(query MemberBlacklistAccessQuery) MemberBlacklistAccessQuery {
	query.Platform = normalizeMemberBlacklistString(query.Platform)
	query.SubjectID = normalizeMemberBlacklistString(query.SubjectID)
	query.GuildID = normalizeStringPtr(query.GuildID)
	return query
}

func validateMemberBlacklistAccessQuery(query MemberBlacklistAccessQuery) error {
	return validateMemberBlacklistKey(memberBlacklistKey{
		Platform: query.Platform, SubjectType: query.SubjectType, SubjectID: query.SubjectID,
		ScopeType: BlacklistScopeGlobal,
	})
}

func normalizeMemberBlacklistListFilter(filter MemberBlacklistListFilter) MemberBlacklistListFilter {
	filter.Platform = normalizeMemberBlacklistString(filter.Platform)
	filter.SubjectID = normalizeMemberBlacklistString(filter.SubjectID)
	filter.GuildID = normalizeMemberBlacklistString(filter.GuildID)
	filter.CreatedByID = normalizeMemberBlacklistString(filter.CreatedByID)
	filter.PageSize = normalizeMemberBlacklistPageSize(filter.PageSize)
	if filter.Status == "" {
		filter.Status = BlacklistStatusActive
	}
	return filter
}

func validateMemberBlacklistListFilter(filter MemberBlacklistListFilter) error {
	if filter.SubjectType != "" && filter.SubjectType != BlacklistSubjectQQUser {
		return ErrMemberBlacklistInvalidInput
	}
	if filter.ScopeType != "" && filter.ScopeType != BlacklistScopeGlobal && filter.ScopeType != BlacklistScopeGuild {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistStatusValid(filter.Status) {
		return ErrMemberBlacklistInvalidInput
	}
	return nil
}

func normalizeMemberBlacklistReleaseInput(input MemberBlacklistReleaseInput) MemberBlacklistReleaseInput {
	input.ID = normalizeMemberBlacklistString(input.ID)
	input.ReleasedByID = normalizeMemberBlacklistString(input.ReleasedByID)
	input.ReleaseReason = normalizeStringPtr(input.ReleaseReason)
	return input
}

func validateMemberBlacklistReleaseInput(input MemberBlacklistReleaseInput, entryPoint memberBlacklistEntryPoint) error {
	if input.ID == "" || input.ReleaseReasonCode == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if input.ReleasedByType == "" || input.ReleasedByID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistReleaseReasonAllowed(entryPoint, input.ReleaseReasonCode) {
		return ErrMemberBlacklistInvalidInput
	}
	if !memberBlacklistReleaseActorAllowed(entryPoint, input.ReleasedByType) {
		return ErrMemberBlacklistSourceForbidden
	}
	return nil
}

func normalizeMemberBlacklistReleaseBySubjectInput(
	input MemberBlacklistReleaseBySubjectInput,
) MemberBlacklistReleaseBySubjectInput {
	input.Platform = normalizeMemberBlacklistString(input.Platform)
	input.SubjectID = normalizeMemberBlacklistString(input.SubjectID)
	input.GuildID = normalizeStringPtr(input.GuildID)
	input.ReleasedByID = normalizeMemberBlacklistString(input.ReleasedByID)
	input.ReleaseReason = normalizeStringPtr(input.ReleaseReason)
	return input
}

func validateMemberBlacklistReleaseBySubjectInput(
	input MemberBlacklistReleaseBySubjectInput,
	entryPoint memberBlacklistEntryPoint,
) error {
	key := memberBlacklistKey{
		Platform:    input.Platform,
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		ScopeType:   input.ScopeType,
		GuildID:     input.GuildID,
	}
	if err := validateMemberBlacklistKey(key); err != nil {
		return err
	}
	return validateMemberBlacklistReleaseInput(MemberBlacklistReleaseInput{
		ID: "subject-release", ReleasedByType: input.ReleasedByType,
		ReleasedByID: input.ReleasedByID, ReleaseReasonCode: input.ReleaseReasonCode,
	}, entryPoint)
}

func validateMemberBlacklistKey(key memberBlacklistKey) error {
	if key.Platform == "" || key.SubjectType != BlacklistSubjectQQUser || key.SubjectID == "" {
		return ErrMemberBlacklistInvalidInput
	}
	if key.ScopeType == BlacklistScopeGlobal && key.GuildID == nil {
		return nil
	}
	if key.ScopeType == BlacklistScopeGuild && key.GuildID != nil && strings.TrimSpace(*key.GuildID) != "" {
		return nil
	}
	return ErrMemberBlacklistInvalidInput
}

func memberBlacklistReleaseActorAllowed(
	entryPoint memberBlacklistEntryPoint,
	actor MemberBlacklistActorType,
) bool {
	switch entryPoint {
	case memberBlacklistEntryPointAdmin:
		return actor == BlacklistActorAdminUser
	case memberBlacklistEntryPointBot:
		return actor == BlacklistActorQQOperator || actor == BlacklistActorServiceAccount
	default:
		return actor == BlacklistActorSystem
	}
}

func memberBlacklistReleaseReasonAllowed(
	entryPoint memberBlacklistEntryPoint,
	reason MemberBlacklistReleaseReasonCode,
) bool {
	switch entryPoint {
	case memberBlacklistEntryPointAdmin, memberBlacklistEntryPointBot:
		return memberBlacklistPublicReleaseReason(reason)
	default:
		return memberBlacklistSystemReleaseReason(reason)
	}
}

func memberBlacklistPublicReleaseReason(reason MemberBlacklistReleaseReasonCode) bool {
	switch reason {
	case BlacklistReleaseManualPardon, BlacklistReleaseOnly, BlacklistReleaseAppealPassed:
		return true
	default:
		return false
	}
}

func memberBlacklistSystemReleaseReason(reason MemberBlacklistReleaseReasonCode) bool {
	switch reason {
	case BlacklistReleasePolicyExpiredAuto, BlacklistReleaseMigrationInverse:
		return true
	default:
		return false
	}
}

func memberBlacklistStatusValid(status MemberBlacklistStatus) bool {
	switch status {
	case BlacklistStatusActive, BlacklistStatusReleased, BlacklistStatusExpired, BlacklistStatusAll:
		return true
	default:
		return false
	}
}

func normalizeMemberBlacklistPageSize(value int) int {
	if value <= 0 {
		return defaultMemberBlacklistPageSize
	}
	if value > maxMemberBlacklistPageSize {
		return maxMemberBlacklistPageSize
	}
	return value
}
