package openplatform

import (
	"context"
	"fmt"

	platformcasdoor "git.stuhelper.com/StuHelper/StuHelper/internal/platform/casdoor"
)

type ApproveScopeInput struct {
	AppID          int64
	Scope          string
	ReviewerUserID int64
	DecisionNote   string
}

type ApprovedApp struct {
	App          *App
	ClientSecret string
}

func (s *Service) ApproveScope(ctx context.Context, input ApproveScopeInput) error {
	scope, err := normalizeSingleScope(input.Scope)
	if err != nil {
		return err
	}
	if input.ReviewerUserID <= 0 {
		return fmt.Errorf("open platform reviewer user id is required")
	}
	if err := s.repo.ApproveScope(ctx, input.AppID, scope, input.ReviewerUserID, input.DecisionNote); err != nil {
		return err
	}
	return nil
}

func (s *Service) ApproveApp(ctx context.Context, appID int64) (*ApprovedApp, error) {
	app, err := s.repo.GetAppByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	scopes, err := s.repo.ListApprovedScopes(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		return nil, ErrScopeNotApproved
	}
	if s.provisioner == nil {
		return nil, fmt.Errorf("open platform app provisioner is not configured")
	}
	secret, err := randomHex("ops_", clientSecretBytes)
	if err != nil {
		return nil, err
	}
	spec := casdoorApplicationSpec(app)
	spec.ClientSecret = secret
	if err := s.provisioner.CreateApplication(ctx, spec); err != nil {
		return nil, fmt.Errorf("ApproveApp provision Casdoor application: %w", err)
	}
	if err := s.repo.MarkAppApproved(ctx, appID, hashClientSecret(secret)); err != nil {
		return nil, err
	}
	app.Status = AppStatusApproved
	return &ApprovedApp{App: app, ClientSecret: secret}, nil
}

func normalizeSingleScope(raw string) (string, error) {
	scopes, err := NormalizeScopes([]string{raw})
	if err != nil {
		return "", err
	}
	return scopes[0], nil
}

func casdoorApplicationSpec(app *App) platformcasdoor.ApplicationSpec {
	return platformcasdoor.ApplicationSpec{
		Name:                 app.CasdoorApplicationName,
		DisplayName:          app.DisplayName,
		Logo:                 "",
		HomepageURL:          app.HomepageURL,
		Description:          app.Description,
		ClientID:             app.ClientID,
		ClientSecret:         app.ClientSecretHash,
		RedirectURIs:         app.RedirectURIs,
		GrantTypes:           []string{"authorization_code", "refresh_token"},
		TokenFormat:          "JWT",
		TokenFields:          []string{},
		ExpireInHours:        1,
		RefreshExpireInHours: 24 * 30,
	}
}
