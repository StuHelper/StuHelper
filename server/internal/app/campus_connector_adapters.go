package app

import (
	"context"
	"errors"

	"github.com/StuHelper/StuHelper/server/internal/modules/campusconnector"
	"github.com/StuHelper/StuHelper/server/internal/modules/studentverification"
)

type campusConnectorSchoolAuthenticator struct {
	service *campusconnector.Service
}

func (a campusConnectorSchoolAuthenticator) Authenticate(
	ctx context.Context,
	request studentverification.SchoolAccountAuthenticationRequest,
) (*studentverification.SchoolAccountAuthenticationResult, error) {
	if a.service == nil {
		return nil, studentverification.ErrDependencyUnavailable
	}
	var applicationID *string
	if request.ApplicationID != "" {
		applicationID = &request.ApplicationID
	}
	result, err := a.service.AuthenticateSchoolAccount(ctx, campusconnector.SchoolAccountInput{
		SchoolID: request.SchoolID, OperationKey: request.ConnectorOperation,
		AdapterID: request.AdapterID, AdapterVersion: request.AdapterVersion,
		StudentID: request.StudentID, Password: request.Password,
		ApplicationID: applicationID,
	})
	if err != nil {
		switch {
		case errors.Is(err, campusconnector.ErrRejected):
			return nil, studentverification.ErrSchoolAccountRejected
		case errors.Is(err, campusconnector.ErrAccountLocked):
			return nil, studentverification.ErrSchoolAccountLocked
		case errors.Is(err, campusconnector.ErrNotStudent):
			return nil, studentverification.ErrSchoolAccountNotStudent
		default:
			return nil, studentverification.ErrDependencyUnavailable
		}
	}
	return &studentverification.SchoolAccountAuthenticationResult{
		AccountSubject: result.AccountSubject,
		StudentID:      result.StudentID,
	}, nil
}

type campusConnectorSnapshotImporter struct {
	service *studentverification.Service
}

type campusConnectorRosterSyncCoordinator struct {
	service *campusconnector.Service
}

func (a campusConnectorRosterSyncCoordinator) RequestRosterSync(
	ctx context.Context,
	input studentverification.AdminRosterSyncInput,
) (*studentverification.AdminRosterSyncRequest, error) {
	if a.service == nil {
		return nil, studentverification.ErrDependencyUnavailable
	}
	request, err := a.service.RequestManualRosterSync(ctx, campusconnector.ManualRosterSyncInput{
		SchoolCode: input.SchoolCode, ActorUserID: input.ActorUserID, Reason: input.Reason,
	})
	if err != nil {
		return nil, mapCampusConnectorRosterSyncError(err)
	}
	return adminRosterSyncProjection(*request), nil
}

func (a campusConnectorRosterSyncCoordinator) ListRosterSyncRequests(
	ctx context.Context,
	schoolCode string,
	limit int,
) ([]studentverification.AdminRosterSyncRequest, error) {
	if a.service == nil {
		return nil, studentverification.ErrDependencyUnavailable
	}
	requests, err := a.service.ListManualRosterSyncRequests(ctx, schoolCode, limit)
	if err != nil {
		return nil, mapCampusConnectorRosterSyncError(err)
	}
	result := make([]studentverification.AdminRosterSyncRequest, len(requests))
	for index := range requests {
		result[index] = *adminRosterSyncProjection(requests[index])
	}
	return result, nil
}

func adminRosterSyncProjection(
	request campusconnector.ManualRosterSyncRequest,
) *studentverification.AdminRosterSyncRequest {
	return &studentverification.AdminRosterSyncRequest{
		ID: request.ID, SchoolCode: request.SchoolCode,
		OperationKey: request.OperationKey, AdapterID: request.AdapterID,
		AdapterVersion: request.AdapterVersion, Status: request.Status,
		ResultCode: request.ResultCode, RequestedByUserID: request.ActorUserID,
		Reason: request.Reason, DeadlineAt: request.DeadlineAt,
		ClaimedAt: request.ClaimedAt, ClaimAttempts: request.ClaimAttempts,
		CompletedAt: request.CompletedAt, LatencyMilliseconds: request.LatencyMilliseconds,
		CreatedAt: request.CreatedAt, UpdatedAt: request.UpdatedAt,
	}
}

