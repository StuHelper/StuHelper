package user

import "time"

// Identity 实名认证记录
type Identity struct {
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

// Profile 学生认证档案
type Profile struct {
	UserID             int64
	SchoolID           *string
	StudentIDs         []string // JSON array
	ActiveStudentID    *string
	VerificationStatus string
	VerificationMethod *string
	Phone              *string
	PhoneVerified      bool
	ConsentGivenAt     *time.Time
	VerifiedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SchoolConfig 学校认证配置
type SchoolConfig struct {
	SchoolID           string
	SchoolName         string
	VerificationMethod string
	LDAPConfig         []byte // encrypted
	AcademicDBTable    *string
	ConsentText        *string
	ManualFormFields   []byte // JSONB
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
