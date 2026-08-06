package user

import "time"

type qqBindingResponse struct {
	UserID    int64     `json:"userID"`
	QQID      string    `json:"qqID"`
	BoundAt   time.Time `json:"boundAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type qqBindingCodeResponse struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type qqVerificationStatusResponse struct {
	QQID              string              `json:"qqID"`
	UserID            *int64              `json:"userID"`
	BoundAt           *time.Time          `json:"boundAt"`
	VerificationState QQVerificationState `json:"verificationState"`
	StudentVerified   bool                `json:"studentVerified"`
}

func qqBindingToJSON(binding *QQBinding) qqBindingResponse {
	return qqBindingResponse{
		UserID:    binding.UserID,
		QQID:      binding.QQID,
		BoundAt:   binding.BoundAt,
		CreatedAt: binding.CreatedAt,
		UpdatedAt: binding.UpdatedAt,
	}
}

func qqVerificationStatusToJSON(status *QQVerificationStatus) qqVerificationStatusResponse {
	return qqVerificationStatusResponse{
		QQID:              status.QQID,
		UserID:            status.UserID,
		BoundAt:           status.BoundAt,
		VerificationState: status.VerificationState,
		StudentVerified:   status.StudentVerified,
	}
}
