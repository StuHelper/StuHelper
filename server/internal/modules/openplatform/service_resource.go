package openplatform

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/pkg/ctxutil"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

const resourceAccessRollbackTimeout = 5 * time.Second

func (s *Service) GrantResourceAccess(ctx context.Context, input ResourceGrantInput) (*ResourceGrantResult, error) {
	if err := s.ensureResourceFGAConfigured(); err != nil {
		return nil, err
	}
	if input.AppID <= 0 || input.ReviewerUserID <= 0 {
		return nil, ErrInvalidResourceAccess
	}
	resourceType, resourceID, err := normalizeResourceTarget(input.ResourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}
	actions, err := normalizeResourceActionsForType(resourceType, input.Actions)
	if err != nil {
		return nil, err
	}
	reason, err := normalizeResourceAccessReason(input.Reason)
	if err != nil {
		return nil, err
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}
	if app.Status != AppStatusApproved {
		return nil, ErrAppNotActive
	}
	if err := s.ensureResourceScopesApproved(ctx, app.ID, actions); err != nil {
		return nil, err
	}

	grants := resourceGrantsForActions(app.ID, resourceType, resourceID, actions)
	tuples := make([]fga.Tuple, 0, len(grants))
	for _, grant := range grants {
		tuples = append(tuples, resourceGrantTuple(grant))
	}
	if err := s.resourceFGA.WriteMissingTuples(ctx, tuples); err != nil {
		return nil, fmt.Errorf("%w: write resource access tuples: %v", ErrResourceAccessUnavailable, err)
	}
	if err := s.recordResourceAccessAdminAudit(ctx, app.ID, input.ReviewerUserID, input.RequestID,
		"open_platform.resource_access.granted", resourceType, resourceID, actions, reason); err != nil {
		if rollbackErr := s.deleteResourceTuplesForRollback(ctx, tuples); rollbackErr != nil {
			return nil, errors.Join(err, fmt.Errorf("%w: rollback granted resource access tuples: %v", ErrResourceAccessUnavailable, rollbackErr))
		}
		return nil, err
	}
	return &ResourceGrantResult{App: app, Grants: grants}, nil
}

