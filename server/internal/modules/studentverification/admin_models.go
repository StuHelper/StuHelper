package studentverification

import (
	"context"
	"encoding/json"
	"time"
)

// AdminVerificationSchoolConfig is the non-secret administration projection
// for a school's verification allowlist. Directory rows without this profile
// are deliberately absent from this model.
type AdminVerificationSchoolConfig struct {
	SchoolID                    int64                           `json:"-"`
	SchoolCode                  string                          `json:"schoolCode"`
	SchoolName                  string                          `json:"schoolName"`
	Location                    *string                         `json:"location"`
	AdapterID                   string                          `json:"adapterID"`
	AdapterVersion              string                          `json:"adapterVersion"`
	EmailDomains                []string                        `json:"emailDomains"`
	StudentIDPolicy             map[string]any                  `json:"studentIDPolicy"`
	NameMatchPolicy             map[string]any                  `json:"nameMatchPolicy"`
	EnrollmentPolicy            map[string]any                  `json:"enrollmentPolicy"`
	ManualFormSchema            map[string]any                  `json:"manualFormSchema"`
	SnapshotSyncIntervalSeconds int                             `json:"snapshotSyncIntervalSeconds"`
	SnapshotWarningAfterSeconds int                             `json:"snapshotWarningAfterSeconds"`
	SnapshotHardExpirySeconds   int                             `json:"snapshotHardExpirySeconds"`
	SnapshotGraceSeconds        int                             `json:"snapshotGraceSeconds"`
	SnapshotAutoActivate        bool                            `json:"snapshotAutoActivate"`
	Enabled                     bool                            `json:"enabled"`
	ValidationStatus            string                          `json:"validationStatus"`
	ValidationCode              *string                         `json:"validationCode"`
	ConfigRevision              int64                           `json:"configRevision"`
	UpdatedAt                   time.Time                       `json:"updatedAt"`
	Methods                     []AdminVerificationMethodConfig `json:"methods"`
	studentIDPolicyRaw          json.RawMessage
	nameMatchPolicyRaw          json.RawMessage
	enrollmentPolicyRaw         json.RawMessage
	manualFormSchemaRaw         json.RawMessage
}

type AdminVerificationMethodConfig struct {
	SchoolID              int64          `json:"-"`
	SchoolCode            string         `json:"-"`
	Method                Method         `json:"method"`
	DisplayName           string         `json:"displayName"`
	Description           string         `json:"description"`
	AdapterID             string         `json:"adapterID"`
	AdapterVersion        string         `json:"adapterVersion"`
	RosterDependency      string         `json:"rosterDependency"`
	ConditionalPolicy     map[string]any `json:"conditionalPolicy"`
	PublicFormSchema      map[string]any `json:"publicFormSchema"`
	RiskPolicy            map[string]any `json:"riskPolicy"`
	CredentialTTLSeconds  *int           `json:"credentialTTLSeconds"`
	ConnectorOperationKey *string        `json:"connectorOperationKey"`
	SecretConfigured      bool           `json:"secretConfigured"`
	PrivacyNoticeVersion  *string        `json:"privacyNoticeVersion"`
	PrivacyNotice         map[string]any `json:"privacyNotice"`
	Enabled               bool           `json:"enabled"`
	ValidationStatus      string         `json:"validationStatus"`
	ValidationCode        *string        `json:"validationCode"`
	HealthStatus          string         `json:"healthStatus"`
	HealthCode            *string        `json:"healthCode"`
	HealthCheckedAt       *time.Time     `json:"healthCheckedAt"`
	ConfigRevision        int64          `json:"configRevision"`
	UpdatedAt             time.Time      `json:"updatedAt"`
	conditionalPolicyRaw  json.RawMessage
	publicFormSchemaRaw   json.RawMessage
	riskPolicyRaw         json.RawMessage
	privacyNoticeRaw      json.RawMessage
}

type UpdateAdminVerificationSchoolConfigInput struct {
	SchoolCode                  string
	ActorUserID                 int64
	AdapterID                   string
	AdapterVersion              string
	EmailDomains                []string
	StudentIDPolicy             map[string]any
	NameMatchPolicy             map[string]any
	EnrollmentPolicy            map[string]any
	ManualFormSchema            map[string]any
	SnapshotSyncIntervalSeconds int
	SnapshotWarningAfterSeconds int
	SnapshotHardExpirySeconds   int
	SnapshotGraceSeconds        int
	SnapshotAutoActivate        bool
	ExpectedRevision            int64
	Reason                      string
}

// CreateAdminVerificationSchoolConfigInput adds an existing directory school
// to the verification allowlist as a disabled, pending-validation profile.
// A directory row alone never becomes an enabled verification school.
type CreateAdminVerificationSchoolConfigInput struct {
	SchoolCode                  string
	ActorUserID                 int64
	AdapterID                   string
	AdapterVersion              string
	EmailDomains                []string
	StudentIDPolicy             map[string]any
	NameMatchPolicy             map[string]any
	EnrollmentPolicy            map[string]any
	ManualFormSchema            map[string]any
	SnapshotSyncIntervalSeconds int
	SnapshotWarningAfterSeconds int
	SnapshotHardExpirySeconds   int
	SnapshotGraceSeconds        int
	SnapshotAutoActivate        bool
	Reason                      string
}

