package studentverification

import (
	"context"
	"encoding/json"
	"time"
)

type Method string

const (
	MethodRealNameIdentityCheck   Method = "real_name_identity_check"
	MethodSchoolSSO               Method = "school_sso"
	MethodStudentEmailOutboundOTP Method = "student_email_outbound_otp"
	MethodStudentEmailInbound     Method = "student_email_inbound_challenge"
	MethodManualMaterialReview    Method = "manual_material_review"
)

type ApplicationStatus string

const (
	ApplicationCreated             ApplicationStatus = "created"
	ApplicationInProgress          ApplicationStatus = "in_progress"
	ApplicationPendingManualReview ApplicationStatus = "pending_manual_review"
	ApplicationApproved            ApplicationStatus = "approved"
	ApplicationRejected            ApplicationStatus = "rejected"
	ApplicationCancelled           ApplicationStatus = "cancelled"
	ApplicationExpired             ApplicationStatus = "expired"
)

type CredentialStatus string

const (
	CredentialPending        CredentialStatus = "pending"
	CredentialActive         CredentialStatus = "active"
	CredentialReviewRequired CredentialStatus = "review_required"
	CredentialExpired        CredentialStatus = "expired"
	CredentialRevoked        CredentialStatus = "revoked"
	CredentialRejected       CredentialStatus = "rejected"
)

type VerificationField struct {
	Key          string                    `json:"key"`
	Label        string                    `json:"label"`
	HelpText     string                    `json:"helpText,omitempty"`
	InputType    string                    `json:"inputType"`
	Autocomplete string                    `json:"autocomplete,omitempty"`
	Required     bool                      `json:"required"`
	MaxLength    *int                      `json:"maxLength,omitempty"`
	Options      []VerificationFieldOption `json:"options,omitempty"`
}

type VerificationFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type PrivacyNotice struct {
	Version          string   `json:"version"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	DataCategories   []string `json:"dataCategories"`
	RetentionSummary string   `json:"retentionSummary"`
	RightsURL        string   `json:"rightsUrl,omitempty"`
}

type MethodCapability struct {
	Method          Method              `json:"method"`
	DisplayName     string              `json:"displayName"`
	Description     string              `json:"description"`
	Availability    string              `json:"availability"`
	UnavailableCode string              `json:"unavailableCode,omitempty"`
	FormFields      []VerificationField `json:"formFields"`
	PrivacyNotice   *PrivacyNotice      `json:"privacyNotice"`
	adapterID       string
}

type VerificationSchool struct {
	ID       int64              `json:"-"`
	Code     string             `json:"code"`
	Name     string             `json:"name"`
	Location string             `json:"location,omitempty"`
	Methods  []MethodCapability `json:"methods"`
}

type MethodConfig struct {
	SchoolID             int64
	SchoolCode           string
	SchoolName           string
	SchoolAdapterID      string
	Method               Method
	AdapterID            string
	AdapterVersion       string
	RosterDependency     string
	PrivacyNoticeVersion string
	PublicFormSchema     json.RawMessage
	StudentIDPolicy      json.RawMessage
	EnrollmentPolicy     json.RawMessage
	ConditionalPolicy    json.RawMessage
	RiskPolicy           json.RawMessage
	EmailDomains         []string
	ConnectorOperation   *string
	SnapshotHardExpiry   time.Duration
	CredentialTTL        *time.Duration
	HealthStatus         string
}

type Application struct {
	ID                   string            `json:"id"`
	UserID               int64             `json:"-"`
	SchoolID             int64             `json:"-"`
	SchoolCode           string            `json:"-"`
	SchoolName           string            `json:"-"`
	Status               ApplicationStatus `json:"status"`
	CurrentMethod        *Method           `json:"currentMethod"`
	PrivacyNoticeVersion *string           `json:"-"`
	ConsentedAt          *time.Time        `json:"-"`
	TerminalCode         *string           `json:"terminalCode,omitempty"`
	Revision             int64             `json:"revision"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	ExpiresAt            time.Time         `json:"expiresAt"`
	CompletedAt          *time.Time        `json:"-"`
	Credential           *Credential       `json:"credential"`
}