func (s *Service) RevokeResourceAccess(ctx context.Context, input ResourceGrantRevokeInput) (*ResourceGrantResult, error) {
	if err := s.ensureResourceFGAConfigured(); err != nil {
		return nil, err
	}
	if input.AppID <= 0 || input.ReviewerUserID <= 0 {
		return nil, ErrInvalidResourceAccess
	}
	resourceType, resourceID, err := normalizeResourceTarget(input.ResourceType, input.ResourceID)
	if err != nil {
		return nil, err
	}
	actions, err := normalizeResourceActionsForType(resourceType, input.Actions)
	if err != nil {
		return nil, err
	}
	reason, err := normalizeResourceAccessReason(input.Reason)
	if err != nil {
		return nil, err
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	grants := resourceGrantsForActions(app.ID, resourceType, resourceID, actions)
	tuples := make([]fga.Tuple, 0, len(grants))
	for _, grant := range grants {
		tuples = append(tuples, resourceGrantTuple(grant))
	}
	existingTuples, err := s.existingResourceTuples(ctx, tuples)
	if err != nil {
		return nil, err
	}
	if err := s.resourceFGA.DeleteTuples(ctx, tuples); err != nil {
		return nil, fmt.Errorf("%w: delete resource access tuples: %v", ErrResourceAccessUnavailable, err)
	}
	if err := s.recordResourceAccessAdminAudit(ctx, app.ID, input.ReviewerUserID, input.RequestID,
		"open_platform.resource_access.revoked", resourceType, resourceID, actions, reason); err != nil {
		if len(existingTuples) > 0 {
			if rollbackErr := s.writeResourceTuplesForRollback(ctx, existingTuples); rollbackErr != nil {
				return nil, errors.Join(err, fmt.Errorf("%w: restore revoked resource access tuples: %v", ErrResourceAccessUnavailable, rollbackErr))
			}
		}
		return nil, err
	}
	return &ResourceGrantResult{App: app, Grants: grants}, nil
}

func (s *Service) ListResourceGrants(ctx context.Context, input ResourceGrantListInput) (*ResourceGrantResult, error) {
	if err := s.ensureResourceFGAConfigured(); err != nil {
		return nil, err
	}
	if input.AppID <= 0 {
		return nil, ErrInvalidResourceAccess
	}
	resourceType, err := normalizeResourceType(input.ResourceType)
	if err != nil {
		return nil, err
	}
	app, err := s.repo.GetAppByID(ctx, input.AppID)
	if err != nil {
		return nil, err
	}

	appUser := openPlatformAppFGAUser(app.ID)
	grants, err := s.listResourceGrantsForApp(ctx, app.ID, appUser, resourceType)
	if err != nil {
		return nil, err
	}
	return &ResourceGrantResult{App: app, Grants: grants}, nil
}

func (s *Service) CheckResourceAccess(ctx context.Context, input ResourceAccessCheckInput) (ResourceAccessDecision, error) {
	if err := s.ensureResourceFGAConfigured(); err != nil {
		return ResourceAccessDecision{}, err
	}
	resourceType, resourceID, err := normalizeResourceTarget(input.ResourceType, input.ResourceID)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	action, err := normalizeSingleResourceAction(input.Action)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	if !resourceActionSupported(resourceType, action) {
		return ResourceAccessDecision{}, ErrInvalidResourceAccess
	}
	relation, requiredScope, err := resourceActionPolicy(action)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	app, tokenScopes, tokenAuthenticated, err := s.resourceAccessApp(ctx, input)
	if err != nil {
		return ResourceAccessDecision{}, err
	}

	decision := ResourceAccessDecision{
		AppID:        app.ID,
		ClientID:     app.ClientID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Relation:     relation,
	}
	if tokenAuthenticated && !resourceAccessTokenHasScope(tokenScopes, requiredScope) {
		decision.Allowed = false
		decision.Reason = "token_scope_missing"
		return decision, s.recordResourceAccessCheckAudit(ctx, app.ID, input.RequestID, decision)
	}
	if err := s.ensureScopesApproved(ctx, app.ID, []string{requiredScope}); err != nil {
		decision.Allowed = false
		decision.Reason = "scope_not_approved"
		return decision, s.recordResourceAccessCheckAudit(ctx, app.ID, input.RequestID, decision)
	}

	allowed, err := s.resourceFGA.Check(
		ctx,
		openPlatformAppFGAUser(app.ID),
		relation,
		resourceFGAObject(resourceType, resourceID),
	)
	if err != nil {
		return ResourceAccessDecision{}, fmt.Errorf("%w: check resource access tuple: %v", ErrResourceAccessUnavailable, err)
	}
	decision.Allowed = allowed
	if allowed {
		decision.Reason = "allowed"
	} else {
		decision.Reason = "fga_denied"
	}
	return decision, s.recordResourceAccessCheckAudit(ctx, app.ID, input.RequestID, decision)
}

func (s *Service) resourceAccessApp(ctx context.Context, input ResourceAccessCheckInput) (*App, []string, bool, error) {
	tokenClientID := strings.TrimSpace(input.AccessTokenClientID)
	if tokenClientID != "" {
		if strings.TrimSpace(input.ClientID) != "" || strings.TrimSpace(input.ClientSecret) != "" {
			return nil, nil, false, ErrInvalidResourceAccess
		}
		scopes, err := NormalizeGrantedOAuthScopes(input.AccessTokenScopes)
		if err != nil {
			return nil, nil, false, ErrInvalidResourceAccess
		}
		app, err := s.repo.GetAppByClientID(ctx, tokenClientID)
		if err != nil {
			return nil, nil, false, err
		}
		if app.Status != AppStatusApproved {
			return nil, nil, false, ErrAppNotActive
		}
		return app, scopes, true, nil
	}
	if strings.TrimSpace(input.ClientID) == "" || strings.TrimSpace(input.ClientSecret) == "" {
		return nil, nil, false, ErrInvalidResourceAccess
	}
	app, err := s.VerifyClientSecret(ctx, input.ClientID, input.ClientSecret)
	if err != nil {
		return nil, nil, false, err
	}
	return app, nil, false, nil
}

func resourceAccessTokenHasScope(scopes []string, requiredScope string) bool {
	for _, scope := range scopes {
		if scope == requiredScope {
			return true
		}
	}
	return false
}

func (s *Service) ensureResourceScopesApproved(ctx context.Context, appID int64, actions []string) error {
	scopes := make([]string, 0, len(actions))
	for _, action := range actions {
		_, scope, err := resourceActionPolicy(action)
		if err != nil {
			return err
		}
		scopes = append(scopes, scope)
	}
	return s.ensureScopesApproved(ctx, appID, scopes)
}

func (s *Service) ensureResourceFGAConfigured() error {
	if s.resourceFGA == nil {
		return ErrResourceAccessUnavailable
	}
	return nil
}

func (s *Service) existingResourceTuples(ctx context.Context, tuples []fga.Tuple) ([]fga.Tuple, error) {
	existing := make([]fga.Tuple, 0, len(tuples))
	for _, tuple := range tuples {
		ok, err := s.resourceFGA.Check(ctx, tuple.User, tuple.Relation, tuple.Object)
		if err != nil {
			return nil, fmt.Errorf("%w: check existing resource access tuple: %v", ErrResourceAccessUnavailable, err)
		}
		if ok {
			existing = append(existing, tuple)
		}
	}
	return existing, nil
}

func (s *Service) revokeAllResourceAccessForRevokedApp(ctx context.Context, appID int64, actorUserID int64, requestID string, reason string) (int, error) {
	if s.resourceFGA == nil {
		return 0, nil
	}
	appUser := openPlatformAppFGAUser(appID)
	allGrants := make([]ResourceGrant, 0)
	for _, resourceType := range []string{ResourceTypeResourceItem, ResourceTypeUserProfile} {
		grants, err := s.listResourceGrantsForApp(ctx, appID, appUser, resourceType)
		if err != nil {
			return 0, err
		}
		allGrants = append(allGrants, grants...)
	}
	if len(allGrants) == 0 {
		return 0, nil
	}

	tuples := make([]fga.Tuple, 0, len(allGrants))
	for _, grant := range allGrants {
		tuples = append(tuples, resourceGrantTuple(grant))
	}
	if err := s.resourceFGA.DeleteTuples(ctx, tuples); err != nil {
		return 0, fmt.Errorf("%w: delete resource access tuples for revoked app: %v", ErrResourceAccessUnavailable, err)
	}

	for _, group := range groupResourceGrantsByTarget(allGrants) {
		if err := s.recordResourceAccessAdminAuditWithSource(ctx, appID, actorUserID, requestID,
			"open_platform.resource_access.revoked", group.resourceType, group.resourceID,
			group.actions, reason, "app_lifecycle"); err != nil {
			if rollbackErr := s.writeResourceTuplesForRollback(ctx, tuples); rollbackErr != nil {
				return 0, errors.Join(err, fmt.Errorf("%w: restore revoked app resource access tuples: %v", ErrResourceAccessUnavailable, rollbackErr))
			}
			return 0, err
		}
	}
	return len(allGrants), nil
}

func (s *Service) deleteResourceTuplesForRollback(ctx context.Context, tuples []fga.Tuple) error {
	rollbackCtx, cancel := resourceAccessRollbackContext(ctx)
	defer cancel()
	return s.resourceFGA.DeleteTuples(rollbackCtx, tuples)
}

func (s *Service) writeResourceTuplesForRollback(ctx context.Context, tuples []fga.Tuple) error {
	rollbackCtx, cancel := resourceAccessRollbackContext(ctx)
	defer cancel()
	return s.resourceFGA.WriteMissingTuples(rollbackCtx, tuples)
}

func resourceAccessRollbackContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctxutil.DetachedTimeout(ctx, resourceAccessRollbackTimeout)
}

