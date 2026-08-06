package studentverification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var (
	ErrRosterImportUnavailable = errors.New("roster import unavailable")
	ErrRosterPolicyInvalid     = errors.New("roster policy invalid")
	ErrRosterQualityFailed     = errors.New("roster quality gate failed")
	ErrRosterSnapshotNotFound  = errors.New("roster snapshot not found")
	ErrRosterSnapshotState     = errors.New("roster snapshot state conflict")
	ErrRosterSourceRegression  = errors.New("roster source time regression")
	ErrRosterSourceConflict    = errors.New("roster source version content conflict")
)

type RosterImportRecord struct {
	StudentID          string     `json:"studentID"`
	Name               string     `json:"name"`
	DocumentType       string     `json:"documentType,omitempty"`
	DocumentNumber     string     `json:"documentNumber,omitempty"`
	Phone              string     `json:"phone,omitempty"`
	StudentStatus      string     `json:"studentStatus,omitempty"`
	OnCampusStatus     string     `json:"onCampusStatus,omitempty"`
	RegistrationStatus string     `json:"registrationStatus,omitempty"`
	EducationLevel     string     `json:"educationLevel,omitempty"`
	StudentCategory    string     `json:"studentCategory,omitempty"`
	EnrollmentYear     *int       `json:"enrollmentYear,omitempty"`
	ValidFrom          *time.Time `json:"validFrom,omitempty"`
	ValidUntil         *time.Time `json:"validUntil,omitempty"`
	CurrentMarker      *bool      `json:"currentMarker,omitempty"`
	EligibilityCode    string     `json:"eligibilityCode"`
	SourceUpdatedAt    *time.Time `json:"sourceUpdatedAt,omitempty"`
}

type FullRosterImportInput struct {
	SchoolCode         string                      `json:"schoolCode"`
	SourceKind         string                      `json:"sourceKind"`
	SourceVersion      string                      `json:"sourceVersion"`
	MappingVersion     string                      `json:"mappingVersion"`
	SourceStartedAt    *time.Time                  `json:"sourceStartedAt,omitempty"`
	SourceCutoffAt     time.Time                   `json:"sourceCutoffAt"`
	ConnectorNodeID    *string                     `json:"connectorNodeID,omitempty"`
	SignatureAlgorithm string                      `json:"signatureAlgorithm,omitempty"`
	SignatureKeyID     string                      `json:"signatureKeyID,omitempty"`
	SnapshotSignature  []byte                      `json:"snapshotSignature,omitempty"`
	SourceQuality      *RosterSourceQualitySummary `json:"sourceQuality,omitempty"`
	Records            []RosterImportRecord        `json:"records"`
}

type RosterSourceQualitySummary struct {
	RowsRead              int64 `json:"rowsRead"`
	RecordsEmitted        int64 `json:"recordsEmitted"`
	MissingDocumentNumber int64 `json:"missingDocumentNumber"`
	InvalidDocumentNumber int64 `json:"invalidDocumentNumber"`
	MissingPhone          int64 `json:"missingPhone"`
	InvalidPhone          int64 `json:"invalidPhone"`
	MissingEnrollmentYear int64 `json:"missingEnrollmentYear"`
	InvalidEnrollmentYear int64 `json:"invalidEnrollmentYear"`
}

type RosterSnapshotSwitchInput struct {
	SchoolCode            string
	SnapshotID            string
	ActorUserID           int64
	Reason                string
	AllowSourceRegression bool
}

type rosterSnapshotSwitchCommand struct {
	SchoolCode            string
	SnapshotID            string
	ActorUserID           *int64
	ActorType             string
	Reason                string
	AllowSourceRegression bool
	RequireAutoActivation bool
}

