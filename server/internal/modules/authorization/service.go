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
	grant, err := s.repo.GetGrant(ctx, input.GrantID)
	if err != nil {
		return MutationResult{}, err
	}
	if grant.Role == RoleSuperAdmin || grant.Source == GrantSourceCasdoorOrganizationAdmin {
		return MutationResult{}, ErrProviderManagedRole
	}
	result, err := s.repo.RevokeGrant(ctx, input)
	if err != nil {
		return MutationResult{}, fmt.Errorf("revoke authorization grant: %w", err)
	}
	return result, nil
}

func (s *Service) ReconcileGrant(ctx context.Context, input ReconcileGrantInput) (MutationResult, error) {
	input, err := normalizeReconcileGrantInput(input)
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
	result, err := s.repo.ReconcileGrant(ctx, input)
	if err != nil {
		return MutationResult{}, fmt.Errorf("reconcile authorization grant: %w", err)
	}
	return result, nil
}

func (s *Service) ReconcileAll(ctx context.Context, input ReconcileAllInput) (ReconcileAllResult, error) {
	input, err := normalizeReconcileAllInput(input)
	if err != nil {
		return ReconcileAllResult{}, err
	}
	exists, err := s.repo.UserExists(ctx, input.ActorUserID)
	if err != nil {
		return ReconcileAllResult{}, err
	}
	if !exists {
		return ReconcileAllResult{}, ErrActorUserNotFound
	}
	queued, err := s.repo.ReconcileAll(ctx, input)
	if err != nil {
		return ReconcileAllResult{}, fmt.Errorf("reconcile all authorization grants: %w", err)
	}
	return ReconcileAllResult{Queued: queued}, nil
}

// SyncCasdoorOrganizationAdmin projects Casdoor's authoritative organization
// administrator flag into the local grant ledger and OpenFGA. It is the only
// supported mutation path for super_admin.
func (s *Service) SyncCasdoorOrganizationAdmin(
	ctx context.Context,
	input CasdoorOrganizationAdminSyncInput,
) (MutationResult, error) {
	input, err := normalizeCasdoorOrganizationAdminSyncInput(input)
	if err != nil {
		return MutationResult{}, err
	}
	exists, err := s.repo.UserExists(ctx, input.SubjectUserID)
	if err != nil {
		return MutationResult{}, err
	}
	if !exists {
		return MutationResult{}, ErrTargetUserNotFound
	}
	result, err := s.repo.SyncCasdoorOrganizationAdmin(ctx, input)
	if err != nil {
		return MutationResult{}, fmt.Errorf("sync Casdoor organization administrator: %w", err)
	}
	return result, nil
}

func (s *Service) ListGrants(ctx context.Context, filter ListGrantsFilter) (GrantList, error) {
	var err error
	filter, err = normalizeListGrantsFilter(filter)
	if err != nil {
		return GrantList{}, err
	}
	return s.repo.ListGrants(ctx, filter)
}

func (s *Service) GetGrant(ctx context.Context, grantID int64) (Grant, error) {
	if grantID <= 0 {
		return Grant{}, ErrInvalidGrant
	}
	return s.repo.GetGrant(ctx, grantID)
}

func (s *Service) ResolveAccessSnapshotByUserID(
	ctx context.Context,
	userID int64,
) (AccessSnapshot, error) {
	return s.repo.ResolveAccessSnapshotByUserID(ctx, userID)
}
