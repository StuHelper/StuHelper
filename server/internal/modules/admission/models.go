package admission

import "time"

const (
	DefaultInitialMuteDurationSeconds       = 30 * 24 * 60 * 60
	DefaultLinkWaitSeconds                  = 60 * 60
	DefaultSubmissionWaitSeconds            = 24 * 60 * 60
	DefaultManualReviewTimeoutSeconds       = 24 * 60 * 60
	DefaultReminderIntervalSeconds          = 15 * 60
	DefaultFailedJoinLimit                  = 3
	DefaultMaxMaterialBytes           int64 = 10 * 1024 * 1024
	DefaultMaxExtensionDays                 = 90
)

type AdmissionSessionStatus string

const (
	StatusJoinedMuted       AdmissionSessionStatus = "joined_muted"
	StatusLinked            AdmissionSessionStatus = "linked"
	StatusMaterialSubmitted AdmissionSessionStatus = "material_submitted"
	StatusVerified          AdmissionSessionStatus = "verified"
	StatusExpiredKicked     AdmissionSessionStatus = "expired_kicked"
	StatusCancelled         AdmissionSessionStatus = "cancelled"
)

type FreshmanApplicationStatus string

const (
	FreshmanApplicationPending  FreshmanApplicationStatus = "pending"
	FreshmanApplicationApproved FreshmanApplicationStatus = "approved"
	FreshmanApplicationRejected FreshmanApplicationStatus = "rejected"
)

type VerificationCredentialKind string

const (
	CredentialSchoolSSO              VerificationCredentialKind = "school_sso"
	CredentialSchoolEmailOTP         VerificationCredentialKind = "school_email_otp"
	CredentialFreshmanMaterialManual VerificationCredentialKind = "freshman_material_manual"
)

type AdmissionPolicy struct {
	ID                         string
	Platform                   string
	GuildID                    string
	SchoolID                   int64
	AutoApproveJoin            bool
	InitialMuteDurationSeconds int
	LinkWaitSeconds            int
	SubmissionWaitSeconds      int
	ManualReviewTimeoutSeconds int
	ReminderIntervalSeconds    int
	FailedJoinLimit            int
	BlacklistDurationSeconds   *int
	FreshmanChannelEnabled     bool
	FreshmanChannelClosesAt    time.Time
	FreshmanDefaultExpiresAt   time.Time
	ForwardRawMaterialToQQ     bool
	ManagementGuildIDs         []string
	MaxMaterialBytes           int64
	MaxExtensionDays           int
}
