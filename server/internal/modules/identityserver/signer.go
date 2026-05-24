package identityserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Signer struct {
	issuer     string
	keyID      string
	privateKey *rsa.PrivateKey
	publicJWK  jose.JSONWebKey
}

func NewSigner(issuer, keyID, privateKeyPEM string) (*Signer, error) {
	key, err := loadPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	kid := strings.TrimSpace(keyID)
	if kid == "" {
		kid = "stuhelper-identity-1"
	}
	return &Signer{
		issuer:     strings.TrimRight(strings.TrimSpace(issuer), "/"),
		keyID:      kid,
		privateKey: key,
		publicJWK: jose.JSONWebKey{
			Key:       &key.PublicKey,
			KeyID:     kid,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		},
	}, nil
}

func (s *Signer) JWKS() jose.JSONWebKeySet {
	return jose.JSONWebKeySet{Keys: []jose.JSONWebKey{s.publicJWK}}
}

func (s *Signer) SignAccessToken(input AccessTokenInput) (string, AccessTokenClaims, error) {
	now := time.Now().UTC()
	jti := uuid.NewString()
	grantType := strings.TrimSpace(input.GrantType)
	authorizationFingerprint := strings.TrimSpace(input.AuthorizationFingerprint)
	claims := jwt.MapClaims{
		"iss":          s.issuer,
		"sub":          input.Subject,
		"aud":          input.ClientID,
		"azp":          input.ClientID,
		"client_id":    input.ClientID,
		"scope":        strings.Join(input.Scopes, " "),
		"stuhelper_id": input.UserID,
		"typ":          "access",
		"jti":          jti,
		"iat":          now.Unix(),
		"exp":          now.Add(input.TTL).Unix(),
	}
	if grantType != "" {
		claims["grant_type"] = grantType
	}
	if authorizationFingerprint != "" {
		claims["stuhelper_authz"] = authorizationFingerprint
	}
	signed, err := s.signClaims(claims)
	if err != nil {
		return "", AccessTokenClaims{}, err
	}
	return signed, AccessTokenClaims{
		Subject:                  input.Subject,
		ClientID:                 input.ClientID,
		UserID:                   input.UserID,
		Scopes:                   append([]string(nil), input.Scopes...),
		GrantType:                grantType,
		AuthorizationFingerprint: authorizationFingerprint,
		JTI:                      jti,
		IssuedAt:                 now,
		Expires:                  now.Add(input.TTL),
	}, nil
}

func (s *Signer) SignIDToken(input IDTokenInput) (string, error) {
	now := time.Now().UTC()
	authTime := input.AuthTime
	if authTime.IsZero() {
		authTime = now
	}
	claims := jwt.MapClaims{
		"iss":       s.issuer,
		"sub":       input.Subject,
		"aud":       input.ClientID,
		"azp":       input.ClientID,
		"typ":       "id_token",
		"auth_time": authTime.UTC().Unix(),
		"iat":       now.Unix(),
		"exp":       now.Add(input.TTL).Unix(),
	}
	if input.Nonce != "" {
		claims["nonce"] = input.Nonce
	}
	for key, value := range input.Profile {
		claims[key] = value
	}
	return s.signClaims(claims)
}

func (s *Signer) VerifyAccessToken(raw string) (AccessTokenClaims, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("identity token alg %q is not supported", token.Header["alg"])
		}
		return &s.privateKey.PublicKey, nil
	})
	if err != nil {
		return AccessTokenClaims{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return AccessTokenClaims{}, errors.New("identity token is invalid")
	}
	if err := validateAccessClaims(claims, s.issuer); err != nil {
		return AccessTokenClaims{}, err
	}
	return accessClaimsFromMap(claims)
}

func (s *Signer) VerifyIDToken(raw string) (IDTokenClaims, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("identity token alg %q is not supported", token.Header["alg"])
		}
		return &s.privateKey.PublicKey, nil
	})
	if err != nil {
		return IDTokenClaims{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return IDTokenClaims{}, errors.New("identity id token is invalid")
	}
	if err := validateIDClaims(claims, s.issuer); err != nil {
		return IDTokenClaims{}, err
	}
	return idClaimsFromMap(claims)
}

func (s *Signer) signClaims(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.keyID
	return token.SignedString(s.privateKey)
}

type AccessTokenInput struct {
	Subject                  string
	ClientID                 string
	UserID                   int64
	Scopes                   []string
	GrantType                string
	AuthorizationFingerprint string
	TTL                      time.Duration
}

type IDTokenInput struct {
	Subject  string
	ClientID string
	Scopes   []string
	Nonce    string
	AuthTime time.Time
	Profile  map[string]any
	TTL      time.Duration
}

