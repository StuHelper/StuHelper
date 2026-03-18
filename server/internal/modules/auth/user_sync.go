package auth

import (
	"context"
	"fmt"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type UserSyncInput struct {
	ExternalID string
	Username   string
	Email      string
	AvatarURL  *string
}

type CapabilityReader interface {
	GetUserCapabilitySnapshot(ctx context.Context, externalID string) (capability.UserAccessSnapshot, error)
}

type UserSyncRepo interface {
	UpsertUser(ctx context.Context, input UserSyncInput) error
}

type UserSyncRepository struct {
	db *db.DB
}

func NewUserSyncRepository(database *db.DB) *UserSyncRepository {
	return &UserSyncRepository{db: database}
}

func (r *UserSyncRepository) UpsertUser(ctx context.Context, input UserSyncInput) error {
	if input.ExternalID == "" || input.Username == "" {
		return fmt.Errorf("UpsertUser: externalID and username are required")
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO users (external_id, username, email, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (external_id) DO UPDATE SET
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			avatar_url = COALESCE(EXCLUDED.avatar_url, users.avatar_url),
			updated_at = NOW()
	`, input.ExternalID, input.Username, emptyToNil(input.Email), input.AvatarURL)
	if err != nil {
		return fmt.Errorf("UpsertUser: %w", err)
	}
	return nil
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
