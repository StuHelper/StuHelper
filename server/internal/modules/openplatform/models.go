package openplatform

import (
	"encoding/json"
	"time"
)

const (
	AppStatusPending   = "pending"
	AppStatusApproved  = "approved"
	AppStatusSuspended = "suspended"
	AppStatusRevoked   = "revoked"

	ScopeStatusPending   = "pending"
	ScopeStatusApproved  = "approved"
	ScopeStatusRejected  = "rejected"
	ScopeStatusWithdrawn = "withdrawn"
)

const (
	ScopeProfileBasicRead   = "profile.basic.read"
	ScopeEmailRead          = "email.read"
	ScopePhoneRead          = "phone.read"
	ScopeIdentityStatusRead = "stu.identity.status.read"
	ScopeIdentityTypeRead   = "stu.identity.type.read"
	ScopeStudentStatusRead  = "stu.student.status.read"
	ScopeStudentSchoolRead  = "stu.student.school.read"
	ScopeQQStatusRead       = "stu.qq.status.read"
	ScopeQQNumberRead       = "stu.qq.number.read"
	ScopeResourceRead       = "resource.read"
	ScopeResourceWrite      = "resource.write"
	ScopeOfflineAccess      = "offline_access"
)

const (
	ResourceTypeUserProfile  = "user_profile"
	ResourceTypeResourceItem = "resource_item"

	ResourceAccessActionRead  = "read"
	ResourceAccessActionWrite = "write"

	ResourceRelationReadByApp  = "can_read_by_app"
	ResourceRelationWriteByApp = "can_write_by_app"
)

type App struct {
	ID                     int64
	CasdoorApplicationName string
	OwnerUserID            int64
	ClientID               string
	ClientSecretHash       string
	DisplayName            string
	Description            string
	HomepageURL            string
	PrivacyPolicyURL       string
	RedirectURIs           []string
	Status                 string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	Scopes              []string
	State               string
	Flow                string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	PromptNone          bool
	ForceConsent        bool
}

type AuthorizeResult struct {
	RedirectURL          string
	ConsentURL           string
	ProfileCompletionURL string
	Scopes               []ScopeDefinition
	MissingFields        []ProfileCompletionField
}

