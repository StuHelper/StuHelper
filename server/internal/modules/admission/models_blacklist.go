package admission

import "time"

type MemberBlacklistSource string

const (
	BlacklistSourceAdmissionFailure          MemberBlacklistSource = "admission_failure"
	BlacklistSourceManualAdmin               MemberBlacklistSource = "manual_admin"
	BlacklistSourceKickBlacklist             MemberBlacklistSource = "kick_blacklist"
	BlacklistSourceModerationAction          MemberBlacklistSource = "moderation_action"
	BlacklistSourceMigrationLegacyKoishi     MemberBlacklistSource = "migration_legacy_koishi"
	BlacklistSourceMigrationAdmissionFailure MemberBlacklistSource = "migration_admission_failure"
)

type MemberBlacklistScopeType string

const (
	BlacklistScopeGlobal MemberBlacklistScopeType = "global"
	BlacklistScopeGuild  MemberBlacklistScopeType = "guild"
)

type MemberBlacklistSubjectType string

const BlacklistSubjectQQUser MemberBlacklistSubjectType = "qq_user"

type MemberBlacklistReasonCode string

const (
	BlacklistReasonAdmissionTimeoutLimit    MemberBlacklistReasonCode = "admission_timeout_limit"
	BlacklistReasonManualBlacklist          MemberBlacklistReasonCode = "manual_blacklist"
	BlacklistReasonManualKickBlacklist      MemberBlacklistReasonCode = "manual_kick_blacklist"
	BlacklistReasonViolationReviewBlacklist MemberBlacklistReasonCode = "violation_review_blacklist"
	BlacklistReasonLegacyKoishiBlacklist    MemberBlacklistReasonCode = "legacy_koishi_blacklist"
	BlacklistReasonLegacyAdmissionBlacklist MemberBlacklistReasonCode = "legacy_admission_blacklist"
)

type MemberBlacklistActorType string

const (
	BlacklistActorSystem         MemberBlacklistActorType = "system"
	BlacklistActorAdminUser      MemberBlacklistActorType = "admin_user"
	BlacklistActorQQOperator     MemberBlacklistActorType = "qq_operator"
	BlacklistActorServiceAccount MemberBlacklistActorType = "service_account"
)

type MemberBlacklistCreatedFrom string

const (
	BlacklistCreatedFromAdmissionWorker  MemberBlacklistCreatedFrom = "admission_worker"
	BlacklistCreatedFromQQCommand        MemberBlacklistCreatedFrom = "qq_command"
	BlacklistCreatedFromKoishiConsole    MemberBlacklistCreatedFrom = "koishi_console"
	BlacklistCreatedFromAdminConsole     MemberBlacklistCreatedFrom = "admin_console"
	BlacklistCreatedFromModerationReview MemberBlacklistCreatedFrom = "moderation_review"
	BlacklistCreatedFromMigrationScript  MemberBlacklistCreatedFrom = "migration_script"
)

type MemberBlacklistReleaseReasonCode string

const (
	BlacklistReleaseManualPardon      MemberBlacklistReleaseReasonCode = "manual_pardon"
	BlacklistReleaseOnly              MemberBlacklistReleaseReasonCode = "release_only"
	BlacklistReleasePolicyExpiredAuto MemberBlacklistReleaseReasonCode = "policy_expired_auto"
	BlacklistReleaseAppealPassed      MemberBlacklistReleaseReasonCode = "admission_appeal_passed"
	BlacklistReleaseMigrationInverse  MemberBlacklistReleaseReasonCode = "migration_inverse"
)

type MemberBlacklistStatus string

const (
	BlacklistStatusActive   MemberBlacklistStatus = "active"
	BlacklistStatusReleased MemberBlacklistStatus = "released"
	BlacklistStatusExpired  MemberBlacklistStatus = "expired"
	BlacklistStatusAll      MemberBlacklistStatus = "all"
)

