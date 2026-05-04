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
	OperatorQQID  string
	GuildID       string
	ChannelID     *string
	RawCommand    string
}
