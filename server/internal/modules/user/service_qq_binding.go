package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const qqBindingCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GetQQBinding 获取当前用户的 QQ 绑定状态。
func (s *Service) GetQQBinding(ctx context.Context, userID int64) (*QQBinding, error) {
	binding, err := s.repo.GetQQBindingByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetQQBinding: %w", err)
	}
	return binding, nil
}

// GenerateQQBindingCode 生成用户私聊机器人时使用的一次性绑定码。
func (s *Service) GenerateQQBindingCode(ctx context.Context, userID int64, ttl time.Duration) (*GeneratedQQBindingCode, error) {
	if err := s.ensureQQBindingAbsent(ctx, userID); err != nil {
		return nil, err
	}

	code, err := generateQQBindingCode()
	if err != nil {
		return nil, fmt.Errorf("GenerateQQBindingCode generate code: %w", err)
	}

	expiresAt := time.Now().Add(ttl)
	if err := s.repo.UpsertQQBindingCode(ctx, &QQBindingCode{
		UserID:    userID,
		CodeHash:  s.hashQQBindingCode(code),
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("GenerateQQBindingCode store: %w", err)
	}

	return &GeneratedQQBindingCode{Code: code, ExpiresAt: expiresAt}, nil
}

// ConsumeQQBindingCode 将绑定码与 QQ 账号建立永久绑定。
func (s *Service) ConsumeQQBindingCode(ctx context.Context, code, qqID string, qqNickname *string) (*QQBinding, error) {
	trimmedCode := strings.TrimSpace(code)
	trimmedQQID := strings.TrimSpace(qqID)
	if trimmedQQID == "" {
		return nil, ErrQQIDRequired
	}

	var result *QQBinding
	if err := s.repo.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		bindingCode, err := s.loadQQBindingCodeForConsume(ctx, tx, trimmedCode)
		if err != nil {
			return err
		}
		result, err = s.consumeQQBindingCodeTx(ctx, tx, bindingCode, trimmedQQID, normalizeOptionalString(qqNickname))
		return err
	}); err != nil {
		return nil, fmt.Errorf("ConsumeQQBindingCode: %w", err)
	}

	return result, nil
}

// GetQQVerificationStateByQQID 查询 QQ 账号对应的学生认证状态。
func (s *Service) GetQQVerificationStateByQQID(ctx context.Context, qqID string) (*QQVerificationStatus, error) {
	trimmedQQID := strings.TrimSpace(qqID)
	if trimmedQQID == "" {
		return nil, ErrQQIDRequired
	}

	binding, err := s.repo.GetQQBindingByQQID(ctx, trimmedQQID)
	if err != nil {
		return nil, fmt.Errorf("GetQQVerificationStateByQQID binding: %w", err)
	}
	if binding == nil {
		return newUnboundQQVerificationStatus(trimmedQQID), nil
	}

	profile, err := s.repo.GetProfileByUserID(ctx, binding.UserID)
	if err != nil {
		return nil, fmt.Errorf("GetQQVerificationStateByQQID profile: %w", err)
	}

	return buildQQVerificationStatus(binding, profile), nil
}

func (s *Service) ensureQQBindingAbsent(ctx context.Context, userID int64) error {
	binding, err := s.repo.GetQQBindingByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("ensureQQBindingAbsent: %w", err)
	}
	if binding != nil {
		return ErrQQBindingAlreadyExists
	}
	return nil
}

func generateQQBindingCode() (string, error) {
	buf := make([]byte, qqBindingCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = qqBindingCodeAlphabet[int(buf[i])%len(qqBindingCodeAlphabet)]
	}
	return string(buf), nil
}

func (s *Service) hashQQBindingCode(code string) string {
	return s.computePersonUID("qq_binding_code", strings.TrimSpace(code))
}