type RosterSnapshot struct {
	ID                 string               `json:"id"`
	SchoolID           int64                `json:"-"`
	SchoolCode         string               `json:"schoolCode"`
	SchoolName         string               `json:"schoolName"`
	SourceKind         string               `json:"sourceKind"`
	SourceVersion      string               `json:"sourceVersion"`
	ImportMode         string               `json:"importMode"`
	SchemaVersion      int                  `json:"schemaVersion"`
	MappingVersion     string               `json:"mappingVersion"`
	Status             string               `json:"status"`
	SourceCutoffAt     time.Time            `json:"sourceCutoffAt"`
	ImportStartedAt    time.Time            `json:"importStartedAt"`
	ImportCompletedAt  *time.Time           `json:"importCompletedAt"`
	ActivatedAt        *time.Time           `json:"activatedAt"`
	RowCount           int64                `json:"rowCount"`
	EligibleRowCount   int64                `json:"eligibleRowCount"`
	DeletedRowCount    int64                `json:"deletedRowCount"`
	Checksum           *string              `json:"checksum"`
	FailureCode        *string              `json:"failureCode"`
	ActivationRevision *int64               `json:"activationRevision"`
	IsCurrent          bool                 `json:"isCurrent"`
	CreatedAt          time.Time            `json:"createdAt"`
	UpdatedAt          time.Time            `json:"updatedAt"`
	QualityChecks      []RosterQualityCheck `json:"qualityChecks"`
}

type RosterQualityCheck struct {
	CheckKey   string         `json:"checkKey"`
	Status     string         `json:"status"`
	Measured   map[string]any `json:"measured"`
	Threshold  map[string]any `json:"threshold"`
	DetailCode string         `json:"detailCode,omitempty"`
	CheckedAt  time.Time      `json:"checkedAt"`
}

type rosterImportConfig struct {
	SchoolID         int64
	SchoolCode       string
	SchoolName       string
	AdapterID        string
	AdapterVersion   string
	EnrollmentPolicy json.RawMessage
}

type rosterPolicy struct {
	KnownEligibilityCodes []string `json:"rosterKnownEligibilityCodes"`
	EligibleCodes         []string `json:"rosterEligibleCodes"`
	MinimumRows           int      `json:"rosterMinimumRows"`
	MaximumRowDeltaRatio  float64  `json:"rosterMaximumRowDeltaRatio"`
	RequireCurrentMarker  bool     `json:"rosterRequireCurrentMarker"`
}

type preparedRosterRecord struct {
	SourceRecordKeyHash string
	StudentIDEnc        []byte
	StudentIDHash       string
	NameEnc             []byte
	NameHash            string
	DocumentType        *string
	DocumentNumberEnc   []byte
	DocumentNumberHash  *string
	PhoneEnc            []byte
	PhoneHash           *string
	StudentStatus       *string
	OnCampusStatus      *string
	RegistrationStatus  *string
	EducationLevel      *string
	StudentCategory     *string
	EnrollmentYear      *int
	ValidFrom           *time.Time
	ValidUntil          *time.Time
	CurrentMarker       *bool
	EligibilityStatus   string
	EligibilityCode     string
	SourceUpdatedAt     *time.Time
	RecordChecksum      string
}

type preparedRosterImport struct {
	Rows          []preparedRosterRecord
	Checksum      string
	EligibleCount int64
	Checks        []RosterQualityCheck
}

type rosterQualityError struct {
	checkKey   string
	detailCode string
}

func (e *rosterQualityError) Error() string { return e.checkKey + ": " + e.detailCode }

