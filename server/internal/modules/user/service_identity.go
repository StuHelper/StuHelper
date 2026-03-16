package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// GetIdentity 获取实名认证状态信息（不含敏感字段）
func (s *Service) GetIdentity(ctx context.Context, userID int64) (*IdentityStatus, error) {
	return s.repo.GetIdentityStatusByUserID(ctx, userID)
}

// SubmitIdentity 提交实名认证
func (s *Service) SubmitIdentity(ctx context.Context, userID int64, req SubmitIdentityRequest) (*IdentityStatus, error) {
	existing, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity check existing: %w", err)
	}
	if existing != nil {
		if existing.Verified {
			return nil, ErrIdentityAlreadyVerified
		}
		return nil, ErrIdentityAlreadyExists
	}

	if req.DocType != DocTypeMainlandID {
		if req.DocPhotoFront == nil || *req.DocPhotoFront == "" {
			return nil, ErrPhotoRequired
		}
	}

	personUID := s.computePersonUID(req.DocType, req.DocNumber)
	docNumberEnc, err := s.docCipher.Encrypt(req.DocNumber)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity encrypt doc number: %w", err)
	}

	identity := &IdentityRecord{
		UserID:         userID,
		DocType:        req.DocType,
		DocNumberEnc:   docNumberEnc,
		PersonUID:      personUID,
		RealName:       req.RealName,
		Verified:       false,
		DocPhotoFront:  req.DocPhotoFront,
		DocPhotoBack:   req.DocPhotoBack,
		DocPhotoSelfie: req.DocPhotoSelfie,
	}

	if req.DocType == DocTypeMainlandID {
		matched, err := s.tryAcademicDBMatch(ctx, req.DocNumber, req.RealName)
		if err != nil {
			logger.L().Warn("academic DB match failed, falling through",
				zap.Int64("user_id", userID),
				zap.Error(err),
			)
		}
		if matched {
			method := VerifyMethodAcademicDB
			now := time.Now()
			identity.Verified = true
			identity.VerifyMethod = &method
			identity.VerifiedAt = &now
		}
	}

	if err := s.repo.CreateIdentity(ctx, identity); err != nil {
		return nil, fmt.Errorf("SubmitIdentity create: %w", err)
	}

	result, err := s.repo.GetIdentityStatusByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("SubmitIdentity reload: %w", err)
	}
	return result, nil
}

// tryAcademicDBMatch 尝试通过学籍数据库匹配进行自动实名验证
func (s *Service) tryAcademicDBMatch(ctx context.Context, docNumber, realName string) (bool, error) {
	students, err := s.repo.FindAcademicStudentsByPersonUID(ctx, DocTypeMainlandID, docNumber)
	if err != nil {
		return false, err
	}

	for _, stu := range students {
		if stu.XM != nil && strings.EqualFold(strings.TrimSpace(*stu.XM), strings.TrimSpace(realName)) {
			return true, nil
		}
	}
	return false, nil
}
