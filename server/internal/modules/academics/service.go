package academics

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidOfferingID = errors.New("academic offering id is invalid")
var ErrAcademicUserRequired = errors.New("academic user id is required")

type Service struct {
	repo     *Repository
	registry *Registry
}

func NewService(repo *Repository, registry *Registry) *Service {
	if repo == nil {
		panic("academics.NewService: repo must not be nil")
	}
	if registry == nil {
		registry = NewRegistry()
	}
	return &Service{repo: repo, registry: registry}
}

func (s *Service) ListSources(ctx context.Context) ([]Source, error) {
	return s.repo.ListSources(ctx)
}

func (s *Service) ListImportJobs(ctx context.Context, page, pageSize int) ([]ImportJob, int, error) {
	return s.repo.ListImportJobs(ctx, page, pageSize)
}

func (s *Service) TriggerImport(ctx context.Context, sourceKey, requestedBy string) (*ImportJob, error) {
	source, err := s.repo.GetSourceByKey(ctx, sourceKey)
	if err != nil {
		return nil, err
	}
	if !source.Enabled {
		return nil, ErrSourceDisabled
	}
	connector, err := s.registry.Build(*source)
	if err != nil {
		return nil, fmt.Errorf("build academic connector: %w", err)
	}
	if err := connector.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("academic connector health check failed: %w", err)
	}
	jobID, err := s.repo.CreateImportJob(ctx, *source, requestedBy)
	if err != nil {
		return nil, err
	}
	if err := s.repo.MarkImportJobRunning(ctx, jobID); err != nil {
		return nil, err
	}
	snapshot, err := loadSnapshot(ctx, connector)
	if err != nil {
		return nil, s.failImportJob(ctx, jobID, err)
	}
	if err := s.repo.ReplaceSnapshot(ctx, jobID, source.ID, snapshot); err != nil {
		return nil, s.failImportJob(ctx, jobID, err)
	}
	if err := s.repo.MarkImportJobSucceeded(ctx, jobID, snapshot.Stats()); err != nil {
		return nil, err
	}
	return s.repo.GetImportJobByID(ctx, jobID)
}

func (s *Service) failImportJob(ctx context.Context, jobID int64, cause error) error {
	if err := s.repo.MarkImportJobFailed(ctx, jobID, cause.Error()); err != nil {
		return errors.Join(cause, fmt.Errorf("mark import job failed: %w", err))
	}
	return cause
}

func loadSnapshot(ctx context.Context, connector Connector) (Snapshot, error) {
	terms, err := connector.FetchTerms(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch academic terms: %w", err)
	}
	courses, err := connector.FetchCourses(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch academic courses: %w", err)
	}
	teachers, err := connector.FetchTeachers(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch academic teachers: %w", err)
	}
	offerings, err := connector.FetchOfferings(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch academic offerings: %w", err)
	}
	memberships, err := connector.FetchMemberships(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("fetch academic memberships: %w", err)
	}
	return Snapshot{
		Terms:       terms,
		Courses:     courses,
		Teachers:    teachers,
		Offerings:   offerings,
		Memberships: memberships,
	}, nil
}

func (s *Service) ListTerms(ctx context.Context) ([]Term, error) {
	return s.repo.ListTerms(ctx)
}

func (s *Service) ListOfferings(ctx context.Context, filters OfferingFilters) ([]Offering, int, error) {
	return s.repo.ListOfferings(ctx, filters)
}

func (s *Service) GetOffering(ctx context.Context, offeringID int64) (*Offering, error) {
	if err := validateOfferingID(offeringID); err != nil {
		return nil, err
	}
	return s.repo.GetOfferingByID(ctx, offeringID)
}

func (s *Service) ListMyCourses(ctx context.Context, externalUserID, termCode string) ([]Offering, error) {
	externalUserID, err := normalizeRequiredAcademicUserID(externalUserID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMyCourses(ctx, externalUserID, termCode)
}

func (s *Service) ListMySchedule(ctx context.Context, externalUserID, termCode string) ([]Offering, error) {
	externalUserID, err := normalizeRequiredAcademicUserID(externalUserID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMySchedule(ctx, externalUserID, termCode)
}

func validateOfferingID(offeringID int64) error {
	if offeringID <= 0 {
		return ErrInvalidOfferingID
	}
	return nil
}

func normalizeRequiredAcademicUserID(externalUserID string) (string, error) {
	externalUserID = strings.TrimSpace(externalUserID)
	if externalUserID == "" {
		return "", ErrAcademicUserRequired
	}
	return externalUserID, nil
}