type UpdateAdminVerificationMethodConfigInput struct {
	SchoolCode            string
	Method                Method
	ActorUserID           int64
	DisplayName           string
	Description           string
	AdapterID             string
	AdapterVersion        string
	RosterDependency      string
	ConditionalPolicy     map[string]any
	PublicFormSchema      map[string]any
	RiskPolicy            map[string]any
	CredentialTTLSeconds  *int
	ConnectorOperationKey *string
	PrivacyNoticeVersion  *string
	PrivacyNotice         map[string]any
	ExpectedRevision      int64
	Reason                string
}

type ValidateAdminVerificationConfigInput struct {
	SchoolCode       string
	Method           *Method
	ActorUserID      int64
	Enable           bool
	ExpectedRevision int64
	Reason           string
}

type AdminStudentCredential struct {
	ID               string           `json:"id"`
	UserID           int64            `json:"userID"`
	SchoolID         int64            `json:"-"`
	SchoolCode       string           `json:"schoolCode"`
	SchoolName       string           `json:"schoolName"`
	Method           Method           `json:"method"`
	Status           CredentialStatus `json:"status"`
	CredentialClass  string           `json:"credentialClass"`
	SubjectDisplay   string           `json:"subjectDisplay"`
	RosterDependency string           `json:"rosterDependency"`
	Assurance        string           `json:"assurance"`
	VerifiedAt       time.Time        `json:"verifiedAt"`
	ExpiresAt        *time.Time       `json:"expiresAt"`
	ReviewRequiredAt *time.Time       `json:"reviewRequiredAt"`
	RevokedReason    *string          `json:"revokedReason"`
	Revision         int64            `json:"revision"`
}

type AdminCredentialRevokeInput struct {
	CredentialID     string
	ActorUserID      int64
	ExpectedRevision int64
	Reason           string
}

type AdminStudentSubjectConflict struct {
	ID              string     `json:"id"`
	SchoolID        int64      `json:"-"`
	SchoolCode      string     `json:"schoolCode"`
	SchoolName      string     `json:"schoolName"`
	ClaimantUserID  int64      `json:"claimantUserID"`
	IncumbentUserID *int64     `json:"incumbentUserID"`
	ApplicationID   *string    `json:"applicationID"`
	Status          string     `json:"status"`
	ReasonCode      string     `json:"reasonCode"`
	ResolutionCode  *string    `json:"resolutionCode"`
	ResolvedAt      *time.Time `json:"resolvedAt"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	subjectHash     string
}

type AdminSubjectConflictDecisionInput struct {
	ConflictID  string
	ActorUserID int64
	Action      string
	Reason      string
}

type AdminCampusConnectorHealth struct {
	ID                       string                                `json:"id"`
	DisplayName              string                                `json:"displayName"`
	Status                   string                                `json:"status"`
	ProtocolVersion          string                                `json:"protocolVersion"`
	SoftwareVersion          string                                `json:"softwareVersion"`
	CertificateNotAfter      time.Time                             `json:"certificateNotAfter"`
	MaxConcurrency           int                                   `json:"maxConcurrency"`
	HeartbeatIntervalSeconds int                                   `json:"heartbeatIntervalSeconds"`
	LastHeartbeatAt          *time.Time                            `json:"lastHeartbeatAt"`
	LastHealthCode           *string                               `json:"lastHealthCode"`
	Revision                 int64                                 `json:"revision"`
	Operations               []AdminCampusConnectorOperationHealth `json:"operations"`
}

type AdminCampusConnectorOperationHealth struct {
	SchoolCode       string     `json:"schoolCode"`
	OperationKey     string     `json:"operationKey"`
	OperationType    string     `json:"operationType"`
	AdapterID        string     `json:"adapterID"`
	AdapterVersion   string     `json:"adapterVersion"`
	Enabled          bool       `json:"enabled"`
	ValidationStatus string     `json:"validationStatus"`
	HealthStatus     string     `json:"healthStatus"`
	HealthCode       *string    `json:"healthCode"`
	HealthCheckedAt  *time.Time `json:"healthCheckedAt"`
	ConfigRevision   int64      `json:"configRevision"`
}

// AdminRosterSyncRequest is the payload-free administration projection for a
// persistent campus-connector command.  It never exposes an Oracle endpoint,
// SQL, connector target, credential, or secret reference.
type AdminRosterSyncRequest struct {
	ID                  string     `json:"id"`
	SchoolCode          string     `json:"schoolCode"`
	OperationKey        string     `json:"operationKey"`
	AdapterID           string     `json:"adapterID"`
	AdapterVersion      string     `json:"adapterVersion"`
	Status              string     `json:"status"`
	ResultCode          *string    `json:"resultCode"`
	RequestedByUserID   int64      `json:"requestedByUserID"`
	Reason              string     `json:"reason"`
	DeadlineAt          time.Time  `json:"deadlineAt"`
	ClaimedAt           *time.Time `json:"claimedAt"`
	ClaimAttempts       int        `json:"claimAttempts"`
	CompletedAt         *time.Time `json:"completedAt"`
	LatencyMilliseconds *int       `json:"latencyMilliseconds"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type AdminRosterSyncInput struct {
	SchoolCode  string
	ActorUserID int64
	Reason      string
}

type AdminRosterSyncCoordinator interface {
	RequestRosterSync(context.Context, AdminRosterSyncInput) (*AdminRosterSyncRequest, error)
	ListRosterSyncRequests(context.Context, string, int) ([]AdminRosterSyncRequest, error)
}
