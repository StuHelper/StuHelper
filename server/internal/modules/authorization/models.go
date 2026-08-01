package authorization

import (
	"context"
	"errors"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

type Role string

const (
	RoleSuperAdmin       Role = "super_admin"
	RoleSchoolAdmin      Role = "school_admin"
	RoleSectionAdmin     Role = "section_admin"
	RoleSectionModerator Role = "section_moderator"
	RoleSectionReviewer  Role = "section_reviewer"
)

type GrantSource string

const (
	GrantSourceManual                   GrantSource = "manual"
	GrantSourceCasdoorOrganizationAdmin GrantSource = "casdoor_org_admin"
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
	ErrGrantNotFound              = errors.New("authorization grant not found")
	ErrInvalidGrant               = errors.New("invalid authorization grant")
	ErrTargetUserNotFound         = errors.New("authorization target user not found")
	ErrActorUserNotFound          = errors.New("authorization actor user not found")
	ErrSchoolNotFound             = errors.New("authorization scope school not found")
	ErrProviderManagedRole        = errors.New("authorization role is managed by the identity provider")
	ErrProjectionStale            = errors.New("authorization projection revision is stale")
	ErrProjectionMalformed        = errors.New("authorization projection payload is malformed")
	ErrReconciliationLimit        = errors.New("authorization projection reconciliation threshold exceeded")
	ErrAuthorityCutoverIncomplete = errors.New("authorization authority cutover is incomplete")
	ErrAuthorityCutoverConflict   = errors.New("authorization authority cutover conflicts with existing state")
)

type Grant struct {
	ID                 int64
	SubjectUserID      int64
	Role               Role
	Source             GrantSource
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
	Source        GrantSource
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

type ReconcileAllInput struct {
	Reason      string
	ActorUserID int64
}

type ReconcileAllResult struct {
	Queued int
}

type CasdoorOrganizationAdminSyncInput struct {
	SubjectUserID     int64
	OrganizationAdmin bool
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

type AuthorityCutoverUser struct {
	InternalUserID  int64
	ProviderSubject string
}

type AuthorityCutoverGrant struct {
	SubjectUserID int64
	Role          Role
	Source        GrantSource
	SchoolID      *int64
	SectionID     *string
}

type AuthorityCutoverInput struct {
	SourceDigest string
	Grants       []AuthorityCutoverGrant
}

type AuthorityCutoverStatus struct {
	Completed          bool
	SourceDigest       string
	ImportedGrantCount int
	CompletedAt        *time.Time
}

type AuthorityCutoverResult struct {
	Changed            bool
	SourceDigest       string
	ImportedGrantCount int
	SkippedTupleCount  int
}

type LegacyAuthorityIdentity struct {
	ID                 string
	Owner              string
	Name               string
	OrganizationAdmin  bool
	ForbiddenOrDeleted bool
}

type LegacyAuthoritySnapshot struct {
	Organization string
	Users        []LegacyAuthorityIdentity
	RoleMembers  map[Role][]string
}

type AuthorityCutoverTupleReader interface {
	ReadTuples(ctx context.Context, object, relation string) ([]fga.Tuple, error)
}
