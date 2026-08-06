package admission

import "time"

type AdmissionPendingAction struct {
	ActionID               string    `json:"actionID,omitempty"`
	DispatchAttempt        int       `json:"dispatchAttempt,omitempty"`
	EligibilityRevision    *int64    `json:"eligibilityRevision,omitempty"`
	SessionID              string    `json:"sessionID"`
	Action                 BotAction `json:"action"`
	Platform               string    `json:"platform,omitempty"`
	BotSelfID              string    `json:"botSelfID,omitempty"`
	GuildID                string    `json:"guildID,omitempty"`
	ChannelID              string    `json:"channelID,omitempty"`
	QQID                   string    `json:"qqID,omitempty"`
	AuthURL                string    `json:"authURL,omitempty"`
	DeadlineAt             time.Time `json:"deadlineAt,omitempty"`
	Reason                 string    `json:"reason,omitempty"`
	FailureCount           int       `json:"failureCount,omitempty"`
	RemainingRetryCount    int       `json:"remainingRetryCount,omitempty"`
	WillBlacklistOnTimeout bool      `json:"willBlacklistOnTimeout,omitempty"`
}

type AdmissionTokenLinkInput struct {
	Token   string
	QQQuery string
	UserID  int64
}

type AdmissionJoinRequestEventInput struct {
	Platform  string
	GuildID   string
	QQID      string
	RequestID string
	Decision  AdmissionJoinRequestDecisionAction
	Success   bool
	Error     string
	RawEvent  map[string]any
}

type AdmissionJoinRequestDecisionAction string

const (
	AdmissionJoinRequestDecisionApprove AdmissionJoinRequestDecisionAction = "approve"
	AdmissionJoinRequestDecisionReject  AdmissionJoinRequestDecisionAction = "reject"
)

type AdmissionJoinRequestVerificationState string

const (
	AdmissionJoinRequestVerified   AdmissionJoinRequestVerificationState = "verified"
	AdmissionJoinRequestUnverified AdmissionJoinRequestVerificationState = "unverified"
)

type AdmissionJoinRequestDecisionInput struct {
	Platform  string
	GuildID   string
	QQID      string
	RequestID string
	RawEvent  map[string]any
}

type AdmissionJoinRequestDecision struct {
	Decision                  AdmissionJoinRequestDecisionAction    `json:"decision"`
	Reason                    string                                `json:"reason,omitempty"`
	VerificationState         AdmissionJoinRequestVerificationState `json:"verificationState"`
	JoinHandlingStrategy      AdmissionJoinHandlingStrategy         `json:"joinHandlingStrategy"`
	AutoApproveVerifiedJoin   bool                                  `json:"autoApproveVerifiedJoin"`
	AutoApproveUnverifiedJoin bool                                  `json:"autoApproveUnverifiedJoin"`
	PolicyID                  string                                `json:"policyID,omitempty"`
	UserID                    *string                               `json:"userID,omitempty"`
}

type FreshmanForwardItem struct {
	Application        *FreshmanApplication `json:"application"`
	MaterialURL        string               `json:"materialURL"`
	ManagementGuildIDs []string             `json:"managementGuildIDs"`
	Platform           string               `json:"platform,omitempty"`
	BotSelfID          string               `json:"botSelfID,omitempty"`
	SchoolName         string               `json:"schoolName,omitempty"`
	QQID               string               `json:"qqID,omitempty"`
}

type adminFreshmanApplicationRow struct {
	Application  FreshmanApplication
	ObjectKey    *string
	QQID         *string
	FailureCount *int
}

type AdmissionSessionListFilter struct {
	Status    AdmissionSessionStatus
	Platform  string
	BotSelfID string
	GuildID   string
	QQID      string
	PageSize  int
	Offset    int
}

type FreshmanApplicationListFilter struct {
	Status   FreshmanApplicationStatus
	PageSize int
	Offset   int
}

type AdminAdmissionSessionActionInput struct {
	SessionID      string
	OperatorUserID int64
}

type AdmissionPendingActionFilter struct {
	Platform  string
	BotSelfID string
	Limit     int
}

type AdmissionBotActionStatus string

const (
	AdmissionBotActionPending    AdmissionBotActionStatus = "pending"
	AdmissionBotActionDispatched AdmissionBotActionStatus = "dispatched"
	AdmissionBotActionSucceeded  AdmissionBotActionStatus = "succeeded"
	AdmissionBotActionFailed     AdmissionBotActionStatus = "failed"
	AdmissionBotActionDeadLetter AdmissionBotActionStatus = "dead_letter"
	AdmissionBotActionStale      AdmissionBotActionStatus = "stale"
)

type AdmissionBotActionOutboxRow struct {
	ID                  int64
	ActionKey           string
	SessionID           string
	Action              BotAction
	Platform            string
	BotSelfID           string
	GuildID             string
	ChannelID           string
	QQID                string
	ScheduledAt         time.Time
	Status              AdmissionBotActionStatus
	AttemptCount        int
	NextAttemptAt       time.Time
	LastError           *string
	MessageID           *string
	EligibilityRevision *int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Session             AdmissionSession
}

type AdmissionBotActionQueueInput struct {
	Session     *AdmissionSession
	Action      BotAction
	ScheduledAt time.Time
	Now         time.Time
}

type BotFreshmanCommandInput struct {
	ApplicationID string
	OperatorQQID  string
	GuildID       string
	ChannelID     *string
	RawCommand    string
}
