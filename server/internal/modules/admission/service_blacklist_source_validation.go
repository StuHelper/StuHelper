package admission

func memberBlacklistSourceAllowed(
	entryPoint memberBlacklistEntryPoint,
	source MemberBlacklistSource,
	createdFrom MemberBlacklistCreatedFrom,
) bool {
	switch entryPoint {
	case memberBlacklistEntryPointAdmin:
		return source == BlacklistSourceManualAdmin && createdFrom == BlacklistCreatedFromAdminConsole
	case memberBlacklistEntryPointBot:
		return memberBlacklistBotSourceAllowed(source, createdFrom)
	case memberBlacklistEntryPointInternal:
		return source == BlacklistSourceAdmissionFailure && createdFrom == BlacklistCreatedFromAdmissionWorker
	default:
		return false
	}
}

func memberBlacklistBotSourceAllowed(source MemberBlacklistSource, createdFrom MemberBlacklistCreatedFrom) bool {
	switch source {
	case BlacklistSourceManualAdmin:
		return createdFrom == BlacklistCreatedFromKoishiConsole || createdFrom == BlacklistCreatedFromQQCommand
	case BlacklistSourceKickBlacklist:
		return createdFrom == BlacklistCreatedFromQQCommand
	case BlacklistSourceModerationAction:
		return createdFrom == BlacklistCreatedFromModerationReview
	default:
		return false
	}
}

func memberBlacklistCreatedFromValidForSource(
	source MemberBlacklistSource,
	createdFrom MemberBlacklistCreatedFrom,
) bool {
	switch source {
	case BlacklistSourceAdmissionFailure:
		return createdFrom == BlacklistCreatedFromAdmissionWorker
	case BlacklistSourceManualAdmin:
		return createdFrom == BlacklistCreatedFromAdminConsole ||
			createdFrom == BlacklistCreatedFromKoishiConsole ||
			createdFrom == BlacklistCreatedFromQQCommand
	case BlacklistSourceKickBlacklist:
		return createdFrom == BlacklistCreatedFromQQCommand
	case BlacklistSourceModerationAction:
		return createdFrom == BlacklistCreatedFromModerationReview
	case BlacklistSourceMigrationLegacyKoishi, BlacklistSourceMigrationAdmissionFailure:
		return createdFrom == BlacklistCreatedFromMigrationScript
	default:
		return false
	}
}

func memberBlacklistSourceReasonAllowed(
	source MemberBlacklistSource,
	reason MemberBlacklistReasonCode,
) bool {
	switch source {
	case BlacklistSourceAdmissionFailure:
		return reason == BlacklistReasonAdmissionTimeoutLimit
	case BlacklistSourceManualAdmin:
		return reason == BlacklistReasonManualBlacklist
	case BlacklistSourceKickBlacklist:
		return reason == BlacklistReasonManualKickBlacklist
	case BlacklistSourceModerationAction:
		return reason == BlacklistReasonViolationReviewBlacklist
	case BlacklistSourceMigrationLegacyKoishi:
		return reason == BlacklistReasonLegacyKoishiBlacklist
	case BlacklistSourceMigrationAdmissionFailure:
		return reason == BlacklistReasonLegacyAdmissionBlacklist
	default:
		return false
	}
}
