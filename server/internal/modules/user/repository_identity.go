package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateIdentity 创建实名认证记录
func (r *Repository) CreateIdentity(ctx context.Context, identity *IdentityRecord) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_identities (
			user_id, doc_type, doc_number_enc, person_uid, real_name,
			verified, verify_method, verified_at,
			doc_photo_front, doc_photo_back, doc_photo_selfie,
			rejection_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
	`, identity.UserID, identity.DocType, identity.DocNumberEnc, identity.PersonUID, identity.RealName,
		identity.Verified, identity.VerifyMethod, identity.VerifiedAt,
		identity.DocPhotoFront, identity.DocPhotoBack, identity.DocPhotoSelfie,
		identity.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("CreateIdentity: %w", err)
	}
	return nil
}

// GetIdentityStatusByUserID 根据用户ID获取实名认证状态（最小字段集，不含 doc_number_enc/person_uid）
func (r *Repository) GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error) {
	var item IdentityStatus
	err := r.db.QueryRow(ctx, `
		SELECT user_id, doc_type, real_name,
		       verified, verify_method, verified_at,
		       rejection_reason, created_at, updated_at
		FROM user_identities
		WHERE user_id = $1
	`, userID).Scan(
		&item.UserID, &item.DocType, &item.RealName,
		&item.Verified, &item.VerifyMethod, &item.VerifiedAt,
		&item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetIdentityStatusByUserID: %w", err)
	}
	return &item, nil
}

// ListIdentityReviewItems 分页查询实名认证审核列表（不含 doc_number_enc/person_uid）
func (r *Repository) ListIdentityReviewItems(ctx context.Context, status string, page, pageSize int) ([]IdentityReviewItem, int, error) {
	var qb strings.Builder
	qb.WriteString(`
		SELECT user_id, doc_type, real_name,
		       verified, verify_method, verified_at,
		       doc_photo_front, doc_photo_back, doc_photo_selfie,
		       rejection_reason, created_at, updated_at,
		       COUNT(*) OVER() AS total
		FROM user_identities
		WHERE 1=1
	`)
	args := make([]any, 0, 4)
	argIdx := 1

	switch status {
	case StatusPending:
		qb.WriteString(` AND verified = false AND rejection_reason IS NULL`)
	case StatusRejected:
		qb.WriteString(` AND verified = false AND rejection_reason IS NOT NULL`)
	case StatusVerified:
		qb.WriteString(` AND verified = true`)
	case StatusUnverified:
		qb.WriteString(` AND verified = false`)
	}

	qb.WriteString(` ORDER BY created_at DESC`)
	qb.WriteString(` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1))
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(ctx, qb.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListIdentityReviewItems: %w", err)
	}
	defer rows.Close()

	list := make([]IdentityReviewItem, 0, pageSize)
	var total int
	for rows.Next() {
		var item IdentityReviewItem
		if err := rows.Scan(
			&item.UserID, &item.DocType, &item.RealName,
			&item.Verified, &item.VerifyMethod, &item.VerifiedAt,
			&item.DocPhotoFront, &item.DocPhotoBack, &item.DocPhotoSelfie,
			&item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
			&total,
		); err != nil {
			return nil, 0, fmt.Errorf("ListIdentityReviewItems scan: %w", err)
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ListIdentityReviewItems rows: %w", err)
	}
	return list, total, nil
}

// UpdateIdentityReviewStatus 精准更新实名认证审核状态（只更新状态字段，不触碰敏感字段）
func (r *Repository) UpdateIdentityReviewStatus(
	ctx context.Context,
	userID int64,
	approved bool,
	verifyMethod *string,
	verifiedAt *time.Time,
	rejectionReason *string,
) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_identities SET
			verified = $2,
			verify_method = $3,
			verified_at = $4,
			rejection_reason = $5,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, approved, verifyMethod, verifiedAt, rejectionReason)
	if err != nil {
		return fmt.Errorf("UpdateIdentityReviewStatus: %w", err)
	}
	return nil
}