func (s *Service) listResourceGrantsForApp(ctx context.Context, appID int64, appUser string, resourceType string) ([]ResourceGrant, error) {
	grants := make([]ResourceGrant, 0)
	for _, action := range supportedResourceActions(resourceType) {
		relation, _, err := resourceActionPolicy(action)
		if err != nil {
			return nil, err
		}
		objects, err := s.resourceFGA.ListObjects(ctx, appUser, relation, resourceType)
		if err != nil {
			return nil, fmt.Errorf("%w: list resource access tuples: %v", ErrResourceAccessUnavailable, err)
		}
		for _, object := range objects {
			objectType, objectID, err := splitFGAObject(object)
			if err != nil || objectType != resourceType {
				continue
			}
			grants = append(grants, ResourceGrant{
				AppID:        appID,
				ResourceType: objectType,
				ResourceID:   objectID,
				Action:       action,
				Relation:     relation,
			})
		}
	}
	sortResourceGrants(grants)
	return grants, nil
}

func (s *Service) recordResourceAccessAdminAudit(
	ctx context.Context,
	appID int64,
	userID int64,
	requestID string,
	eventType string,
	resourceType string,
	resourceID string,
	actions []string,
	reason string,
) error {
	return s.recordResourceAccessAdminAuditWithSource(ctx, appID, userID, requestID, eventType, resourceType, resourceID, actions, reason, "openfga")
}

