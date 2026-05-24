package identityserver

import "time"

type AuthorizeRequest struct {
	ResponseType        string
	ResponseMode        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Nonce               string
	Prompt              string
	MaxAge              string
	AuthTime            time.Time
}

type LogoutRequest struct {
	IDTokenHint           string
	ClientID              string
	PostLogoutRedirectURI string
	State                 string
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
	AuthTime            time.Time `json:"authTime,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	ExpiresAt           time.Time `json:"expiresAt"`
}

type RefreshToken struct {
	ClientID                 string    `json:"clientID"`
	FamilyID                 string    `json:"familyID,omitempty"`
	Generation               int       `json:"generation,omitempty"`
	AuthorizationFingerprint string    `json:"authorizationFingerprint,omitempty"`
	Scopes                   []string  `json:"scopes"`
	UserID                   int64     `json:"userID"`
	Subject                  string    `json:"subject"`
	Nonce                    string    `json:"nonce,omitempty"`
	AuthTime                 time.Time `json:"authTime,omitempty"`
	CreatedAt                time.Time `json:"createdAt"`
	ExpiresAt                time.Time `json:"expiresAt"`
}

type AccessTokenClaims struct {
	Subject                  string
	ClientID                 string
	UserID                   int64
	Scopes                   []string
	GrantType                string
	AuthorizationFingerprint string
	JTI                      string
	Expires                  time.Time
	IssuedAt                 time.Time
}

type IDTokenClaims struct {
	Subject  string
	ClientID string
	Expires  time.Time
	IssuedAt time.Time
}
