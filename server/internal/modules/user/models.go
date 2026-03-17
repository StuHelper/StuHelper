package user

import (
	"encoding/json"
	"time"
)

// IdentityRecord 实名认证完整记录（仅用于写入路径：创建/更新认证记录）
// 包含敏感字段 DocNumberEnc 和 PersonUID，不允许用于 JSON 序列化返回
type IdentityRecord struct {
	UserID          int64
	DocType         string
	DocNumberEnc    []byte
	PersonUID       string
	RealName        string
	Verified        bool
	VerifyMethod    *string
	VerifiedAt      *time.Time
	DocPhotoFront   *string
	DocPhotoBack    *string
	DocPhotoSelfie  *string
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IdentityStatus 实名认证状态（用于普通用户查询）
// 不包含 DocNumberEnc 和 PersonUID，安全可序列化
type IdentityStatus struct {
	UserID          int64
	DocType         string
	RealName        string
	Verified        bool
	VerifyMethod    *string
	VerifiedAt      *time.Time
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IdentityReviewItem 实名认证审核列表项（用于管理员列表和审核）
// 不包含 DocNumberEnc 和 PersonUID，包含照片字段供审核
type IdentityReviewItem struct {
	UserID          int64
	DocType         string
	RealName        string
	Verified        bool
	VerifyMethod    *string
	VerifiedAt      *time.Time
	DocPhotoFront   *string
	DocPhotoBack    *string
	DocPhotoSelfie  *string
	RejectionReason *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Profile 学生认证档案
type Profile struct {
	UserID             int64
	SchoolID           *string
	StudentIDs         []string // JSON array
	ActiveStudentID    *string
	ManualFormData     json.RawMessage
	VerificationStatus string
	VerificationMethod *string
	RejectionReason    *string
	ReviewedAt         *time.Time
	Phone              *string
	PhoneVerified      bool
	ConsentGivenAt     *time.Time
	VerifiedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ManualFieldDescriptor manual 模式动态字段定义
type ManualFieldDescriptor struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"`
	Placeholder *string  `json:"placeholder,omitempty"`
}

// SchoolConfig 学校认证配置
type SchoolConfig struct {
	SchoolID           string
	SchoolName         string
	VerificationMethod string
	LDAPConfig         json.RawMessage
	AcademicDBTable    *string
	ConsentText        *string
	ManualFormFields   json.RawMessage
	Enabled            bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// AcademicStudent 教务系统学生记录
type AcademicStudent struct {
	XH       string
	XM       *string
	SFZJLXDM *string
	SFZJH    *string
	YXDM     *string
	ZYDM     *string
	BJDM     *string
	XZNJ     *string
	RXNJ     *string
	PYCCDM   *string
	XSLBDM   *string
	SJH      *string
	DZXX     *string
	XJZTDM   *string
	SFZX     *string
	SFZJ     *string
	SyncedAt time.Time
}

// SystemConfig 系统配置项
type SystemConfig struct {
	Key         string
	Value       string
	Description *string
	UpdatedAt   time.Time
}
