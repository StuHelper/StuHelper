package admission

import "time"

type AdmissionPendingAction struct {
	SessionID  string    `json:"sessionID"`
	Action     BotAction `json:"action"`
	Platform   string    `json:"platform,omitempty"`
	BotSelfID  string    `json:"botSelfID,omitempty"`
	GuildID    string    `json:"guildID,omitempty"`
	ChannelID  string    `json:"channelID,omitempty"`
	QQID       string    `json:"qqID,omitempty"`
	AuthURL    string    `json:"authURL,omitempty"`
	DeadlineAt time.Time `json:"deadlineAt,omitempty"`
	Reason     string    `json:"reason,omitempty"`
}

type AdmissionQQAccess struct {
	CanJoin         bool    `json:"canJoin"`
	Reason          *string `json:"reason,omitempty"`
	AutoApproveJoin bool    `json:"autoApproveJoin,omitempty"`
}

type AdmissionQQAccessQuery struct {
	Platform string
	GuildID  string
	QQID     string
}

type AdmissionTokenLinkInput struct {
	Token   string
	QQQuery string
	UserID  int64
}

type MemberBlacklistScopeType string

const (
	MemberBlacklistScopeGuild  MemberBlacklistScopeType = "guild"
	MemberBlacklistScopeGlobal MemberBlacklistScopeType = "global"
)

type MemberBlacklistSubjectType string

const MemberBlacklistSubjectQQUser MemberBlacklistSubjectType = "qq_user"

type MemberBlacklistSource string

const (
	MemberBlacklistSourceAdmissionFailure MemberBlacklistSource = "admission_failure"
	MemberBlacklistSourceManualAdmin      MemberBlacklistSource = "manual_admin"
	MemberBlacklistSourceKickBlacklist    MemberBlacklistSource = "kick_blacklist"
	MemberBlacklistSourceModerationAction MemberBlacklistSource = "moderation_action"
	MemberBlacklistSourceLegacyKoishi     MemberBlacklistSource = "migration_legacy_koishi"
	MemberBlacklistSourceLegacyAdmission  MemberBlacklistSource = "migration_admission_failure"
)

type MemberBlacklistActorType string

const (
	MemberBlacklistActorSystem         MemberBlacklistActorType = "system"
	MemberBlacklistActorAdminUser      MemberBlacklistActorType = "admin_user"
	MemberBlacklistActorQQOperator     MemberBlacklistActorType = "qq_operator"
	MemberBlacklistActorServiceAccount MemberBlacklistActorType = "service_account"
)

type MemberBlacklistCreatedFrom string

const (
	MemberBlacklistFromAdmissionWorker  MemberBlacklistCreatedFrom = "admission_worker"
	MemberBlacklistFromQQCommand        MemberBlacklistCreatedFrom = "qq_command"
	MemberBlacklistFromKoishiConsole    MemberBlacklistCreatedFrom = "koishi_console"
	MemberBlacklistFromAdminConsole     MemberBlacklistCreatedFrom = "admin_console"
	MemberBlacklistFromModerationReview MemberBlacklistCreatedFrom = "moderation_review"
)

type MemberBlacklistEntry struct {
	ID                string                     `json:"id"`
	Platform          string                     `json:"platform"`
	SubjectType       MemberBlacklistSubjectType `json:"subjectType"`
	SubjectID         string                     `json:"subjectID"`
	ScopeType         MemberBlacklistScopeType   `json:"scopeType"`
	GuildID           *string                    `json:"guildID,omitempty"`
	Source            MemberBlacklistSource      `json:"source"`
	ReasonCode        string                     `json:"reasonCode"`
	ReasonText        string                     `json:"reasonText"`
	CreatedByType     MemberBlacklistActorType   `json:"createdByType"`
	CreatedByID       string                     `json:"createdByID"`
	CreatedFrom       MemberBlacklistCreatedFrom `json:"createdFrom"`
	ExpiresAt         *time.Time                 `json:"expiresAt,omitempty"`
	ReleasedAt        *time.Time                 `json:"releasedAt,omitempty"`
	ReleasedByType    *MemberBlacklistActorType  `json:"releasedByType,omitempty"`
	ReleasedByID      *string                    `json:"releasedByID,omitempty"`
	ReleaseReasonCode *string                    `json:"releaseReasonCode,omitempty"`
	ReleaseReason     *string                    `json:"releaseReason,omitempty"`
	Metadata          map[string]any             `json:"metadata"`
	CreatedAt         time.Time                  `json:"createdAt"`
	UpdatedAt         time.Time                  `json:"updatedAt"`
}

type MemberBlacklistAccessQuery struct {
	Platform    string
	SubjectType MemberBlacklistSubjectType
	SubjectID   string
	GuildID     string
}

type MemberBlacklistAccessDecision struct {
	CanJoin          bool                  `json:"canJoin"`
	Decision         string                `json:"decision"`
	MatchedBlacklist *MemberBlacklistEntry `json:"matchedBlacklist,omitempty"`
}

type MemberBlacklistCreateInput struct {
	ID            string
	Platform      string
	SubjectType   MemberBlacklistSubjectType
	SubjectID     string
	ScopeType     MemberBlacklistScopeType
	GuildID       string
	Source        MemberBlacklistSource
	ReasonCode    string
	ReasonText    string
	CreatedByType MemberBlacklistActorType
	CreatedByID   string
	CreatedFrom   MemberBlacklistCreatedFrom
	ExpiresAt     *time.Time
	Metadata      map[string]any
}

type MemberBlacklistCreateTxInput struct {
	Entry MemberBlacklistCreateInput
	Now   time.Time
}

type MemberBlacklistReleaseInput struct {
	ID                string
	ReleasedByType    MemberBlacklistActorType
	ReleasedByID      string
	ReleaseReasonCode string
	ReleaseReason     string
}

type MemberBlacklistReleaseBySubjectInput struct {
	Platform          string
	SubjectType       MemberBlacklistSubjectType
	SubjectID         string
	ScopeType         MemberBlacklistScopeType
	GuildID           string
	ReleasedByType    MemberBlacklistActorType
	ReleasedByID      string
	ReleaseReasonCode string
	ReleaseReason     string
}

type MemberBlacklistListFilter struct {
	Platform    string
	SubjectType MemberBlacklistSubjectType
	SubjectID   string
	ScopeType   MemberBlacklistScopeType
	GuildID     string
	PageSize    int
	Offset      int
	ActiveOnly  bool
}

type AdmissionBlacklistReleaseInput struct {
	Platform string
	GuildID  string
	QQID     string
}

type AdmissionJoinRequestEventInput struct {
	Platform  string
	GuildID   string
	QQID      string
	RequestID string
	Success   bool
	Error     string
	RawEvent  map[string]any
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

type AdmissionSessionListFilter struct {
	Status   AdmissionSessionStatus
	PageSize int
	Offset   int
}

type FreshmanApplicationListFilter struct {
	Status   FreshmanApplicationStatus
	PageSize int
	Offset   int
}

type AdmissionPendingActionFilter struct {
	Platform  string
	BotSelfID string
	Limit     int
}

type BotFreshmanCommandInput struct {
	ApplicationID string
	Platform      string
	TargetGuildID string
	OperatorQQID  string
	GuildID       string
	ChannelID     *string
	RawCommand    string
}
