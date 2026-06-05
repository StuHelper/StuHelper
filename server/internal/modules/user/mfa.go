package user

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	MFAMethodTOTP     = "totp"
	MFAMethodWebAuthn = "webauthn"
	MFAMethodSMS      = "sms"
)

var (
	ErrInvalidMFAMethod            = errors.New("invalid mfa method")
	ErrMFAEnrollmentMethodRequired = errors.New("active mfa enrollment requires at least one method")
	ErrMFAEnrollmentNotFound       = errors.New("mfa enrollment not found")
	ErrMFAUserInvalid              = errors.New("mfa user id is invalid")
	ErrMFARecoveryUserInvalid      = ErrMFAUserInvalid
)

type MFAEnrollment struct {
	UserID                int64
	Active                bool
	Methods               []string
	RecoveryCodesIssuedAt *time.Time
	ResetRequired         bool
	LastEnrolledAt        *time.Time
	LastDisabledAt        *time.Time
	LastResetAt           *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type MFAEnrollmentUpsert struct {
	UserID                int64
	Active                bool
	Methods               []string
	RecoveryCodesIssuedAt *time.Time
	ResetRequired         bool
}

type MFAEnrollmentChangeAction string

const (
	MFAEnrollmentChangeDisable MFAEnrollmentChangeAction = "disable"
	MFAEnrollmentChangeReset   MFAEnrollmentChangeAction = "reset"
)

type MFAEnrollmentStateChange struct {
	UserID        int64
	Active        bool
	Methods       []string
	ResetRequired bool
	ChangedAt     time.Time
	Action        MFAEnrollmentChangeAction
}

func validMFAEnrollmentChange(action MFAEnrollmentChangeAction) bool {
	return action == MFAEnrollmentChangeDisable || action == MFAEnrollmentChangeReset
}

func normalizeMFAMethods(methods []string, active bool) ([]string, error) {
	out := make([]string, 0, len(methods))
	seen := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		item := strings.ToLower(strings.TrimSpace(method))
		if item == "" {
			continue
		}
		if !validMFAMethod(item) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMFAMethod, item)
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	if active && len(out) == 0 {
		return nil, ErrMFAEnrollmentMethodRequired
	}
	return out, nil
}

func validMFAMethod(method string) bool {
	return method == MFAMethodTOTP || method == MFAMethodWebAuthn || method == MFAMethodSMS
}