type ScopeRequest struct {
	ID             int64
	AppID          int64
	Scope          string
	Reason         string
	Status         string
	ReviewerUserID *int64
	ReviewedAt     *time.Time
	DecisionNote   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RedirectURIRequest struct {
	ID             int64
	AppID          int64
	RedirectURIs   []string
	Reason         string
	Status         string
	ReviewerUserID *int64
	ReviewedAt     *time.Time
	DecisionNote   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AppWithScopes struct {
	App                 *App
	Scopes              []ScopeRequest
	RedirectURIRequests []RedirectURIRequest
}

type AppListResult struct {
	List  []AppWithScopes
	Total int
}

type ScopeChangeResult struct {
	Scopes []ScopeRequest
}

type AuditEvent struct {
	ID        int64
	AppID     *int64
	UserID    *int64
	EventType string
	Scope     *string
	RequestID *string
	Metadata  map[string]any
	CreatedAt time.Time
}

type AuditEventListResult struct {
	List  []AuditEvent
	Total int
}

type UserConsentAuditEvent struct {
	ID             int64
	AppID          *int64
	AppDisplayName *string
	ClientID       *string
	EventType      string
	Scope          *string
	Scopes         []string
	Endpoint       *string
	Result         *string
	RequestID      *string
	Details        map[string]any
	CreatedAt      time.Time
}

type UserConsentAuditEventListResult struct {
	List  []UserConsentAuditEvent
	Total int
}

type DeveloperAppAuditEvent struct {
	ID        int64
	EventType string
	Scope     *string
	Scopes    []string
	Endpoint  *string
	Result    *string
	RequestID *string
	Details   map[string]any
	CreatedAt time.Time
}

type DeveloperAppAuditEventListResult struct {
	List  []DeveloperAppAuditEvent
	Total int
}

type TokenProbeEvidence struct {
	ID                     int64
	AppID                  int64
	ReviewerUserID         int64
	RequestID              string
	CasdoorApplicationName string
	ClientID               string
	RedirectURI            string
	ProbeMethod            string
	Result                 string
	InspectedClaims        []string
	BusinessClaims         []string
	TokenClaims            map[string][]string
	Metadata               map[string]any
	Error                  string
	CreatedAt              time.Time
}

type TokenProbeEvidenceRecord struct {
	ID                     int64
	AppID                  int64
	ReviewerUserID         *int64
	RequestID              *string
	CasdoorApplicationName string
	ClientID               string
	RedirectURI            string
	ProbeMethod            string
	Result                 string
	InspectedClaims        []string
	BusinessClaims         []string
	TokenClaims            map[string][]string
	Metadata               map[string]any
	Error                  string
	CreatedAt              time.Time
}

type TokenProbeEvidenceListResult struct {
	List  []TokenProbeEvidenceRecord
	Total int
}

type DisclosureReport struct {
	Summary             DisclosureReportSummary
	Endpoints           []DisclosureEndpointStats
	DenialReasons       []DisclosureReasonStats
	RateLimitDimensions []DisclosureRateLimitStats
	RecentReplayEvents  []DisclosureReplayEvent
}

type DisclosureReportSummary struct {
	WindowHours    int
	Total          int
	Granted        int
	Denied         int
	RateLimited    int
	ReplayDetected int
}

type DisclosureEndpointStats struct {
	Endpoint       string
	Total          int
	Granted        int
	Denied         int
	RateLimited    int
	ReplayDetected int
}

type DisclosureReasonStats struct {
	Reason string
	Total  int
}

type DisclosureRateLimitStats struct {
	Dimension string
	Total     int
}

type DisclosureReplayEvent struct {
	ID         int64
	AppID      *int64
	UserID     *int64
	Endpoint   string
	Result     string
	Count      int
	Scopes     []string
	RequestID  *string
	Metadata   map[string]any
	DetectedAt time.Time
}

type ResourceGrant struct {
	AppID        int64
	ResourceType string
	ResourceID   string
	Action       string
	Relation     string
}

type ResourceGrantResult struct {
	App    *App
	Grants []ResourceGrant
}

type ResourceAccessDecision struct {
	Allowed      bool
	AppID        int64
	ClientID     string
	ResourceType string
	ResourceID   string
	Action       string
	Relation     string
	Reason       string
}

type Consent struct {
	AppID       int64
	UserID      int64
	Scope       string
	GrantedAt   time.Time
	RevokedAt   *time.Time
	GrantSource string
	RequestID   string
}

type UserConsentScope struct {
	Scope       string
	GrantedAt   time.Time
	LastUsedAt  *time.Time
	GrantSource string
	RequestID   string
	Reason      string
}

type UserAuthorizedApp struct {
	App    *App
	Scopes []UserConsentScope
}

type AdminUserAuthorizedApp struct {
	UserID int64
	App    *App
	Scopes []UserConsentScope
}

type AdminUserConsentListResult struct {
	List  []AdminUserAuthorizedApp
	Total int
}

type UserProjection struct {
	UserID           int64
	Username         string
	Email            string
	AvatarURL        *string
	PhoneVerified    bool
	IdentityVerified bool
	ProfileStatus    *string
	SchoolID         *int64
	SchoolName       *string
	QQID             *string
}

type AuthorizationDecision struct {
	App                  *App
	UserID               int64
	Scopes               []string
	OAuthScopes          []string
	ConsentURL           string
	ProfileCompletionURL string
	InteractionRequired  bool
	InteractionError     string
	MissingFields        []ProfileCompletionField
}

type ConsentChallenge struct {
	Token               string
	AppID               int64
	UserID              int64
	Scopes              []string
	OAuthScopes         []string
	ConsentScopes       []string
	RedirectURI         string
	State               string
	Flow                string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	CreatedAt           time.Time
	ExpiresAt           time.Time
}

type ConsentPage struct {
	Token       string
	App         ConsentApp
	Scopes      []ScopeDefinition
	RedirectURI string
	ExpiresAt   time.Time
}

type ConsentApp struct {
	ID               int64
	ClientID         string
	DisplayName      string
	Description      string
	HomepageURL      string
	PrivacyPolicyURL string
}

type ConsentChallengePayload struct {
	AppID               int64    `json:"appID"`
	UserID              int64    `json:"userID"`
	Scopes              []string `json:"scopes"`
	OAuthScopes         []string `json:"oauthScopes,omitempty"`
	ConsentScopes       []string `json:"consentScopes,omitempty"`
	RedirectURI         string   `json:"redirectURI"`
	State               string   `json:"state"`
	Flow                string   `json:"flow,omitempty"`
	CodeChallenge       string   `json:"codeChallenge,omitempty"`
	CodeChallengeMethod string   `json:"codeChallengeMethod,omitempty"`
	Nonce               string   `json:"nonce,omitempty"`
	CreatedAt           string   `json:"createdAt"`
	ExpiresAt           string   `json:"expiresAt"`
}

func (c ConsentChallenge) MarshalPayload() ([]byte, error) {
	return json.Marshal(ConsentChallengePayload{
		AppID:               c.AppID,
		UserID:              c.UserID,
		Scopes:              c.Scopes,
		OAuthScopes:         c.OAuthScopes,
		ConsentScopes:       c.ConsentScopes,
		RedirectURI:         c.RedirectURI,
		State:               c.State,
		Flow:                c.Flow,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
		Nonce:               c.Nonce,
		CreatedAt:           c.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt:           c.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
}

type ProfileCompletionField struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	ActionURL   string `json:"actionURL"`
}

type ProfileCompletionChallenge struct {
	Token               string
	AppID               int64
	UserID              int64
	Scopes              []string
	OAuthScopes         []string
	RedirectURI         string
	State               string
	Flow                string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	CreatedAt           time.Time
	ExpiresAt           time.Time
}

type ProfileCompletionChallengePayload struct {
	AppID               int64    `json:"appID"`
	UserID              int64    `json:"userID"`
	Scopes              []string `json:"scopes"`
	OAuthScopes         []string `json:"oauthScopes,omitempty"`
	RedirectURI         string   `json:"redirectURI"`
	State               string   `json:"state"`
	Flow                string   `json:"flow,omitempty"`
	CodeChallenge       string   `json:"codeChallenge,omitempty"`
	CodeChallengeMethod string   `json:"codeChallengeMethod,omitempty"`
	Nonce               string   `json:"nonce,omitempty"`
	CreatedAt           string   `json:"createdAt"`
	ExpiresAt           string   `json:"expiresAt"`
}

func (c ProfileCompletionChallenge) MarshalPayload() ([]byte, error) {
	return json.Marshal(ProfileCompletionChallengePayload{
		AppID:               c.AppID,
		UserID:              c.UserID,
		Scopes:              c.Scopes,
		OAuthScopes:         c.OAuthScopes,
		RedirectURI:         c.RedirectURI,
		State:               c.State,
		Flow:                c.Flow,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
		Nonce:               c.Nonce,
		CreatedAt:           c.CreatedAt.UTC().Format(time.RFC3339Nano),
		ExpiresAt:           c.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
}

type ProfileCompletionPage struct {
	Token         string
	App           ConsentApp
	Scopes        []ScopeDefinition
	MissingFields []ProfileCompletionField
	RedirectURI   string
	ExpiresAt     time.Time
}
