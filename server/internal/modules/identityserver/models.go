package identityserver

import "time"

type AuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
}

type AuthorizationCode struct {
	ClientID            string    `json:"clientID"`
	RedirectURI         string    `json:"redirectURI"`
	Scopes              []string  `json:"scopes"`
	UserID              int64     `json:"userID"`
	Subject             string    `json:"subject"`
	CodeChallenge       string    `json:"codeChallenge,omitempty"`
	CodeChallengeMethod string    `json:"codeChallengeMethod,omitempty"`
	Nonce               string    `json:"nonce,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	ExpiresAt           time.Time `json:"expiresAt"`
}

type AccessTokenClaims struct {
	Subject  string
	ClientID string
	UserID   int64
	Scopes   []string
	JTI      string
	Expires  time.Time
	IssuedAt time.Time
}