type ApplicationView struct {
	ID            string            `json:"id"`
	School        SchoolReference   `json:"school"`
	Status        ApplicationStatus `json:"status"`
	CurrentMethod *Method           `json:"currentMethod"`
	TerminalCode  *string           `json:"terminalCode,omitempty"`
	Revision      int64             `json:"revision"`
	NextActions   []string          `json:"nextActions"`
	Credential    *Credential       `json:"credential"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	ExpiresAt     time.Time         `json:"expiresAt"`
}

type SchoolReference struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Credential struct {
	ID               string           `json:"id"`
	UserID           int64            `json:"-"`
	SchoolID         int64            `json:"-"`
	SchoolCode       string           `json:"schoolCode"`
	SchoolName       string           `json:"schoolName"`
	Method           Method           `json:"method"`
	Status           CredentialStatus `json:"status"`
	CredentialClass  string           `json:"credentialClass"`
	SubjectHash      string           `json:"-"`
	SubjectDisplay   string           `json:"subjectDisplay"`
	EnrollmentID     *string          `json:"-"`
	RosterDependency string           `json:"-"`
	Assurance        string           `json:"-"`
	VerifiedAt       time.Time        `json:"verifiedAt"`
	ExpiresAt        *time.Time       `json:"expiresAt"`
	ReviewRequiredAt *time.Time       `json:"reviewRequiredAt"`
	Revision         int64            `json:"revision"`
}

type Eligibility struct {
	Eligible          bool       `json:"eligible"`
	SchoolCode        string     `json:"schoolCode"`
	CredentialClass   *string    `json:"credentialClass"`
	CredentialMethods []Method   `json:"credentialMethods"`
	ExpiresAt         *time.Time `json:"expiresAt"`
	EvaluatedAt       time.Time  `json:"evaluatedAt"`
	Revision          int64      `json:"revision"`
}

// CurrentStudentStatus is a minimal cross-domain projection for account and
// authorization surfaces that do not have a school in their request context.
// It deliberately omits identity evidence and raw student identifiers.
type CurrentStudentStatus struct {
	Eligible        bool
	SchoolID        *int64
	CredentialClass *string
}

type RosterRecord struct {
	SnapshotID           string
	SnapshotRevision     int64
	SourceCutoffAt       time.Time
	StudentIDHash        string
	NameHash             string
	DocumentType         *string
	DocumentNumberHash   *string
	EligibilityStatus    string
	EligibilityCode      string
	EncryptionKeyVersion int
	HMACKeyVersion       int
}

type RosterSnapshotState struct {
	SnapshotID       string
	SnapshotRevision int64
	SourceCutoffAt   time.Time
}

type EnrollmentSubject struct {
	ID             string
	UserID         int64
	SchoolID       int64
	SubjectHash    string
	StudentIDHash  string
	StudentDisplay string
	BindingStatus  string
}

type CreateApplicationInput struct {
	UserID            int64
	SchoolCode        string
	ContinuationToken string
}

type VerifyRealNameInput struct {
	UserID               int64
	ApplicationID        string
	StudentID            string
	Name                 string
	DocumentNumber       string
	PrivacyNoticeVersion string
	SensitiveDataConsent bool
}

type StudentEmailIdentityInput struct {
	UserID               int64
	ApplicationID        string
	StudentID            string
	Name                 string
	PrivacyNoticeVersion string
	SensitiveDataConsent bool
}

type VerifyStudentEmailOTPInput struct {
	UserID        int64
	ApplicationID string
	Code          string
}

type VerifySchoolSSOInput struct {
	UserID               int64
	ApplicationID        string
	StudentID            string
	Password             []byte
	PrivacyNoticeVersion string
	SensitiveDataConsent bool
}

type SchoolAccountAuthenticationRequest struct {
	ApplicationID      string
	SchoolID           int64
	AdapterID          string
	AdapterVersion     string
	ConnectorOperation string
	StudentID          string
	Password           []byte
}

type SchoolAccountAuthenticationResult struct {
	AccountSubject string
	StudentID      string
	Attributes     map[string]bool
}

type StudentEmailOTPChallenge struct {
	ApplicationID     string    `json:"applicationID"`
	MaskedEmail       string    `json:"maskedEmail"`
	ExpiresAt         time.Time `json:"expiresAt"`
	ResendAvailable   time.Time `json:"resendAvailableAt"`
	RemainingAttempts int       `json:"remainingAttempts"`
}

type InboundEmailChallenge struct {
	ID                   string     `json:"-"`
	ApplicationID        string     `json:"applicationID"`
	UserID               int64      `json:"-"`
	SchoolID             int64      `json:"-"`
	Status               string     `json:"status"`
	TargetAddress        string     `json:"targetAddress"`
	ExpectedSenderMasked string     `json:"expectedSenderMasked"`
	Subject              string     `json:"subject"`
	ChallengeValue       string     `json:"challengeValue,omitempty"`
	ExpiresAt            time.Time  `json:"expiresAt"`
	VerifiedAt           *time.Time `json:"verifiedAt,omitempty"`
}

type storedInboundEmailChallenge struct {
	InboundEmailChallenge
	ExpectedSenderHash    string
	ChallengeValueEnc     []byte
	ChallengeValueHash    string
	EncryptionKeyVersion  int
	HMACKeyVersion        int
	StudentIDHash         string
	NameHash              string
	EnrollmentSubjectHash string
	StudentIDDisplay      string
	SnapshotID            string
	SnapshotRevision      int64
	PrivacyNoticeVersion  string
	MessageReferenceHash  *string
	CreatedAt             time.Time
}

type InboundEmailEvent struct {
	EventReference string
	EnvelopeFrom   string
	HeaderFrom     string
	Subject        string
	TextBody       string
	SPFPass        bool
	DKIMPass       bool
	DMARCPass      bool
	ReceivedAt     time.Time
}

type InboundEmailTargetResolver interface {
	TargetAddress(ctx context.Context, schoolCode string) (string, error)
}

type PhoneVerificationMethod string

const (
	PhoneMethodRosterMatch PhoneVerificationMethod = "school_roster_phone_match"
	PhoneMethodSMS         PhoneVerificationMethod = "sms_possession"
)

type PhoneOperationKind string

const (
	PhoneOperationBind   PhoneOperationKind = "bind"
	PhoneOperationChange PhoneOperationKind = "change"
	PhoneOperationUnbind PhoneOperationKind = "unbind"
)

type PhoneOperationStatus string

const (
	PhoneOperationPendingVerification   PhoneOperationStatus = "pending_verification"
	PhoneOperationVerificationSucceeded PhoneOperationStatus = "verification_succeeded"
	PhoneOperationCasdoorUpdatePending  PhoneOperationStatus = "casdoor_update_pending"
	PhoneOperationCasdoorUpdated        PhoneOperationStatus = "casdoor_updated"
	PhoneOperationProjectionPending     PhoneOperationStatus = "projection_sync_pending"
	PhoneOperationCompleted             PhoneOperationStatus = "completed"
	PhoneOperationFailed                PhoneOperationStatus = "failed"
	PhoneOperationCancelled             PhoneOperationStatus = "cancelled"
	PhoneOperationExpired               PhoneOperationStatus = "expired"
)

type PhoneStatus struct {
	State                          string                   `json:"state"`
	MaskedPhone                    *string                  `json:"maskedPhone"`
	Method                         *PhoneVerificationMethod `json:"method"`
	VerifiedAt                     *time.Time               `json:"verifiedAt"`
	ExpiresAt                      *time.Time               `json:"expiresAt"`
	PublishingRequirementSatisfied bool                     `json:"publishingRequirementSatisfied"`
	Revision                       int64                    `json:"revision"`
}

type PhoneBindingOperation struct {
	ID                   string                   `json:"id"`
	UserID               int64                    `json:"-"`
	OperationKind        PhoneOperationKind       `json:"operationKind"`
	Status               PhoneOperationStatus     `json:"status"`
	VerificationMethod   *PhoneVerificationMethod `json:"-"`
	TargetPhoneEnc       []byte                   `json:"-"`
	TargetPhoneHash      *string                  `json:"-"`
	TargetPhoneMasked    *string                  `json:"maskedPhone"`
	EncryptionKeyVersion *int                     `json:"-"`
	HMACKeyVersion       *int                     `json:"-"`
	FailureCode          *string                  `json:"-"`
	AttemptCount         int                      `json:"-"`
	SMSResendAvailableAt *time.Time               `json:"smsResendAvailableAt"`
	Revision             int64                    `json:"revision"`
	ExpiresAt            time.Time                `json:"expiresAt"`
	VerifiedAt           *time.Time               `json:"-"`
	CasdoorUpdatedAt     *time.Time               `json:"-"`
	ProjectionSyncedAt   *time.Time               `json:"-"`
	CompletedAt          *time.Time               `json:"-"`
	CreatedAt            time.Time                `json:"-"`
	UpdatedAt            time.Time                `json:"-"`
	VerificationStep     string                   `json:"verificationStep"`
}

type CreatePhoneOperationInput struct {
	UserID     int64
	Kind       PhoneOperationKind
	Phone      string
	SchoolCode string
	StudentID  string
	Name       string
}

type VerifyPhoneSMSInput struct {
	UserID      int64
	OperationID string
	Code        string
}

type PhoneGateEligibility struct {
	Eligible    bool                     `json:"eligible"`
	Method      *PhoneVerificationMethod `json:"method"`
	ExpiresAt   *time.Time               `json:"expiresAt"`
	EvaluatedAt time.Time                `json:"evaluatedAt"`
	Revision    int64                    `json:"revision"`
}

type PhoneRosterEvidence struct {
	SchoolID            int64
	EnrollmentSubjectID string
	SnapshotID          string
	SnapshotRevision    int64
	SourceCutoffAt      time.Time
	HardExpiry          time.Duration
}

type phoneCredentialEvidence struct {
	SchoolID            *int64
	EnrollmentSubjectID *string
	SnapshotID          *string
	SnapshotRevision    *int64
}

type attemptResult struct {
	Status           string
	ResultCode       string
	Method           Method
	AdapterID        string
	AdapterVersion   string
	SnapshotID       *string
	SnapshotRevision *int64
	PrivacyNotice    *string
	ConsentedAt      *time.Time
}

type ManualReviewStatus string

const (
	ManualReviewDraft              ManualReviewStatus = "draft"
	ManualReviewPending            ManualReviewStatus = "pending"
	ManualReviewSupplementRequired ManualReviewStatus = "supplement_required"
	ManualReviewApproved           ManualReviewStatus = "approved"
	ManualReviewRejected           ManualReviewStatus = "rejected"
	ManualReviewCancelled          ManualReviewStatus = "cancelled"
	ManualReviewExpired            ManualReviewStatus = "expired"
)

type ManualMaterialType string

const (
	ManualMaterialCampusCard      ManualMaterialType = "campus_card"
	ManualMaterialStudentCard     ManualMaterialType = "student_card"
	ManualMaterialAdmissionNotice ManualMaterialType = "admission_notice"
	ManualMaterialOtherApproved   ManualMaterialType = "other_approved"
)

type ManualReviewMaterial struct {
	ID            string    `json:"id"`
	ContentType   string    `json:"contentType"`
	SizeBytes     int64     `json:"sizeBytes"`
	Width         int       `json:"width"`
	Height        int       `json:"height"`
	CapturedAt    time.Time `json:"capturedAt"`
	ObjectKey     string    `json:"-"`
	SHA256        string    `json:"-"`
	RetentionAt   time.Time `json:"-"`
	CaptureSource string    `json:"-"`
	FacingMode    string    `json:"-"`
}

type ManualReviewCase struct {
	ID                        string                 `json:"id"`
	ApplicationID             string                 `json:"applicationID"`
	UserID                    int64                  `json:"-"`
	SchoolID                  int64                  `json:"-"`
	SchoolCode                string                 `json:"-"`
	SchoolName                string                 `json:"-"`
	School                    SchoolReference        `json:"school"`
	Status                    ManualReviewStatus     `json:"status"`
	MaterialType              ManualMaterialType     `json:"materialType"`
	ApplicantNameMasked       string                 `json:"applicantNameMasked"`
	StudentIDMasked           string                 `json:"studentIDMasked"`
	EmailMasked               *string                `json:"emailMasked"`
	EmailVerified             bool                   `json:"emailVerified"`
	EmailVerificationRequired bool                   `json:"emailVerificationRequired"`
	EmailVerifiedAt           *time.Time             `json:"-"`
	Materials                 []ManualReviewMaterial `json:"materials"`
	UserVisibleReason         *string                `json:"userVisibleReason"`
	CredentialClass           *string                `json:"credentialClass"`
	CredentialExpiresAt       *time.Time             `json:"credentialExpiresAt"`
	Revision                  int64                  `json:"revision"`
	NextActions               []string               `json:"nextActions"`
	SubmittedAt               *time.Time             `json:"submittedAt"`
	ReviewedAt                *time.Time             `json:"reviewedAt"`
	CreatedAt                 time.Time              `json:"createdAt"`
	UpdatedAt                 time.Time              `json:"updatedAt"`
	FormDataEnc               []byte                 `json:"-"`
	FormDigest                string                 `json:"-"`
	EncryptionKeyVersion      int                    `json:"-"`
	StudentIDHash             string                 `json:"-"`
	EmailHash                 *string                `json:"-"`
	PrivacyNoticeVersion      string                 `json:"-"`
	ConsentedAt               time.Time              `json:"-"`
	ReviewedByUserID          *int64                 `json:"-"`
	InternalRiskNoteEnc       []byte                 `json:"-"`
	CredentialID              *string                `json:"-"`
}

type UpsertManualReviewInput struct {
	UserID               int64
	ApplicationID        string
	MaterialType         ManualMaterialType
	FormValues           map[string]string
	PrivacyNoticeVersion string
	SensitiveDataConsent bool
}

type ManualCameraCaptureInput struct {
	UserID              int64
	ApplicationID       string
	Token               string
	ContentType         string
	ImageBase64         string
	CaptureSource       string
	RequestedFacingMode string
}

type ManualCameraHandoffStatus string

const (
	ManualHandoffPending  ManualCameraHandoffStatus = "pending"
	ManualHandoffUploaded ManualCameraHandoffStatus = "uploaded"
	ManualHandoffLocked   ManualCameraHandoffStatus = "locked"
	ManualHandoffExpired  ManualCameraHandoffStatus = "expired"
)

type ManualCameraHandoff struct {
	ID               string                    `json:"id"`
	CaseID           string                    `json:"caseID"`
	ApplicationID    string                    `json:"-"`
	UserID           int64                     `json:"-"`
	Status           ManualCameraHandoffStatus `json:"status"`
	MobileURL        string                    `json:"mobileURL,omitempty"`
	ContinueOn       *string                   `json:"continueOn"`
	Material         *ManualReviewMaterial     `json:"material"`
	MaterialID       *string                   `json:"-"`
	ExpiresAt        time.Time                 `json:"expiresAt"`
	UploadedAt       *time.Time                `json:"-"`
	ChosenAt         *time.Time                `json:"-"`
	CreatedAt        time.Time                 `json:"-"`
	MaxMaterialBytes int64                     `json:"maxMaterialBytes"`
}

type ManualReviewDecisionAction string

const (
	ManualDecisionRequestSupplement ManualReviewDecisionAction = "request_supplement"
	ManualDecisionApprove           ManualReviewDecisionAction = "approve"
	ManualDecisionReject            ManualReviewDecisionAction = "reject"
)

type ManualReviewDecisionInput struct {
	CaseID            string
	ReviewerUserID    int64
	Action            ManualReviewDecisionAction
	UserVisibleReason string
	InternalRiskNote  string
	ExpiresInDays     *int
}

type AdminManualReviewDetail struct {
	Case       *ManualReviewCase `json:"case"`
	FormValues map[string]string `json:"formValues"`
}

type ManualMaterialAccess struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type SchoolVerificationSuggestion struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type ManualEmailOTPChallenge struct {
	CaseID            string    `json:"caseID"`
	MaskedEmail       string    `json:"maskedEmail"`
	ExpiresAt         time.Time `json:"expiresAt"`
	ResendAvailableAt time.Time `json:"resendAvailableAt"`
	RemainingAttempts int       `json:"remainingAttempts"`
}
