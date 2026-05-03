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

type BotAction string

const (
	BotActionMute      BotAction = "mute"
	BotActionRemind    BotAction = "remind"
	BotActionRelease   BotAction = "release"
	BotActionKick      BotAction = "kick"
	BotActionBlacklist BotAction = "blacklist"
	BotActionForward   BotAction = "forward"
)

const defaultAdmissionAuthBaseURL = "https://auth.stuhelper.com/admission/a/"

type AdmissionSession struct {
	ID                       string                 `json:"id"`
	Platform                 string                 `json:"platform"`
	GuildID                  string                 `json:"guildID"`
	ChannelID                string                 `json:"channelID"`
	QQID                     string                 `json:"qqID"`
	QQNickname               *string                `json:"qqNickname,omitempty"`
	UserID                   *int64                 `json:"userID,omitempty"`
	TokenHash                string                 `json:"-"`
	TokenExpiresAt           time.Time              `json:"tokenExpiresAt"`
	TokenConsumedAt          *time.Time             `json:"tokenConsumedAt,omitempty"`
	Status                   AdmissionSessionStatus `json:"status"`
	LinkWaitDeadlineAt       time.Time              `json:"linkWaitDeadlineAt"`
	SubmissionWaitDeadlineAt time.Time              `json:"submissionWaitDeadlineAt"`
	ManualReviewDeadlineAt   *time.Time             `json:"manualReviewDeadlineAt,omitempty"`
	InitialMuteUntil         time.Time              `json:"initialMuteUntil"`
	VerifiedAt               *time.Time             `json:"verifiedAt,omitempty"`
	CancelledAt              *time.Time             `json:"cancelledAt,omitempty"`
	LastBotError             *string                `json:"lastBotError,omitempty"`
	ProjectionPending        bool                   `json:"projectionPending"`
}

type CreatedAdmissionSession struct {
	Session *AdmissionSession `json:"session"`
	Token   string            `json:"token"`
	AuthURL string            `json:"authURL"`
}

type BotSessionCreateInput struct {
	Platform   string
	GuildID    string
	ChannelID  string
	QQID       string
	QQNickname *string
	BotSelfID  string
}

type BotEventInput struct {
	Action    BotAction
	Success   bool
	MessageID string
	Error     string
}

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
