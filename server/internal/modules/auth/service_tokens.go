package auth

import (
	"context"
	"fmt"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/token"
)

// SignPhoneTokenPair 为手机登录用户签发自签名 JWT 对（含 session ID）
func (s *Service) SignPhoneTokenPair(
	user *PhoneUser,
	roles []string,
	sessionID string,
) (accessToken, refreshToken string, err error) {
	hmacKey := crypto.GetHMACKey()
	if len(hmacKey) == 0 {
		return "", "", fmt.Errorf("HMAC key not initialized")
	}

	accessToken, err = signPhoneToken(phoneTokenSignInput{
		User: user, Roles: roles, SessionID: sessionID,
		HMACKey: hmacKey, TokenType: token.JWTTokenTypeAccess,
		TTL: s.tokenService.GetAccessTokenTTL(),
	})
	if err != nil {
		return "", "", err
	}
	refreshTTL := time.Duration(s.tokenConfig.RefreshTokenTTL) * time.Second
	refreshToken, err = signPhoneToken(phoneTokenSignInput{
		User: user, Roles: roles, SessionID: sessionID,
		HMACKey: hmacKey, TokenType: token.JWTTokenTypeRefresh, TTL: refreshTTL,
	})
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

type phoneTokenSignInput struct {
	User      *PhoneUser
	Roles     []string
	SessionID string
	HMACKey   []byte
	TokenType token.JWTTokenType
	TTL       time.Duration
}

func signPhoneToken(input phoneTokenSignInput) (string, error) {
	claims := phoneJWTClaims(input)
	signed, err := token.SignJWT(input.HMACKey, claims, input.TTL)
	if err != nil {
		return "", fmt.Errorf("sign %s JWT: %w", input.TokenType, err)
	}
	return signed, nil
}

func phoneJWTClaims(input phoneTokenSignInput) token.JWTClaims {
	claims := token.JWTClaims{
		Sub:         input.User.CasdoorSubject,
		Name:        input.User.Username,
		Email:       input.User.Email,
		DisplayName: input.User.Username,
		Roles:       input.Roles,
		Typ:         input.TokenType,
		Sid:         input.SessionID,
	}
	if input.User.AvatarURL != nil {
		claims.Avatar = *input.User.AvatarURL
	}
	return claims
}

// SyncOIDCUser 同步 OIDC 登录的用户到本地 shadow user 表。
// 登录成功必须意味着内部主体已就绪。
func (s *Service) SyncOIDCUser(ctx context.Context, input UserSyncInput) error {
	return s.userSyncRepo.UpsertUser(ctx, input)
}

// SyncPhoneUser 通过手机号查找或创建用户。
// TODO(iam-v2): 删除本地手机号登录 token 路径；注册/登录入口应完全由 Casdoor 承担。
func (s *Service) SyncPhoneUser(ctx context.Context, phone string) (*PhoneUser, error) {
	return s.userSyncRepo.UpsertByPhone(ctx, phone)
}

// UserExistsByCasdoorSubject 检查用户是否存在（用于 refresh token 校验）。
func (s *Service) UserExistsByCasdoorSubject(ctx context.Context, casdoorSubject string) (bool, error) {
	return s.userSyncRepo.ExistsByCasdoorSubject(ctx, casdoorSubject)
}

// hashTokenForSession 生成 token 的 HMAC hash（用于 session 内存储）
func hashTokenForSession(tokenStr string) (string, error) {
	return crypto.HMACHash(tokenStr)
}
