package rbac

import "context"

type userExistenceRepo interface {
	UserExists(ctx context.Context, userID int64) (bool, error)
}

type roleCountRepo interface {
	CountRolesByIDs(ctx context.Context, roleIDs []int64) (int, error)
}

type userCountRepo interface {
	CountUsersByIDs(ctx context.Context, userIDs []int64) (int, error)
}

type permissionCountRepo interface {
	CountPermissionsByIDs(ctx context.Context, permIDs []int64) (int, error)
}

func uniquePositiveIDs(ids []int64) ([]int64, bool) {
	if len(ids) == 0 {
		return []int64{}, true
	}

	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, false
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, true
}
