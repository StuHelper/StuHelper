package user

import "time"

// QQBinding 记录用户与 QQ 账号的绑定关系。
type QQBinding struct {
	UserID    int64
	QQID      string
	BoundAt   time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// QQBindingCode 记录待消费的 QQ 绑定码。
type QQBindingCode struct {
	UserID     int64
	CodeHash   string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GeneratedQQBindingCode 是返回给前端的明文绑定码。
type GeneratedQQBindingCode struct {
	Code      string
	ExpiresAt time.Time
}

// QQVerificationState 描述 QQ 账号关联用户的学生认证状态。
type QQVerificationState string

const (
	QQVerificationStateUnbound         QQVerificationState = "unbound"
	QQVerificationStateBoundUnverified QQVerificationState = "bound_unverified"
	QQVerificationStateVerified        QQVerificationState = "verified"
)

// QQVerificationStatus 是机器人按 QQ 查询到的聚合状态。
type QQVerificationStatus struct {
	QQID                      string
	UserID                    *int64
	BoundAt                   *time.Time
	VerificationState         QQVerificationState
	ProfileVerificationStatus string
	StudentVerified           bool
}