func (s *Service) recordResourceAccessAdminAuditWithSource(
	ctx context.Context,
	appID int64,
	userID int64,
	requestID string,
	eventType string,
	resourceType string,
	resourceID string,
	actions []string,
	reason string,
	source string,
) error {
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		UserID:    userID,
		EventType: eventType,
		RequestID: requestID,
		Metadata: map[string]any{
			"resourceType": resourceType,
			"resourceID":   resourceID,
			"actions":      actions,
			"reason":       strings.TrimSpace(reason),
			"source":       source,
		},
	}); err != nil {
		return fmt.Errorf("%w: resource access audit unavailable", ErrResourceAccessUnavailable)
	}
	return nil
}

func (s *Service) recordResourceAccessCheckAudit(
	ctx context.Context,
	appID int64,
	requestID string,
	decision ResourceAccessDecision,
) error {
	if err := s.repo.RecordAuditEvent(ctx, auditEvent{
		AppID:     appID,
		EventType: "open_platform.resource_access.checked",
		RequestID: requestID,
		Metadata: map[string]any{
			"resourceType": decision.ResourceType,
			"resourceID":   decision.ResourceID,
			"action":       decision.Action,
			"relation":     decision.Relation,
			"allowed":      decision.Allowed,
			"reason":       decision.Reason,
			"source":       "openfga",
		},
	}); err != nil {
		return fmt.Errorf("%w: resource access audit unavailable", ErrResourceAccessUnavailable)
	}
	return nil
}

func resourceGrantsForActions(appID int64, resourceType string, resourceID string, actions []string) []ResourceGrant {
	grants := make([]ResourceGrant, 0, len(actions))
	for _, action := range actions {
		relation, _, err := resourceActionPolicy(action)
		if err != nil {
			continue
		}
		grants = append(grants, ResourceGrant{
			AppID:        appID,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			Action:       action,
			Relation:     relation,
		})
	}
	return grants
}

type groupedResourceGrant struct {
	resourceType string
	resourceID   string
	actions      []string
}

func groupResourceGrantsByTarget(grants []ResourceGrant) []groupedResourceGrant {
	byTarget := make(map[string]*groupedResourceGrant, len(grants))
	order := make([]string, 0, len(grants))
	for _, grant := range grants {
		key := grant.ResourceType + "\x00" + grant.ResourceID
		group, ok := byTarget[key]
		if !ok {
			group = &groupedResourceGrant{
				resourceType: grant.ResourceType,
				resourceID:   grant.ResourceID,
				actions:      []string{},
			}
			byTarget[key] = group
			order = append(order, key)
		}
		group.actions = append(group.actions, grant.Action)
	}
	sort.Strings(order)
	groups := make([]groupedResourceGrant, 0, len(order))
	for _, key := range order {
		group := byTarget[key]
		sort.Strings(group.actions)
		groups = append(groups, *group)
	}
	return groups
}

