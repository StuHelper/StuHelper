package rbac

import "time"

// Role RBAC 角色
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description,omitempty"`
	IsSystem    bool      `json:"isSystem"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Permission RBAC 权限
type Permission struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Module         string    `json:"module"`
	Action         string    `json:"action"`
	DisplayName    string    `json:"displayName"`
	ScopeSchoolIDs []string  `json:"scopeSchoolIDs,omitempty"` // JSON array, nil = no restriction
	ScopeRoles     []string  `json:"scopeRoles,omitempty"`     // JSON array, nil = no restriction
	CreatedAt      time.Time `json:"createdAt"`
}

// UserGroup 用户组
type UserGroup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description *string   `json:"description,omitempty"`
	MemberCount int       `json:"memberCount"`
	CreatedBy   *int64    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GroupMember 用户组成员详情（join users 表）
type GroupMember struct {
	UserID    int64     `json:"userID"`
	Username  string    `json:"username"`
	Email     *string   `json:"email,omitempty"`
	AvatarURL *string   `json:"avatarURL,omitempty"`
	JoinedAt  time.Time `json:"joinedAt"`
}

// UserPermissionOverride 用户个人权限覆盖
type UserPermissionOverride struct {
	UserID         int64  `json:"userID"`
	PermissionID   int64  `json:"permissionID"`
	PermissionName string `json:"permissionName"`
	Granted        bool   `json:"granted"`
}

// EffectivePermission 用户最终生效权限
type EffectivePermission struct {
	PermissionID int64  `json:"permissionID"`
	Name         string `json:"name"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Source       string `json:"source"` // "role", "group", "override"
	Granted      bool   `json:"granted"`
}