func validateAccessClaims(claims jwt.MapClaims, issuer string) error {
	if !claims.VerifyIssuer(issuer, true) {
		return errors.New("identity token issuer is invalid")
	}
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return errors.New("identity token is expired")
	}
	if typ := stringClaim(claims, "typ"); typ != "access" {
		return errors.New("identity token type is invalid")
	}
	clientID := strings.TrimSpace(stringClaim(claims, "client_id"))
	if strings.TrimSpace(stringClaim(claims, "sub")) == "" ||
		clientID == "" ||
		strings.TrimSpace(stringClaim(claims, "jti")) == "" {
		return errors.New("identity token required claim is missing")
	}
	if !audienceContains(claims, clientID) {
		return errors.New("identity token audience is invalid")
	}
	if azp := strings.TrimSpace(stringClaim(claims, "azp")); azp == "" || azp != clientID {
		return errors.New("identity token authorized party is invalid")
	}
	return nil
}

func validateIDClaims(claims jwt.MapClaims, issuer string) error {
	if !claims.VerifyIssuer(issuer, true) {
		return errors.New("identity id token issuer is invalid")
	}
	if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		return errors.New("identity id token is expired")
	}
	audiences := audienceClaims(claims)
	if strings.TrimSpace(stringClaim(claims, "sub")) == "" ||
		len(audiences) == 0 {
		return errors.New("identity id token required claim is missing")
	}
	if typ := stringClaim(claims, "typ"); typ != "" && typ != "id_token" {
		return errors.New("identity id token type is invalid")
	}
	azp := strings.TrimSpace(stringClaim(claims, "azp"))
	if len(audiences) > 1 && azp == "" {
		return errors.New("identity id token authorized party is invalid")
	}
	if azp != "" && !audienceContains(claims, azp) {
		return errors.New("identity id token authorized party is invalid")
	}
	return nil
}

func accessClaimsFromMap(claims jwt.MapClaims) (AccessTokenClaims, error) {
	exp, err := numericTimeClaim(claims, "exp")
	if err != nil {
		return AccessTokenClaims{}, err
	}
	iat, err := numericTimeClaim(claims, "iat")
	if err != nil {
		return AccessTokenClaims{}, err
	}
	userID, err := int64Claim(claims, "stuhelper_id")
	if err != nil {
		return AccessTokenClaims{}, err
	}
	return AccessTokenClaims{
		Subject:                  stringClaim(claims, "sub"),
		ClientID:                 stringClaim(claims, "client_id"),
		UserID:                   userID,
		Scopes:                   strings.Fields(stringClaim(claims, "scope")),
		GrantType:                stringClaim(claims, "grant_type"),
		AuthorizationFingerprint: stringClaim(claims, "stuhelper_authz"),
		JTI:                      stringClaim(claims, "jti"),
		Expires:                  exp,
		IssuedAt:                 iat,
	}, nil
}

func idClaimsFromMap(claims jwt.MapClaims) (IDTokenClaims, error) {
	exp, err := numericTimeClaim(claims, "exp")
	if err != nil {
		return IDTokenClaims{}, err
	}
	iat, err := numericTimeClaim(claims, "iat")
	if err != nil {
		return IDTokenClaims{}, err
	}
	return IDTokenClaims{
		Subject:  stringClaim(claims, "sub"),
		ClientID: idTokenClientIDClaim(claims),
		Expires:  exp,
		IssuedAt: iat,
	}, nil
}

func audienceClaim(claims jwt.MapClaims) string {
	audiences := audienceClaims(claims)
	if len(audiences) == 0 {
		return ""
	}
	return audiences[0]
}

func idTokenClientIDClaim(claims jwt.MapClaims) string {
	if azp := stringClaim(claims, "azp"); azp != "" {
		return azp
	}
	return audienceClaim(claims)
}

func audienceContains(claims jwt.MapClaims, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	for _, audience := range audienceClaims(claims) {
		if audience == expected {
			return true
		}
	}
	return false
}

func audienceClaims(claims jwt.MapClaims) []string {
	var audiences []string
	appendAudience := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			audiences = append(audiences, value)
		}
	}
	switch value := claims["aud"].(type) {
	case string:
		appendAudience(value)
	case []string:
		for _, audience := range value {
			appendAudience(audience)
		}
	case []any:
		for _, audience := range value {
			if text, ok := audience.(string); ok {
				appendAudience(text)
			}
		}
	}
	return audiences
}

func numericTimeClaim(claims jwt.MapClaims, key string) (time.Time, error) {
	value, err := int64Claim(claims, key)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(value, 0).UTC(), nil
}

func int64Claim(claims jwt.MapClaims, key string) (int64, error) {
	switch value := claims[key].(type) {
	case float64:
		return int64(value), nil
	case int64:
		return value, nil
	case jsonNumber:
		return value.Int64()
	default:
		return 0, fmt.Errorf("identity token claim %q is invalid", key)
	}
}

type jsonNumber interface {
	Int64() (int64, error)
}

func stringClaim(claims jwt.MapClaims, key string) string {
	value, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func loadPrivateKey(raw string) (*rsa.PrivateKey, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return rsa.GenerateKey(rand.Reader, 2048)
	}
	normalized := strings.ReplaceAll(trimmed, `\n`, "\n")
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		decoded, err := base64.StdEncoding.DecodeString(trimmed)
		if err == nil {
			block, _ = pem.Decode(decoded)
		}
	}
	if block == nil {
		return nil, errors.New("identity signing private key PEM is invalid")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse identity signing private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("identity signing private key must be RSA")
	}
	return key, nil
}