func (s *Service) loadQQBindingCodeForConsume(ctx context.Context, tx pgx.Tx, code string) (*QQBindingCode, error) {
	if strings.TrimSpace(code) == "" {
		return nil, ErrQQBindingCodeInvalid
	}

	bindingCode, err := s.repo.GetQQBindingCodeByHashTx(ctx, tx, s.hashQQBindingCode(code))
	if err != nil {
		return nil, fmt.Errorf("loadQQBindingCodeForConsume: %w", err)
	}
	if bindingCode == nil || bindingCode.ConsumedAt != nil {
		return nil, ErrQQBindingCodeInvalid
	}
	if time.Now().After(bindingCode.ExpiresAt) {
		return nil, ErrQQBindingCodeExpired
	}
	return bindingCode, nil
}

func (s *Service) consumeQQBindingCodeTx(ctx context.Context, tx pgx.Tx, bindingCode *QQBindingCode, qqID string, qqNickname *string) (*QQBinding, error) {
	userBinding, err := s.repo.GetQQBindingByUserIDTx(ctx, tx, bindingCode.UserID)
	if err != nil {
		return nil, fmt.Errorf("consumeQQBindingCodeTx user binding: %w", err)
	}

	qqBinding, err := s.repo.GetQQBindingByQQIDTx(ctx, tx, qqID)
	if err != nil {
		return nil, fmt.Errorf("consumeQQBindingCodeTx qq binding: %w", err)
	}

	existing, err := resolveQQBindingConflict(userBinding, qqBinding, bindingCode.UserID, qqID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := s.repo.MarkQQBindingCodeConsumedTx(ctx, tx, bindingCode.UserID, time.Now()); err != nil {
			return nil, fmt.Errorf("consumeQQBindingCodeTx consume code: %w", err)
		}
		return existing, nil
	}

	now := time.Now()
	binding := &QQBinding{
		UserID:     bindingCode.UserID,
		QQID:       qqID,
		QQNickname: qqNickname,
		BoundAt:    now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.repo.CreateQQBindingTx(ctx, tx, binding); err != nil {
		return nil, fmt.Errorf("consumeQQBindingCodeTx create binding: %w", err)
	}
	if err := s.repo.MarkQQBindingCodeConsumedTx(ctx, tx, bindingCode.UserID, now); err != nil {
		return nil, fmt.Errorf("consumeQQBindingCodeTx consume code: %w", err)
	}
	return binding, nil
}

func resolveQQBindingConflict(userBinding, qqBinding *QQBinding, userID int64, qqID string) (*QQBinding, error) {
	if userBinding != nil && userBinding.QQID != qqID {
		return nil, ErrQQBindingUserConflict
	}
	if qqBinding != nil && qqBinding.UserID != userID {
		return nil, ErrQQBindingQQAlreadyBound
	}
	if userBinding != nil {
		return userBinding, nil
	}
	if qqBinding != nil {
		return qqBinding, nil
	}
	return nil, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func newUnboundQQVerificationStatus(qqID string) *QQVerificationStatus {
	return &QQVerificationStatus{
		QQID:                      qqID,
		VerificationState:         QQVerificationStateUnbound,
		ProfileVerificationStatus: StatusUnverified,
		StudentVerified:           false,
	}
}

func buildQQVerificationStatus(binding *QQBinding, profile *Profile) *QQVerificationStatus {
	state := QQVerificationStateBoundUnverified
	status := StatusUnverified
	verified := false
	if profile != nil {
		status = profile.VerificationStatus
	}
	if profile != nil && profile.VerificationStatus == StatusVerified {
		state = QQVerificationStateVerified
		verified = true
	}

	return &QQVerificationStatus{
		QQID:                      binding.QQID,
		UserID:                    &binding.UserID,
		QQNickname:                binding.QQNickname,
		BoundAt:                   &binding.BoundAt,
		VerificationState:         state,
		ProfileVerificationStatus: status,
		StudentVerified:           verified,
	}
}
