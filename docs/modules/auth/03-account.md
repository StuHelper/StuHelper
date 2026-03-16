# Account Synchronization

The account model revolves around Casdoor users and the local `users` table. Casdoor provides external identity; the application side stores local user records needed for business operations.

## Data Sync

Both login callback and `/auth/me` call `UpsertUser` in `user_sync.go`:

| Field | Source |
| --- | --- |
| `external_id` | Casdoor `id` |
| `username` | Casdoor `name` |
| `email` | Casdoor `email` |
| `avatar_url` | Casdoor `avatar` |

## Table Relationships

| Table | Purpose |
| --- | --- |
| `users` | Local basic account records |
| `user_identities` | Identity verification data |
| `user_profiles` | Student verification data, school, phone |
| `user_roles` / `user_permissions` / `user_group_*` | Application authorization relationships |

## User Identifiers

| Identifier | Description |
| --- | --- |
| `users.external_id` | Stable external ID aligned with Casdoor |
| `users.username` | Username after login |
| `user_identities.person_uid` | Stable matching identifier derived from government ID |
| `user_profiles.active_student_id` | Currently active student ID |

## Account Flow

```text
Casdoor user → auth callback → users upsert → /auth/me → frontend state initialization
```

## Sync Logic

```go
// Simplified sync logic
func (s *Service) UpsertUser(ctx context.Context, casdoorUser *CasdoorUser) (*User, error) {
    user := &User{
        ExternalID: casdoorUser.ID,
        Username:   casdoorUser.Name,
        Email:      casdoorUser.Email,
        AvatarURL:  casdoorUser.Avatar,
    }

    // Upsert: insert if not exists, update if exists
    if err := s.repo.UpsertUser(ctx, user); err != nil {
        return nil, fmt.Errorf("upsert user: %w", err)
    }

    return user, nil
}
```
