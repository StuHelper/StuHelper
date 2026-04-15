package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto/pii"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/phoneutil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/usersync"
)

// AuthSyncInput 认证同步输入，别名到 usersync.Input。
type AuthSyncInput = usersync.Input

// PhoneUser 手机号登录用户，别名到 usersync.PhoneUser。
type PhoneUser = usersync.PhoneUser

// ErrPhoneUserNotFound 手机号对应的用户不存在，别名到 usersync.ErrPhoneUserNotFound。
var ErrPhoneUserNotFound = usersync.ErrPhoneUserNotFound

// UserSyncRepository 用户同步持久化层，负责认证流程中的用户创建 / 查找 / 回填。
// 业务归属为 user domain，auth domain 通过窄接口依赖。
type UserSyncRepository struct {
	db          *db.DB
	phoneCipher pii.Encryptor
	hmacKey     []byte
}

// NewUserSyncRepository 创建用户同步仓储
func NewUserSyncRepository(database *db.DB, phoneCipher pii.Encryptor, hmacKey []byte) *UserSyncRepository {
	return &UserSyncRepository{
		db:          database,
		phoneCipher: phoneCipher,
		hmacKey:     hmacKey,
	}
}

// UpsertUser 同步 OIDC / SSO 登录用户到本地 shadow user 表。
func (r *UserSyncRepository) UpsertUser(ctx context.Context, input AuthSyncInput) error {
	if input.ExternalID == "" || input.Username == "" {
		return fmt.Errorf("UpsertUser: externalID and username are required")
	}

	userHash, err := crypto.HMACHashWithKey(input.ExternalID, r.hmacKey)
	if err != nil {
		return fmt.Errorf("UpsertUser: compute user_hash: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO users (external_id, username, email, avatar_url, user_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (external_id) DO UPDATE SET
			username = EXCLUDED.username,
			email = COALESCE(EXCLUDED.email, users.email),
			avatar_url = COALESCE(EXCLUDED.avatar_url, users.avatar_url),
			user_hash = COALESCE(EXCLUDED.user_hash, users.user_hash),
			updated_at = NOW()
	`, input.ExternalID, input.Username, emptyToNil(input.Email), input.AvatarURL, userHash)
	if err != nil {
		return fmt.Errorf("UpsertUser: %w", err)
	}
	return nil
}

func (r *UserSyncRepository) encryptAndHashPhone(phone string) ([]byte, string, error) {
	if r.phoneCipher == nil {
		return nil, "", fmt.Errorf("phone cipher is not configured")
	}

	phoneEnc, err := r.phoneCipher.Encrypt(phone)
	if err != nil {
		return nil, "", fmt.Errorf("encrypt phone: %w", err)
	}
	phoneHash, err := phoneutil.HashLookupWithKey(phone, r.hmacKey)
	if err != nil {
		return nil, "", fmt.Errorf("hash phone: %w", err)
	}
	return phoneEnc, phoneHash, nil
}

// FindByPhone 通过手机号查找用户。
func (r *UserSyncRepository) FindByPhone(ctx context.Context, phone string) (*PhoneUser, error) {
	var u PhoneUser
	phoneHash, err := phoneutil.HashLookupWithKey(phone, r.hmacKey)
	if err != nil {
		return nil, fmt.Errorf("FindByPhone hash phone: %w", err)
	}

	err = r.db.QueryRow(ctx, `
		SELECT id, external_id, username, COALESCE(email, ''), avatar_url
		FROM users WHERE phone_hash = $1
	`, phoneHash).Scan(&u.ID, &u.ExternalID, &u.Username, &u.Email, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPhoneUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("FindByPhone: %w", err)
	}
	u.MaskedPhone = phoneutil.Mask(phone)
	return &u, nil
}

func isAuthSyncUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *UserSyncRepository) loadPhoneUserForUpdate(ctx context.Context, tx pgx.Tx, phoneHash string) (*PhoneUser, error) {
	var u PhoneUser
	err := tx.QueryRow(ctx, `
		SELECT id, external_id, username, COALESCE(email, ''), avatar_url
		FROM users
		WHERE phone_hash = $1
		LIMIT 1
		FOR UPDATE
	`, phoneHash).Scan(&u.ID, &u.ExternalID, &u.Username, &u.Email, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserSyncRepository) upsertPhoneUserTx(ctx context.Context, tx pgx.Tx, phone, phoneHash string, phoneEnc []byte) (*PhoneUser, error) {
	existing, err := r.loadPhoneUserForUpdate(ctx, tx, phoneHash)
	if err != nil {
		return nil, fmt.Errorf("load phone user: %w", err)
	}
	if existing != nil {
		userHash, hashErr := crypto.HMACHashWithKey(existing.ExternalID, r.hmacKey)
		if hashErr != nil {
			return nil, fmt.Errorf("compute user_hash for existing phone user: %w", hashErr)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET phone_enc = $2,
			    phone_hash = $3,
			    user_hash = COALESCE(users.user_hash, $4),
			    updated_at = NOW()
			WHERE id = $1
		`, existing.ID, phoneEnc, phoneHash, userHash); err != nil {
			return nil, fmt.Errorf("update phone user: %w", err)
		}
		existing.MaskedPhone = phoneutil.Mask(phone)
		return existing, nil
	}

	externalID := "phone:" + uuid.NewString()
	userHash, hashErr := crypto.HMACHashWithKey(externalID, r.hmacKey)
	if hashErr != nil {
		return nil, fmt.Errorf("compute user_hash for new phone user: %w", hashErr)
	}
	username := "user_" + phone[len(phone)-4:]
	created := &PhoneUser{MaskedPhone: phoneutil.Mask(phone)}
	insertErr := tx.QueryRow(ctx, `
		INSERT INTO users (external_id, username, phone_enc, phone_hash, user_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, external_id, username, COALESCE(email, ''), avatar_url
	`, externalID, username, phoneEnc, phoneHash, userHash).Scan(
		&created.ID,
		&created.ExternalID,
		&created.Username,
		&created.Email,
		&created.AvatarURL,
	)
	if insertErr == nil {
		return created, nil
	}
	if !isAuthSyncUniqueViolation(insertErr) {
		return nil, fmt.Errorf("insert phone user: %w", insertErr)
	}

	existing, err = r.loadPhoneUserForUpdate(ctx, tx, phoneHash)
	if err != nil {
		return nil, fmt.Errorf("reload phone user after conflict: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("insert phone user conflict without row")
	}
	existing.MaskedPhone = phoneutil.Mask(phone)
	return existing, nil
}

// UpsertByPhone 通过手机号查找或创建用户。
func (r *UserSyncRepository) UpsertByPhone(ctx context.Context, phone string) (*PhoneUser, error) {
	phoneEnc, phoneHash, err := r.encryptAndHashPhone(phone)
	if err != nil {
		return nil, fmt.Errorf("UpsertByPhone prepare phone: %w", err)
	}

	var u *PhoneUser
	err = r.db.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		created, err := r.upsertPhoneUserTx(ctx, tx, phone, phoneHash, phoneEnc)
		if err != nil {
			return err
		}
		u = created
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("UpsertByPhone: %w", err)
	}
	return u, nil
}

