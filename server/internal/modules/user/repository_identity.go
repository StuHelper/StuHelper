package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/StuHelper/StuHelper/server/internal/pkg/httputil"
)

// CreateIdentity 创建实名认证记录
func (r *Repository) CreateIdentity(ctx context.Context, identity *IdentityRecord) error {
	ctx = withDBTable(ctx, "user_identities")
	_, err := r.db.Exec(ctx, `
		INSERT INTO user_identities (
			user_id, doc_type, doc_number_enc, person_uid, real_name,
			verified, verify_method, reviewed_at, verified_at,
			doc_photo_front, doc_photo_back, doc_photo_selfie,
			rejection_reason, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
	`, identity.UserID, identity.DocType, identity.DocNumberEnc, identity.PersonUID, identity.RealName,
		identity.Verified, identity.VerifyMethod, identity.ReviewedAt, identity.VerifiedAt,
		identity.DocPhotoFront, identity.DocPhotoBack, identity.DocPhotoSelfie,
		identity.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("CreateIdentity: %w", err)
	}
	return nil
}

// UpdateIdentitySubmission 覆盖已驳回的实名认证提交内容，并重置审核状态。
func (r *Repository) UpdateIdentitySubmission(ctx context.Context, identity *IdentityRecord) error {
	ctx = withDBTable(ctx, "user_identities")
	tag, err := r.db.Exec(ctx, `
		UPDATE user_identities SET
			doc_type = $2,
			doc_number_enc = $3,
			person_uid = $4,
			real_name = $5,
			verified = $6,
			verify_method = $7,
			reviewed_at = $8,
			verified_at = $9,
			doc_photo_front = $10,
			doc_photo_back = $11,
			doc_photo_selfie = $12,
			rejection_reason = $13,
			updated_at = NOW()
		WHERE user_id = $1
	`, identity.UserID, identity.DocType, identity.DocNumberEnc, identity.PersonUID, identity.RealName,
		identity.Verified, identity.VerifyMethod, identity.ReviewedAt, identity.VerifiedAt,
		identity.DocPhotoFront, identity.DocPhotoBack, identity.DocPhotoSelfie,
		identity.RejectionReason,
	)
	if err != nil {
		return fmt.Errorf("UpdateIdentitySubmission: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIdentityNotFound
	}
	return nil
}

// GetIdentityStatusByUserID 根据用户ID获取实名认证状态（最小字段集，不含 doc_number_enc/person_uid）
func (r *Repository) GetIdentityStatusByUserID(ctx context.Context, userID int64) (*IdentityStatus, error) {
	ctx = withDBTable(ctx, "user_identities")
	var item IdentityStatus
	err := r.db.QueryRow(ctx, `
		SELECT user_id, doc_type, real_name,
		       verified, verify_method, reviewed_at, verified_at,
		       rejection_reason, created_at, updated_at
		FROM user_identities
		WHERE user_id = $1
	`, userID).Scan(
		&item.UserID, &item.DocType, &item.RealName,
		&item.Verified, &item.VerifyMethod, &item.ReviewedAt, &item.VerifiedAt,
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
	ctx = withDBTable(ctx, "user_identities")
	pageSize = httputil.ClampPageSize(pageSize)
	offset := httputil.SafeOffset(page, pageSize)
	args := make([]any, 0, 4)
	whereClause := ""

	switch status {
	case StatusPending:
		whereClause = ` WHERE verified = false AND reviewed_at IS NULL`
	case StatusRejected:
		whereClause = ` WHERE verified = false AND reviewed_at IS NOT NULL`
	case StatusVerified:
		whereClause = ` WHERE verified = true`
	}

	countQuery := `SELECT COUNT(*) FROM user_identities` + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("ListIdentityReviewItems count: %w", err)
	}
	if total == 0 {
		return []IdentityReviewItem{}, 0, nil
	}

	args = append(args, pageSize, offset)
	rows, err := r.db.Query(ctx, `
		SELECT user_id, doc_type, real_name,
		       verified, verify_method, reviewed_at, verified_at,
		       doc_photo_front, doc_photo_back, doc_photo_selfie,
		       rejection_reason, created_at, updated_at
		FROM user_identities
	`+whereClause+`
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("ListIdentityReviewItems data: %w", err)
	}
	defer rows.Close()

	list := make([]IdentityReviewItem, 0)
	for rows.Next() {
		var item IdentityReviewItem
		if err := rows.Scan(
			&item.UserID, &item.DocType, &item.RealName,
			&item.Verified, &item.VerifyMethod, &item.ReviewedAt, &item.VerifiedAt,
			&item.DocPhotoFront, &item.DocPhotoBack, &item.DocPhotoSelfie,
			&item.RejectionReason, &item.CreatedAt, &item.UpdatedAt,
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
	reviewedAt *time.Time,
	verifiedAt *time.Time,
	rejectionReason *string,
) error {
	ctx = withDBTable(ctx, "user_identities")
	tag, err := r.db.Exec(ctx, `
		UPDATE user_identities SET
			verified = $2,
			verify_method = $3,
			reviewed_at = $4,
			verified_at = $5,
			rejection_reason = $6,
			updated_at = NOW()
		WHERE user_id = $1
	`, userID, approved, verifyMethod, reviewedAt, verifiedAt, rejectionReason)
	if err != nil {
		return fmt.Errorf("UpdateIdentityReviewStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIdentityNotFound
	}
	return nil
}
