package authorization

import (
	"errors"
	"time"
)

type Role string

const (
	RoleSuperAdmin       Role = "super_admin"
	RoleSchoolAdmin      Role = "school_admin"
	RoleSectionAdmin     Role = "section_admin"
	RoleSectionModerator Role = "section_moderator"
	RoleSectionReviewer  Role = "section_reviewer"
)

type DesiredState string

const (
	DesiredGranted DesiredState = "granted"
	DesiredRevoked DesiredState = "revoked"
)

type ProjectionStatus string

const (
	ProjectionPending ProjectionStatus = "pending"
	ProjectionApplied ProjectionStatus = "applied"
	ProjectionFailed  ProjectionStatus = "failed"
)

const (
	ProjectionJobType = "authorization_grant_projection"
	maxReasonLength   = 500
)

var (
	ErrGrantNotFound       = errors.New("authorization grant not found")
	ErrInvalidGrant        = errors.New("invalid authorization grant")
	ErrTargetUserNotFound  = errors.New("authorization target user not found")
	ErrActorUserNotFound   = errors.New("authorization actor user not found")
	ErrSchoolNotFound      = errors.New("authorization scope school not found")
	ErrLastSuperAdmin      = errors.New("cannot revoke the last active super admin")
	ErrProjectionStale     = errors.New("authorization projection revision is stale")
	ErrProjectionMalformed = errors.New("authorization projection payload is malformed")
)

type Grant struct {
	ID                 int64
	SubjectUserID      int64
	Role               Role
	SchoolID           *int64
	SectionID          *string
	DesiredState       DesiredState
	ProjectionStatus   ProjectionStatus
	Revision           int64
	Reason             string
	CreatedByUserID    *int64
	UpdatedByUserID    *int64
	ActivatedAt        *time.Time
	RevokedAt          *time.Time
	ProjectedAt        *time.Time
	LastError          *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	SubjectUsername    string
	SubjectDisplayName string
}

type CreateGrantInput struct {
	SubjectUserID int64
	Role          Role
	SchoolID      *int64
	SectionID     *string
	Reason        string
	ActorUserID   int64
}

type RevokeGrantInput struct {
	GrantID     int64
	Reason      string
	ActorUserID int64
}

type ReconcileGrantInput struct {
	GrantID     int64
	Reason      string
	ActorUserID int64
}

type ListGrantsFilter struct {
	SubjectUserID *int64
	Role          *Role
	DesiredState  *DesiredState
	Projection    *ProjectionStatus
	Limit         int
	Offset        int
}

type GrantList struct {
	Items []Grant
	Total int
}

type ProjectionPayload struct {
	GrantID      int64        `json:"grantId"`
	Revision     int64        `json:"revision"`
	DesiredState DesiredState `json:"desiredState"`
}

type AccessSnapshot struct {
	InternalUserID int64
	Roles          []string
	RoleScopes     map[string][]string
}

type MutationResult struct {
	Grant   Grant
	Changed bool
}