type MemberBlacklistEntry struct {
	ID                string                            `json:"id"`
	Platform          string                            `json:"platform"`
	SubjectType       MemberBlacklistSubjectType        `json:"subjectType"`
	SubjectID         string                            `json:"subjectID"`
	ScopeType         MemberBlacklistScopeType          `json:"scopeType"`
	GuildID           *string                           `json:"guildID,omitempty"`
	Source            MemberBlacklistSource             `json:"source"`
	ReasonCode        MemberBlacklistReasonCode         `json:"reasonCode"`
	ReasonText        string                            `json:"reasonText"`
	CreatedByType     MemberBlacklistActorType          `json:"createdByType"`
	CreatedByID       string                            `json:"createdByID"`
	CreatedFrom       MemberBlacklistCreatedFrom        `json:"createdFrom"`
	ExpiresAt         *time.Time                        `json:"expiresAt,omitempty"`
	ReleasedAt        *time.Time                        `json:"releasedAt,omitempty"`
	ReleasedByType    *MemberBlacklistActorType         `json:"releasedByType,omitempty"`
	ReleasedByID      *string                           `json:"releasedByID,omitempty"`
	ReleaseReasonCode *MemberBlacklistReleaseReasonCode `json:"releaseReasonCode,omitempty"`
	ReleaseReason     *string                           `json:"releaseReason,omitempty"`
	Metadata          map[string]any                    `json:"metadata"`
	CreatedAt         time.Time                         `json:"createdAt"`
	UpdatedAt         time.Time                         `json:"updatedAt"`
}

type MemberBlacklistCreateInput struct {
	Platform      string
	SubjectType   MemberBlacklistSubjectType
	SubjectID     string
	ScopeType     MemberBlacklistScopeType
	GuildID       *string
	Source        MemberBlacklistSource
	ReasonCode    MemberBlacklistReasonCode
	ReasonText    string
	CreatedByType MemberBlacklistActorType
	CreatedByID   string
	CreatedFrom   MemberBlacklistCreatedFrom
	ExpiresAt     *time.Time
	Metadata      map[string]any
}

type MemberBlacklistAccessQuery struct {
	Platform    string
	SubjectType MemberBlacklistSubjectType
	SubjectID   string
	GuildID     *string
}

type MemberBlacklistAccessDecision struct {
	CanJoin          bool                  `json:"canJoin"`
	Decision         string                `json:"decision"`
	MatchedBlacklist *MemberBlacklistEntry `json:"matchedBlacklist,omitempty"`
	Reason           *string               `json:"reason,omitempty"`
}

type MemberBlacklistListFilter struct {
	Platform    string
	SubjectType MemberBlacklistSubjectType
	SubjectID   string
	ScopeType   MemberBlacklistScopeType
	Source      MemberBlacklistSource
	GuildID     string
	CreatedByID string
	Status      MemberBlacklistStatus
	PageSize    int
	Offset      int
}

type MemberBlacklistReleaseInput struct {
	ID                string
	ReleasedByType    MemberBlacklistActorType
	ReleasedByID      string
	ReleaseReasonCode MemberBlacklistReleaseReasonCode
	ReleaseReason     *string
}

type MemberBlacklistReleaseBySubjectInput struct {
	Platform          string
	SubjectType       MemberBlacklistSubjectType
	SubjectID         string
	ScopeType         MemberBlacklistScopeType
	GuildID           *string
	ReleasedByType    MemberBlacklistActorType
	ReleasedByID      string
	ReleaseReasonCode MemberBlacklistReleaseReasonCode
	ReleaseReason     *string
}

type memberBlacklistEntryPoint string

const (
	memberBlacklistEntryPointAdmin    memberBlacklistEntryPoint = "admin"
	memberBlacklistEntryPointBot      memberBlacklistEntryPoint = "bot"
	memberBlacklistEntryPointInternal memberBlacklistEntryPoint = "internal"
)

type memberBlacklistKey struct {
	Platform    string
	SubjectType MemberBlacklistSubjectType
	SubjectID   string
	ScopeType   MemberBlacklistScopeType
	GuildID     *string
}
