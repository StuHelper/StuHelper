package reviewaccess

// SchoolConfig 是 review 访问策略构建所需的最小学校配置投影。
type SchoolConfig struct {
	SchoolID int64
}

// SystemConfig 是 review 访问策略构建所需的最小系统配置投影。
type SystemConfig struct {
	Key   string
	Value string
}

// Subject 是 review 访问控制判定所需的最小用户事实集合。
type Subject struct {
	InternalUserID   int64
	SchoolID         *int64
	StudentVerified  bool
	IdentityVerified bool
}
