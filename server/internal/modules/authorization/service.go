package authorization

import (
	"context"
	"fmt"
)

type Service struct {
	repo       *Repository
	projection ProjectionClient
}

type ServiceOption func(*Service)

func WithProjectionClient(client ProjectionClient) ServiceOption {
	return func(service *Service) {
		service.projection = client
	}
}

func NewService(repo *Repository, opts ...ServiceOption) *Service {
	if repo == nil {
		panic("authorization.NewService: repository is required")
	}
	service := &Service{repo: repo}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

func (s *Service) CreateGrant(ctx context.Context, input CreateGrantInput) (MutationResult, error) {
	input, err := normalizeCreateGrantInput(input)
	if err != nil {
		return MutationResult{}, err
	}
	exists, err := s.repo.UserExists(ctx, input.ActorUserID)
	if err != nil {
		return MutationResult{}, err
	}
	if !exists {
		return MutationResult{}, ErrActorUserNotFound
	}
	exists, err = s.repo.UserExists(ctx, input.SubjectUserID)
	if err != nil {
		return MutationResult{}, err
	}
	if !exists {
		return MutationResult{}, ErrTargetUserNotFound
	}
	if input.SchoolID != nil {
		exists, err = s.repo.SchoolExists(ctx, *input.SchoolID)
		if err != nil {
			return MutationResult{}, err
		}
		if !exists {
			return MutationResult{}, ErrSchoolNotFound
		}
	}
	result, err := s.repo.CreateOrRestoreGrant(ctx, input)
	if err != nil {
		return MutationResult{}, fmt.Errorf("create authorization grant: %w", err)
	}
	return result, nil
}

func (s *Service) RevokeGrant(ctx context.Context, input RevokeGrantInput) (MutationResult, error) {
	input, err := normalizeRevokeGrantInput(input)
	if err != nil {
		return MutationResult{}, err
	}
	exists, err := s.repo.UserExists(ctx, input.ActorUserID)
	if err != nil {
		return MutationResult{}, err
	}
	if !exists {
		return MutationResult{}, ErrActorUserNotFound
	}
	result, err := s.repo.RevokeGrant(ctx, input)
	if err != nil {
		return MutationResult{}, fmt.Errorf("revoke authorization grant: %w", err)
	}
	return result, nil
}

func (s *Service) ListGrants(ctx context.Context, filter ListGrantsFilter) (GrantList, error) {
	return s.repo.ListGrants(ctx, filter)
}
