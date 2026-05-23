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

	ScopeStatusPending  = "pending"
	ScopeStatusApproved = "approved"
	ScopeStatusRejected = "rejected"
)

const (
	ScopeProfileBasicRead   = "profile.basic.read"
	ScopeEmailRead          = "email.read"
	ScopePhoneRead          = "phone.read"
	ScopeIdentityStatusRead = "stu.identity.status.read"
	ScopeIdentityTypeRead   = "stu.identity.type.read"
	ScopeStudentStatusRead  = "stu.student.status.read"
	ScopeStudentSchoolRead  = "stu.student.school.read"
	ScopeResourceRead       = "resource.read"
	ScopeResourceWrite      = "resource.write"
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

type Consent struct {
	AppID       int64
	UserID      int64
	Scope       string
	GrantedAt   time.Time
	RevokedAt   *time.Time
	GrantSource string
	RequestID   string
}

type UserProjection struct {
	Username         string
	Email            string
	AvatarURL        *string
	PhoneEnc         []byte
	PhoneVerified    bool
	IdentityVerified bool
	ProfileStatus    *string
	SchoolID         *int64
	SchoolName       *string
}

type AuthorizationDecision struct {
	App                  *App
	UserID               int64
	Scopes               []string
	ConsentURL           string
	ProfileCompletionURL string
	MissingFields        []ProfileCompletionField
}

type ConsentChallenge struct {
	Token               string
	AppID               int64
	UserID              int64
	Scopes              []string
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