func mapCampusConnectorRosterSyncError(err error) error {
	switch {
	case errors.Is(err, campusconnector.ErrRequestInFlight):
		return studentverification.ErrAdminRosterSyncConflict
	case errors.Is(err, campusconnector.ErrRejected):
		return studentverification.ErrAdminConfigInvalid
	case errors.Is(err, campusconnector.ErrUnavailable):
		return studentverification.ErrDependencyUnavailable
	default:
		return err
	}
}

func (a campusConnectorSnapshotImporter) ImportCampusConnectorSnapshot(
	ctx context.Context,
	request campusconnector.SnapshotImportRequest,
) (string, error) {
	if a.service == nil {
		return "", campusconnector.ErrSnapshotRejected
	}
	records := make([]studentverification.RosterImportRecord, len(request.Payload.Records))
	for index := range request.Payload.Records {
		record := request.Payload.Records[index]
		records[index] = studentverification.RosterImportRecord{
			StudentID: record.StudentID, Name: record.Name,
			DocumentType: record.DocumentType, DocumentNumber: record.DocumentNumber,
			Phone: record.Phone, StudentStatus: record.StudentStatus,
			OnCampusStatus: record.OnCampusStatus, RegistrationStatus: record.RegistrationStatus,
			EducationLevel: record.EducationLevel, StudentCategory: record.StudentCategory,
			EnrollmentYear: record.EnrollmentYear, ValidFrom: record.ValidFrom,
			ValidUntil: record.ValidUntil, CurrentMarker: record.CurrentMarker,
			EligibilityCode: record.EligibilityCode, SourceUpdatedAt: record.SourceUpdatedAt,
		}
	}
	defer clearRosterImportRecords(records)
	nodeID := request.NodeID
	var sourceQuality *studentverification.RosterSourceQualitySummary
	if request.Manifest.QualitySummary != nil {
		quality := request.Manifest.QualitySummary
		sourceQuality = &studentverification.RosterSourceQualitySummary{
			RowsRead: quality.RowsRead, RecordsEmitted: quality.RecordsEmitted,
			MissingDocumentNumber: quality.MissingDocumentNumber,
			InvalidDocumentNumber: quality.InvalidDocumentNumber,
			MissingPhone:          quality.MissingPhone, InvalidPhone: quality.InvalidPhone,
			MissingEnrollmentYear: quality.MissingEnrollmentYear,
			InvalidEnrollmentYear: quality.InvalidEnrollmentYear,
		}
	}
	snapshot, err := a.service.ImportFullRoster(ctx, studentverification.FullRosterImportInput{
		SchoolCode: request.Manifest.SchoolCode, SourceKind: "campus_connector",
		SourceVersion:   request.Manifest.SourceVersion,
		MappingVersion:  request.Manifest.MappingVersion,
		SourceStartedAt: &request.Manifest.SourceStartedAt,
		SourceCutoffAt:  request.Manifest.SourceCutoffAt,
		ConnectorNodeID: &nodeID, SignatureAlgorithm: "Ed25519",
		SignatureKeyID:    request.Manifest.SigningKeyID,
		SnapshotSignature: request.Signature, SourceQuality: sourceQuality, Records: records,
	})
	if err != nil {
		return "", err
	}
	snapshot, err = a.service.AutoActivateImportedRosterSnapshot(
		ctx,
		request.Manifest.SchoolCode,
		snapshot.ID,
	)
	if err != nil {
		return "", err
	}
	return snapshot.ID, nil
}

func clearRosterImportRecords(records []studentverification.RosterImportRecord) {
	for index := range records {
		records[index].StudentID = ""
		records[index].Name = ""
		records[index].DocumentType = ""
		records[index].DocumentNumber = ""
		records[index].Phone = ""
		records[index].StudentStatus = ""
		records[index].OnCampusStatus = ""
		records[index].RegistrationStatus = ""
		records[index].EducationLevel = ""
		records[index].StudentCategory = ""
		records[index].EligibilityCode = ""
	}
}