// ExistsByExternalID 检查 external_id 对应的用户是否仍然存在。
func (r *UserSyncRepository) ExistsByExternalID(ctx context.Context, externalID string) (bool, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE external_id = $1)`, externalID).Scan(&exists); err != nil {
		return false, fmt.Errorf("ExistsByExternalID: %w", err)
	}
	return exists, nil
}

// CountMissingUserHashes 返回 user_hash 为空的用户数量（用于启动时检查）。
func (r *UserSyncRepository) CountMissingUserHashes(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE user_hash IS NULL`).Scan(&count); err != nil {
		return 0, fmt.Errorf("CountMissingUserHashes: %w", err)
	}
	return count, nil
}

// BackfillUserHashes 回填所有 user_hash 为空的用户。
func (r *UserSyncRepository) BackfillUserHashes(ctx context.Context) (int64, error) {
	const batchSize = 500
	var totalCount int64

	for {
		rows, err := r.db.Query(ctx, `SELECT id, external_id FROM users WHERE user_hash IS NULL LIMIT $1`, batchSize)
		if err != nil {
			return totalCount, fmt.Errorf("BackfillUserHashes query: %w", err)
		}

		var batchCount int64
		for rows.Next() {
			var id int64
			var externalID string
			if err := rows.Scan(&id, &externalID); err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes scan: %w", err)
			}
			hash, err := crypto.HMACHashWithKey(externalID, r.hmacKey)
			if err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes hash user %d: %w", id, err)
			}
			if _, err := r.db.Exec(ctx, `UPDATE users SET user_hash = $1 WHERE id = $2 AND user_hash IS NULL`, hash, id); err != nil {
				rows.Close()
				return totalCount, fmt.Errorf("BackfillUserHashes update user %d: %w", id, err)
			}
			batchCount++
			totalCount++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return totalCount, err
		}

		if batchCount < batchSize {
			break
		}
	}

	return totalCount, nil
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