func (s *Service) ImportFullRoster(ctx context.Context, input FullRosterImportInput) (*RosterSnapshot, error) {
	if s.rosterCipher == nil || s.rosterEncryptionKeyVersion <= 0 {
		return nil, ErrRosterImportUnavailable
	}
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.SourceKind = strings.TrimSpace(input.SourceKind)
	input.SourceVersion = strings.TrimSpace(input.SourceVersion)
	input.MappingVersion = strings.TrimSpace(input.MappingVersion)
	if !schoolCodePattern.MatchString(input.SchoolCode) ||
		!validRosterSourceKind(input.SourceKind) ||
		input.SourceVersion == "" || len(input.SourceVersion) > 255 ||
		input.MappingVersion == "" || len(input.MappingVersion) > 64 ||
		input.SourceCutoffAt.IsZero() || input.SourceCutoffAt.After(s.now().Add(allowedSnapshotFutureSkew)) {
		return nil, ErrRosterPolicyInvalid
	}
	if input.SourceKind == "campus_connector" {
		if input.ConnectorNodeID == nil || strings.TrimSpace(*input.ConnectorNodeID) == "" ||
			input.SignatureAlgorithm != "Ed25519" || strings.TrimSpace(input.SignatureKeyID) == "" ||
			len(input.SnapshotSignature) == 0 {
			return nil, ErrRosterPolicyInvalid
		}
		if input.SourceQuality == nil || !validRosterSourceQuality(*input.SourceQuality, len(input.Records)) {
			return nil, ErrRosterPolicyInvalid
		}
	} else if input.ConnectorNodeID != nil || input.SignatureAlgorithm != "" ||
		input.SignatureKeyID != "" || len(input.SnapshotSignature) != 0 || input.SourceQuality != nil {
		return nil, ErrRosterPolicyInvalid
	}
	config, err := s.repo.GetRosterImportConfig(ctx, input.SchoolCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSchoolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load roster import configuration: %w", err)
	}
	policy, err := decodeRosterPolicy(config.EnrollmentPolicy)
	if err != nil {
		return nil, err
	}

	now := s.now()
	prepared, prepareErr := s.prepareRosterImport(*config, policy, input.Records, now)
	if prepareErr == nil && input.SourceQuality != nil {
		prepared.Checks = append(prepared.Checks, qualityCheck(
			"source.quality", "passed",
			map[string]any{
				"rowsRead":              input.SourceQuality.RowsRead,
				"recordsEmitted":        input.SourceQuality.RecordsEmitted,
				"missingDocumentNumber": input.SourceQuality.MissingDocumentNumber,
				"invalidDocumentNumber": input.SourceQuality.InvalidDocumentNumber,
				"missingPhone":          input.SourceQuality.MissingPhone,
				"invalidPhone":          input.SourceQuality.InvalidPhone,
				"missingEnrollmentYear": input.SourceQuality.MissingEnrollmentYear,
				"invalidEnrollmentYear": input.SourceQuality.InvalidEnrollmentYear,
			}, nil, "", now,
		))
	}
	checksum, err := rosterImportFingerprint(s.hmacKey, input)
	if err != nil {
		return nil, fmt.Errorf("fingerprint roster import: %w", err)
	}
	if prepareErr == nil {
		checksum = prepared.Checksum
		checks, checkErr := s.evaluateRosterAggregateChecks(
			ctx, *config, policy, input.SourceCutoffAt, prepared, now,
		)
		if checkErr != nil {
			return nil, fmt.Errorf("evaluate roster aggregate checks: %w", checkErr)
		}
		prepared.Checks = append(prepared.Checks, checks...)
	}

	snapshotID, err := newID()
	if err != nil {
		return nil, err
	}
	snapshot := RosterSnapshot{
		ID: snapshotID, SchoolID: config.SchoolID, SchoolCode: config.SchoolCode,
		SchoolName: config.SchoolName, SourceKind: input.SourceKind,
		SourceVersion: input.SourceVersion, ImportMode: "full", SchemaVersion: 1,
		MappingVersion: input.MappingVersion, Status: "staging",
		SourceCutoffAt: input.SourceCutoffAt, ImportStartedAt: now,
		Checksum: &checksum, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.CreateRosterSnapshot(ctx, snapshot, input.SourceStartedAt, input.ConnectorNodeID,
		s.rosterEncryptionKeyVersion, RosterHMACKeyVersion, input.SignatureAlgorithm,
		input.SignatureKeyID, input.SnapshotSignature); err != nil {
		if isUniqueViolation(err, "uq_student_roster_snapshots_source") {
			existing, getErr := s.repo.GetRosterSnapshotBySource(
				ctx, config.SchoolID, input.SourceKind, input.SourceVersion,
			)
			if getErr != nil {
				return nil, getErr
			}
			if existing.Checksum == nil ||
				!hmac.Equal([]byte(*existing.Checksum), []byte(checksum)) {
				return nil, ErrRosterSourceConflict
			}
			if existing.Status == "failed" {
				return existing, ErrRosterQualityFailed
			}
			return existing, nil
		}
		return nil, err
	}

	if prepareErr != nil {
		qualityErr := &rosterQualityError{checkKey: "input.validity", detailCode: "invalid_record"}
		var typedQualityErr *rosterQualityError
		if errors.As(prepareErr, &typedQualityErr) {
			qualityErr = typedQualityErr
		}
		if failErr := s.repo.FailRosterSnapshot(
			ctx, snapshot.ID, qualityErr.checkKey, qualityErr.detailCode, now,
		); failErr != nil {
			return nil, errors.Join(ErrRosterQualityFailed, fmt.Errorf("mark invalid roster snapshot failed: %w", failErr))
		}
		result, getErr := s.repo.GetRosterSnapshot(ctx, snapshot.ID)
		if getErr != nil {
			return nil, getErr
		}
		return result, ErrRosterQualityFailed
	}
	if err := s.repo.StageRosterRecords(ctx, snapshot, prepared, s.rosterEncryptionKeyVersion, RosterHMACKeyVersion, now); err != nil {
		if failErr := s.repo.FailRosterSnapshot(
			ctx, snapshot.ID, "storage.integrity", "record_insert_failed", now,
		); failErr != nil {
			return nil, errors.Join(err, fmt.Errorf("mark failed roster snapshot: %w", failErr))
		}
		return nil, err
	}
	result, err := s.repo.GetRosterSnapshot(ctx, snapshot.ID)
	if err != nil {
		return nil, err
	}
	if result.Status == "failed" {
		return result, ErrRosterQualityFailed
	}
	return result, nil
}

func (s *Service) ActivateRosterSnapshot(
	ctx context.Context,
	input RosterSnapshotSwitchInput,
) (*RosterSnapshot, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.SnapshotID = strings.TrimSpace(input.SnapshotID)
	input.Reason = strings.TrimSpace(input.Reason)
	if !schoolCodePattern.MatchString(input.SchoolCode) || input.SnapshotID == "" ||
		input.ActorUserID <= 0 || len(input.Reason) < 4 || len(input.Reason) > 500 {
		return nil, ErrRosterSnapshotState
	}
	result, err := s.repo.SwitchRosterSnapshot(ctx, input, false, s.now())
	if err != nil {
		return nil, err
	}
	return s.repo.GetRosterSnapshot(ctx, result.ID)
}

func (s *Service) RollbackRosterSnapshot(
	ctx context.Context,
	input RosterSnapshotSwitchInput,
) (*RosterSnapshot, error) {
	input.SchoolCode = strings.TrimSpace(input.SchoolCode)
	input.SnapshotID = strings.TrimSpace(input.SnapshotID)
	input.Reason = strings.TrimSpace(input.Reason)
	if !schoolCodePattern.MatchString(input.SchoolCode) || input.SnapshotID == "" ||
		input.ActorUserID <= 0 || len(input.Reason) < 4 || len(input.Reason) > 500 {
		return nil, ErrRosterSnapshotState
	}
	result, err := s.repo.SwitchRosterSnapshot(ctx, input, true, s.now())
	if err != nil {
		return nil, err
	}
	return s.repo.GetRosterSnapshot(ctx, result.ID)
}

// AutoActivateImportedRosterSnapshot applies the school's explicit automatic
// activation policy. A disabled policy is a successful no-op and returns the
// ready snapshot; there is no fake administrator identity in the audit trail.
func (s *Service) AutoActivateImportedRosterSnapshot(
	ctx context.Context,
	schoolCode string,
	snapshotID string,
) (*RosterSnapshot, error) {
	schoolCode = strings.TrimSpace(schoolCode)
	snapshotID = strings.TrimSpace(snapshotID)
	if !schoolCodePattern.MatchString(schoolCode) || snapshotID == "" {
		return nil, ErrRosterSnapshotState
	}
	return s.repo.AutoActivateRosterSnapshot(ctx, schoolCode, snapshotID, s.now())
}

func (s *Service) ListRosterSnapshots(
	ctx context.Context,
	schoolCode string,
	limit int,
	offset int,
) ([]RosterSnapshot, error) {
	schoolCode = strings.TrimSpace(schoolCode)
	if !schoolCodePattern.MatchString(schoolCode) {
		return nil, ErrSchoolNotFound
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListRosterSnapshots(ctx, schoolCode, limit, offset)
}

func (s *Service) GetRosterSnapshotForSchool(
	ctx context.Context,
	schoolCode string,
	snapshotID string,
) (*RosterSnapshot, error) {
	if !schoolCodePattern.MatchString(strings.TrimSpace(schoolCode)) || strings.TrimSpace(snapshotID) == "" {
		return nil, ErrRosterSnapshotNotFound
	}
	snapshot, err := s.repo.GetRosterSnapshot(ctx, snapshotID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && snapshot.SchoolCode != schoolCode) {
		return nil, ErrRosterSnapshotNotFound
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func decodeRosterPolicy(raw json.RawMessage) (rosterPolicy, error) {
	var policy rosterPolicy
	if len(raw) == 0 || json.Unmarshal(raw, &policy) != nil ||
		len(policy.KnownEligibilityCodes) == 0 || len(policy.EligibleCodes) == 0 ||
		policy.MinimumRows <= 0 || policy.MaximumRowDeltaRatio <= 0 ||
		policy.MaximumRowDeltaRatio > 10 {
		return rosterPolicy{}, ErrRosterPolicyInvalid
	}
	known := make(map[string]struct{}, len(policy.KnownEligibilityCodes))
	for _, code := range policy.KnownEligibilityCodes {
		code = strings.TrimSpace(code)
		if code == "" || len(code) > 100 {
			return rosterPolicy{}, ErrRosterPolicyInvalid
		}
		known[code] = struct{}{}
	}
	for _, code := range policy.EligibleCodes {
		if _, ok := known[strings.TrimSpace(code)]; !ok {
			return rosterPolicy{}, ErrRosterPolicyInvalid
		}
	}
	return policy, nil
}

func validRosterSourceQuality(summary RosterSourceQualitySummary, recordCount int) bool {
	if recordCount < 0 || summary.RowsRead < 0 || summary.RecordsEmitted < 0 ||
		summary.MissingDocumentNumber < 0 || summary.InvalidDocumentNumber < 0 ||
		summary.MissingPhone < 0 || summary.InvalidPhone < 0 ||
		summary.MissingEnrollmentYear < 0 || summary.InvalidEnrollmentYear < 0 {
		return false
	}
	return summary.RowsRead >= int64(recordCount) && summary.RecordsEmitted == int64(recordCount)
}

func (s *Service) prepareRosterImport(
	config rosterImportConfig,
	policy rosterPolicy,
	records []RosterImportRecord,
	now time.Time,
) (preparedRosterImport, error) {
	if config.AdapterID != BUAAAdapterID || config.AdapterVersion == "" {
		return preparedRosterImport{}, ErrRosterImportUnavailable
	}
	knownCodes := make(map[string]struct{}, len(policy.KnownEligibilityCodes))
	for _, code := range policy.KnownEligibilityCodes {
		knownCodes[strings.TrimSpace(code)] = struct{}{}
	}
	eligibleCodes := make(map[string]struct{}, len(policy.EligibleCodes))
	for _, code := range policy.EligibleCodes {
		eligibleCodes[strings.TrimSpace(code)] = struct{}{}
	}
	prepared := preparedRosterImport{Rows: make([]preparedRosterRecord, 0, len(records))}
	seenStudents := make(map[string]struct{}, len(records))
	for _, record := range records {
		row, err := s.prepareBUAARosterRecord(config, policy, knownCodes, eligibleCodes, record, now)
		if err != nil {
			return preparedRosterImport{}, err
		}
		if _, duplicate := seenStudents[row.StudentIDHash]; duplicate {
			return preparedRosterImport{}, &rosterQualityError{
				checkKey: "student_id.unique", detailCode: "duplicate_student_id",
			}
		}
		seenStudents[row.StudentIDHash] = struct{}{}
		if row.EligibilityStatus == "eligible" {
			prepared.EligibleCount++
		}
		prepared.Rows = append(prepared.Rows, row)
	}
	sort.Slice(prepared.Rows, func(i, j int) bool {
		return prepared.Rows[i].StudentIDHash < prepared.Rows[j].StudentIDHash
	})
	hasher := sha256.New()
	for _, row := range prepared.Rows {
		_, _ = hasher.Write([]byte(row.RecordChecksum))
	}
	prepared.Checksum = hex.EncodeToString(hasher.Sum(nil))
	prepared.Checks = []RosterQualityCheck{
		qualityCheck("required_fields.valid", "passed", map[string]any{"rows": len(prepared.Rows)}, nil, "", now),
		qualityCheck("student_id.unique", "passed", map[string]any{"duplicates": 0}, map[string]any{"maximum": 0}, "", now),
		qualityCheck("identity_projection.integrity", "passed", map[string]any{"rows": len(prepared.Rows)}, nil, "", now),
		qualityCheck("eligibility_codes.known", "passed", map[string]any{"unknown": 0}, map[string]any{"maximum": 0}, "", now),
		qualityCheck("snapshot.checksum", "passed", map[string]any{"computed": true}, nil, "", now),
	}
	return prepared, nil
}

func (s *Service) prepareBUAARosterRecord(
	config rosterImportConfig,
	policy rosterPolicy,
	knownCodes map[string]struct{},
	eligibleCodes map[string]struct{},
	record RosterImportRecord,
	_ time.Time,
) (preparedRosterRecord, error) {
	studentID, ok := s.buaa.NormalizeStudentID(record.StudentID)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("required_fields.valid", "student_id_invalid")
	}
	name, ok := s.buaa.NormalizeName(record.Name)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("required_fields.valid", "name_invalid")
	}
	code := strings.TrimSpace(record.EligibilityCode)
	if _, ok := knownCodes[code]; !ok {
		return preparedRosterRecord{}, qualityFailure("eligibility_codes.known", "unknown_eligibility_code")
	}
	if policy.RequireCurrentMarker && record.CurrentMarker == nil {
		return preparedRosterRecord{}, qualityFailure("current_enrollment.deterministic", "current_marker_missing")
	}
	if record.EnrollmentYear != nil && (*record.EnrollmentYear < 1900 || *record.EnrollmentYear > 3000) {
		return preparedRosterRecord{}, qualityFailure("enrollment_year.valid", "enrollment_year_invalid")
	}
	if record.ValidFrom != nil && record.ValidUntil != nil && record.ValidUntil.Before(*record.ValidFrom) {
		return preparedRosterRecord{}, qualityFailure("enrollment_validity.valid", "validity_range_invalid")
	}

	studentHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexStudentID, studentID)
	if err != nil {
		return preparedRosterRecord{}, err
	}
	nameHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexName, name)
	if err != nil {
		return preparedRosterRecord{}, err
	}
	studentEnc, err := s.rosterCipher.Encrypt(studentID)
	if err != nil {
		return preparedRosterRecord{}, qualityFailure("identity_projection.integrity", "student_id_encrypt_failed")
	}
	nameEnc, err := s.rosterCipher.Encrypt(name)
	if err != nil {
		return preparedRosterRecord{}, qualityFailure("identity_projection.integrity", "name_encrypt_failed")
	}
	studentStatus, ok := normalizedOptionalRosterValue(record.StudentStatus)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("status_fields.valid", "student_status_invalid")
	}
	onCampusStatus, ok := normalizedOptionalRosterValue(record.OnCampusStatus)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("status_fields.valid", "on_campus_status_invalid")
	}
	registrationStatus, ok := normalizedOptionalRosterValue(record.RegistrationStatus)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("status_fields.valid", "registration_status_invalid")
	}
	educationLevel, ok := normalizedOptionalRosterValue(record.EducationLevel)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("status_fields.valid", "education_level_invalid")
	}
	studentCategory, ok := normalizedOptionalRosterValue(record.StudentCategory)
	if !ok {
		return preparedRosterRecord{}, qualityFailure("status_fields.valid", "student_category_invalid")
	}
	row := preparedRosterRecord{
		SourceRecordKeyHash: studentHash, StudentIDEnc: studentEnc,
		StudentIDHash: studentHash, NameEnc: nameEnc, NameHash: nameHash,
		StudentStatus: studentStatus, OnCampusStatus: onCampusStatus,
		RegistrationStatus: registrationStatus, EducationLevel: educationLevel,
		StudentCategory: studentCategory,
		EnrollmentYear:  record.EnrollmentYear, ValidFrom: normalizeDatePtr(record.ValidFrom),
		ValidUntil: normalizeDatePtr(record.ValidUntil), CurrentMarker: record.CurrentMarker,
		EligibilityCode: code, SourceUpdatedAt: record.SourceUpdatedAt,
	}
	if _, eligible := eligibleCodes[code]; eligible {
		row.EligibilityStatus = "eligible"
	} else {
		row.EligibilityStatus = "ineligible"
	}

	documentType := strings.TrimSpace(record.DocumentType)
	documentNumber := strings.TrimSpace(record.DocumentNumber)
	if documentType != "" || documentNumber != "" {
		normalizedDocument, valid := s.buaa.NormalizeMainlandDocumentNumber(documentNumber)
		if !valid || !s.buaa.SupportsMainlandDocumentType(documentType, config.EnrollmentPolicy) {
			return preparedRosterRecord{}, qualityFailure("document.valid", "document_invalid")
		}
		documentHash, err := ComputeRosterBlindIndex(
			s.hmacKey, config.SchoolID, BlindIndexDocumentNumber, normalizedDocument,
		)
		if err != nil {
			return preparedRosterRecord{}, err
		}
		documentEnc, err := s.rosterCipher.Encrypt(normalizedDocument)
		if err != nil {
			return preparedRosterRecord{}, qualityFailure("identity_projection.integrity", "document_encrypt_failed")
		}
		row.DocumentType = &documentType
		row.DocumentNumberHash = &documentHash
		row.DocumentNumberEnc = documentEnc
	}
	if strings.TrimSpace(record.Phone) != "" {
		mainland, _, valid := normalizeMainlandPhone(record.Phone)
		if !valid {
			return preparedRosterRecord{}, qualityFailure("phone.valid", "phone_invalid")
		}
		phoneHash, err := ComputeRosterBlindIndex(s.hmacKey, config.SchoolID, BlindIndexPhone, mainland)
		if err != nil {
			return preparedRosterRecord{}, err
		}
		phoneEnc, err := s.rosterCipher.Encrypt(mainland)
		if err != nil {
			return preparedRosterRecord{}, qualityFailure("identity_projection.integrity", "phone_encrypt_failed")
		}
		row.PhoneHash = &phoneHash
		row.PhoneEnc = phoneEnc
	}
	if err := s.verifyPreparedRosterProjection(config.SchoolID, studentID, name, row); err != nil {
		return preparedRosterRecord{}, err
	}
	row.RecordChecksum = rosterRecordChecksum(row)
	return row, nil
}

