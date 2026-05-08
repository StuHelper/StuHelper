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
	BotSelfID                string                 `json:"botSelfID,omitempty"`
	GuildID                  string                 `json:"guildID"`
	ChannelID                string                 `json:"channelID"`
	QQID                     string                 `json:"qqID"`
	QQNickname               *string                `json:"qqNickname,omitempty"`
	UserID                   *int64                 `json:"userID,omitempty"`
	TokenHash                string                 `json:"-"`
	AuthURL                  string                 `json:"authURL,omitempty"`
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

type AdmissionMe struct {
	Status               AdmissionSessionStatus      `json:"status"`
	ProjectionPending    bool                        `json:"projectionPending"`
	Session              *AdmissionSession           `json:"session,omitempty"`
	CredentialKind       *VerificationCredentialKind `json:"credentialKind,omitempty"`
	ProvisionalExpiresAt *time.Time                  `json:"provisionalExpiresAt,omitempty"`
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

type FreshmanMaterialType string

const (
	MaterialAdmissionNotice      FreshmanMaterialType = "admission_notice"
	MaterialAdmissionCertificate FreshmanMaterialType = "admission_certificate"
)

type FreshmanApplication struct {
	ID                   string
	UserID               int64
	SchoolID             int64
	AdmissionSessionID   *string
	Status               FreshmanApplicationStatus
	ApplicantName        string
	ApplicantNameMasked  string
	DepartmentOrMajor    *string
	MaterialType         FreshmanMaterialType
	ProvisionalExpiresAt *time.Time
	ReviewedAt           *time.Time
	CreatedAt            time.Time
}

type FreshmanApplicationCreateInput struct {
	UserID            int64
	SchoolID          int64
	ApplicantName     string
	DepartmentOrMajor *string
	MaterialType      FreshmanMaterialType
}

type FreshmanReviewAction string

const (
	FreshmanReviewApprove FreshmanReviewAction = "approve"
	FreshmanReviewReject  FreshmanReviewAction = "reject"
)

type BotFreshmanReviewInput struct {
	ApplicationID string
	Action        FreshmanReviewAction
	Reason        *string
	ExpiresInDays *int
	OperatorQQID  string
	GuildID       string
	ChannelID     *string
	RawCommand    string
}

type AdminFreshmanReviewInput struct {
	ApplicationID  string
	Action         FreshmanReviewAction
	Reason         *string
	ExpiresInDays  *int
	OperatorUserID int64
}

type CameraCaptureInput struct {
	UserID        int64
	ApplicationID string
	ContentType   string
	ImageBase64   string
}

type SchoolEmailOTPInput struct {
	UserID   int64
	SchoolID int64
	Email    string
}

type SchoolEmailOTPVerifyInput struct {
	UserID   int64
	SchoolID int64
	Email    string
	Code     string
}

type SchoolEmailOTPResponse struct {
	CooldownSeconds int
}

type SchoolSSOStartInput struct {
	UserID    int64
	SchoolID  int64
	ReturnURL string
}

type SchoolSSOStartResult struct {
	RedirectURL string
	State       string
}

type SchoolSSOCompleteInput struct {
	SchoolID     int64
	State        string
	UserID       int64
	Code         string
	CodeVerifier string
}

type SchoolSSOCompleteResult struct {
	ReturnURL string
}

type SchoolSSOExchangeInput struct {
	SchoolID     int64
	Code         string
	CodeVerifier string
}

type SchoolSSOIdentity struct {
	Subject        string
	SubjectDisplay string
}

type AdmissionSchoolConfig struct {
	SchoolID        int64
	Enabled         bool
	EmailDomains    []string
	SSOLoginURL     string
	FreshmanEnabled bool
}

type ExpiredFreshmanCredential struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
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
