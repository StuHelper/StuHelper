package user

import (
	"context"
	"fmt"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

// AuthorityCutoverIdentity is the narrow identity-link record exposed to the
// one-time authorization migration command. Provider subjects stay owned by
// the user domain and never become a business authorization key.
type AuthorityCutoverIdentity struct {
	InternalUserID  int64
	ProviderSubject string
}

// AuthorityCutoverRepository owns the one-time shadow-user identity scan. It
// intentionally has no mutation methods and is not part of the runtime authz
// request path.
type AuthorityCutoverRepository struct {
	db *db.DB
}

func NewAuthorityCutoverRepository(database *db.DB) *AuthorityCutoverRepository {
	if database == nil {
		panic("user.NewAuthorityCutoverRepository: database is required")
	}
	return &AuthorityCutoverRepository{db: database}
}

func (r *AuthorityCutoverRepository) ListLinkedIdentities(
	ctx context.Context,
) ([]AuthorityCutoverIdentity, error) {
	ctx = withDBTable(ctx, "users")
	rows, err := r.db.Query(ctx, `
		SELECT id, casdoor_subject
		FROM users
		WHERE btrim(casdoor_subject) <> ''
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list authority cutover identities: %w", err)
	}
	defer rows.Close()

	identities := make([]AuthorityCutoverIdentity, 0)
	for rows.Next() {
		var identity AuthorityCutoverIdentity
		if err := rows.Scan(&identity.InternalUserID, &identity.ProviderSubject); err != nil {
			return nil, fmt.Errorf("scan authority cutover identity: %w", err)
		}
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list authority cutover identity rows: %w", err)
	}
	return identities, nil
}