func (s *Service) verifyPreparedRosterProjection(
	schoolID int64,
	studentID string,
	name string,
	row preparedRosterRecord,
) error {
	studentPlain, err := s.rosterCipher.Decrypt(row.StudentIDEnc)
	if err != nil || studentPlain != studentID {
		return qualityFailure("identity_projection.integrity", "student_id_roundtrip_failed")
	}
	namePlain, err := s.rosterCipher.Decrypt(row.NameEnc)
	if err != nil || namePlain != name {
		return qualityFailure("identity_projection.integrity", "name_roundtrip_failed")
	}
	studentHash, err := ComputeRosterBlindIndex(s.hmacKey, schoolID, BlindIndexStudentID, studentPlain)
	if err != nil || studentHash != row.StudentIDHash {
		return qualityFailure("identity_projection.integrity", "student_id_hmac_failed")
	}
	nameHash, err := ComputeRosterBlindIndex(s.hmacKey, schoolID, BlindIndexName, namePlain)
	if err != nil || nameHash != row.NameHash {
		return qualityFailure("identity_projection.integrity", "name_hmac_failed")
	}
	if row.DocumentNumberHash != nil {
		documentPlain, decryptErr := s.rosterCipher.Decrypt(row.DocumentNumberEnc)
		if decryptErr != nil {
			return qualityFailure("identity_projection.integrity", "document_roundtrip_failed")
		}
		documentHash, hashErr := ComputeRosterBlindIndex(
			s.hmacKey, schoolID, BlindIndexDocumentNumber, documentPlain,
		)
		if hashErr != nil || !hmac.Equal([]byte(documentHash), []byte(*row.DocumentNumberHash)) {
			return qualityFailure("identity_projection.integrity", "document_hmac_failed")
		}
	}
	if row.PhoneHash != nil {
		phonePlain, decryptErr := s.rosterCipher.Decrypt(row.PhoneEnc)
		if decryptErr != nil {
			return qualityFailure("identity_projection.integrity", "phone_roundtrip_failed")
		}
		phoneHash, hashErr := ComputeRosterBlindIndex(s.hmacKey, schoolID, BlindIndexPhone, phonePlain)
		if hashErr != nil || !hmac.Equal([]byte(phoneHash), []byte(*row.PhoneHash)) {
			return qualityFailure("identity_projection.integrity", "phone_hmac_failed")
		}
	}
	return nil
}