func sortResourceGrants(grants []ResourceGrant) {
	sort.Slice(grants, func(i, j int) bool {
		if grants[i].ResourceType != grants[j].ResourceType {
			return grants[i].ResourceType < grants[j].ResourceType
		}
		if grants[i].ResourceID != grants[j].ResourceID {
			return grants[i].ResourceID < grants[j].ResourceID
		}
		return grants[i].Action < grants[j].Action
	})
}

func resourceGrantTuple(grant ResourceGrant) fga.Tuple {
	return fga.Tuple{
		User:     openPlatformAppFGAUser(grant.AppID),
		Relation: grant.Relation,
		Object:   resourceFGAObject(grant.ResourceType, grant.ResourceID),
	}
}

func normalizeResourceTarget(rawType string, rawID string) (string, string, error) {
	resourceType, err := normalizeResourceType(rawType)
	if err != nil {
		return "", "", err
	}
	resourceID := strings.TrimSpace(rawID)
	if !resourceIDPattern.MatchString(resourceID) {
		return "", "", ErrInvalidResourceAccess
	}
	return resourceType, resourceID, nil
}

func normalizeResourceType(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case ResourceTypeUserProfile:
		return ResourceTypeUserProfile, nil
	case ResourceTypeResourceItem:
		return ResourceTypeResourceItem, nil
	default:
		return "", ErrInvalidResourceAccess
	}
}

func normalizeResourceActions(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	actions := make([]string, 0, len(raw))
	for _, value := range raw {
		action, err := normalizeSingleResourceAction(value)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[action]; ok {
			continue
		}
		seen[action] = struct{}{}
		actions = append(actions, action)
	}
	if len(actions) == 0 {
		return nil, ErrInvalidResourceAccess
	}
	return actions, nil
}

func normalizeResourceActionsForType(resourceType string, raw []string) ([]string, error) {
	actions, err := normalizeResourceActions(raw)
	if err != nil {
		return nil, err
	}
	for _, action := range actions {
		if !resourceActionSupported(resourceType, action) {
			return nil, ErrInvalidResourceAccess
		}
	}
	return actions, nil
}

func normalizeResourceAccessReason(reason string) (string, error) {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return "", ErrResourceAccessReasonRequired
	}
	return trimmed, nil
}

func normalizeSingleResourceAction(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case ResourceAccessActionRead:
		return ResourceAccessActionRead, nil
	case ResourceAccessActionWrite:
		return ResourceAccessActionWrite, nil
	default:
		return "", ErrInvalidResourceAccess
	}
}

func supportedResourceActions(resourceType string) []string {
	switch resourceType {
	case ResourceTypeUserProfile:
		return []string{ResourceAccessActionRead}
	case ResourceTypeResourceItem:
		return []string{ResourceAccessActionRead, ResourceAccessActionWrite}
	default:
		return nil
	}
}

func resourceActionSupported(resourceType string, action string) bool {
	for _, supported := range supportedResourceActions(resourceType) {
		if action == supported {
			return true
		}
	}
	return false
}

func resourceActionPolicy(action string) (string, string, error) {
	switch action {
	case ResourceAccessActionRead:
		return ResourceRelationReadByApp, ScopeResourceRead, nil
	case ResourceAccessActionWrite:
		return ResourceRelationWriteByApp, ScopeResourceWrite, nil
	default:
		return "", "", ErrInvalidResourceAccess
	}
}

func openPlatformAppFGAUser(appID int64) string {
	return fmt.Sprintf("open_platform_app:%d", appID)
}

func resourceFGAObject(resourceType string, resourceID string) string {
	return resourceType + ":" + resourceID
}

func splitFGAObject(object string) (string, string, error) {
	parts := strings.SplitN(object, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ErrInvalidResourceAccess
	}
	return parts[0], parts[1], nil
}