func rosterRecordChecksum(row preparedRosterRecord) string {
	parts := []string{
		row.StudentIDHash, row.NameHash, pointerString(row.DocumentType),
		pointerString(row.DocumentNumberHash), pointerString(row.PhoneHash),
		pointerString(row.StudentStatus), pointerString(row.OnCampusStatus),
		pointerString(row.RegistrationStatus), pointerString(row.EducationLevel),
		pointerString(row.StudentCategory), pointerInt(row.EnrollmentYear),
		pointerDate(row.ValidFrom), pointerDate(row.ValidUntil), pointerBool(row.CurrentMarker),
		row.EligibilityStatus, row.EligibilityCode, pointerTime(row.SourceUpdatedAt),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func (s *Service) evaluateRosterAggregateChecks(
	ctx context.Context,
	config rosterImportConfig,
	policy rosterPolicy,
	sourceCutoff time.Time,
	prepared preparedRosterImport,
	now time.Time,
) ([]RosterQualityCheck, error) {
	checks := make([]RosterQualityCheck, 0, 3)
	rowStatus := "passed"
	rowDetail := ""
	if len(prepared.Rows) < policy.MinimumRows {
		rowStatus = "failed"
		rowDetail = "below_minimum_rows"
	}
	checks = append(checks, qualityCheck(
		"row_count.minimum", rowStatus,
		map[string]any{"rows": len(prepared.Rows)},
		map[string]any{"minimum": policy.MinimumRows}, rowDetail, now,
	))
	previous, err := s.repo.GetCurrentRosterAggregate(ctx, config.SchoolID)
	if errors.Is(err, pgx.ErrNoRows) {
		checks = append(checks,
			qualityCheck("source_time.monotonic", "passed", map[string]any{"hasPrevious": false}, nil, "", now),
			qualityCheck("row_count.delta", "passed", map[string]any{"hasPrevious": false}, map[string]any{"maximumRatio": policy.MaximumRowDeltaRatio}, "", now),
		)
		return checks, nil
	}
	if err != nil {
		return nil, err
	}
	sourceStatus := "passed"
	sourceDetail := ""
	if sourceCutoff.Before(previous.SourceCutoffAt) {
		sourceStatus = "failed"
		sourceDetail = "source_time_regressed"
	}
	checks = append(checks, qualityCheck(
		"source_time.monotonic", sourceStatus,
		map[string]any{"notBeforeCurrent": !sourceCutoff.Before(previous.SourceCutoffAt)}, nil,
		sourceDetail, now,
	))
	delta := rowDeltaRatio(int64(len(prepared.Rows)), previous.RowCount)
	deltaStatus := "passed"
	deltaDetail := ""
	if delta > policy.MaximumRowDeltaRatio {
		deltaStatus = "failed"
		deltaDetail = "row_count_delta_exceeded"
	}
	checks = append(checks, qualityCheck(
		"row_count.delta", deltaStatus,
		map[string]any{"ratio": delta},
		map[string]any{"maximumRatio": policy.MaximumRowDeltaRatio}, deltaDetail, now,
	))
	return checks, nil
}

func qualityFailure(checkKey, detailCode string) error {
	return &rosterQualityError{checkKey: checkKey, detailCode: detailCode}
}

func qualityCheck(
	key string,
	status string,
	measured map[string]any,
	threshold map[string]any,
	detail string,
	now time.Time,
) RosterQualityCheck {
	if measured == nil {
		measured = map[string]any{}
	}
	if threshold == nil {
		threshold = map[string]any{}
	}
	return RosterQualityCheck{
		CheckKey: key, Status: status, Measured: measured,
		Threshold: threshold, DetailCode: detail, CheckedAt: now,
	}
}

func validRosterSourceKind(value string) bool {
	return value == "campus_connector" || value == "isolated_oracle_worker" || value == "fixture"
}

func normalizedOptionalRosterValue(value string) (*string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 100 {
		return nil, false
	}
	return &value, true
}

func rosterImportFingerprint(key []byte, input FullRosterImportInput) (string, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, err = mac.Write([]byte("student-roster-import:v1\x00"))
	if err != nil {
		return "", err
	}
	if _, err := mac.Write(encoded); err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func normalizeDatePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	date := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &date
}

func rowDeltaRatio(current, previous int64) float64 {
	if previous <= 0 {
		if current == 0 {
			return 0
		}
		return 1
	}
	delta := current - previous
	if delta < 0 {
		delta = -delta
	}
	return float64(delta) / float64(previous)
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pointerInt(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func pointerDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func pointerTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func pointerBool(value *bool) string {
	if value == nil {
		return ""
	}
	return strconv.FormatBool(*value)
}

func hasFailedQualityCheck(checks []RosterQualityCheck) bool {
	return slices.ContainsFunc(checks, func(check RosterQualityCheck) bool { return check.Status == "failed" })
}
