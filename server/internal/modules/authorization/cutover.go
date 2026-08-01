package authorization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
)

var scopedCutoverRoles = []Role{
	RoleSchoolAdmin,
	RoleSectionAdmin,
	RoleSectionModerator,
	RoleSectionReviewer,
}

func (s *Service) RequireAuthorityCutoverComplete(ctx context.Context) error {
	status, err := s.AuthorityCutoverStatus(ctx)
	if err != nil {
		return err
	}
	if !status.Completed {
		return ErrAuthorityCutoverIncomplete
	}
	return nil
}

// AuthorityCutoverStatus exposes the durable one-time cutover marker to the
// deployment command without allowing callers to mutate the marker directly.
func (s *Service) AuthorityCutoverStatus(ctx context.Context) (AuthorityCutoverStatus, error) {
	return s.repo.AuthorityCutoverStatus(ctx)
}

func (s *Service) ImportLegacyAuthority(
	ctx context.Context,
	users []AuthorityCutoverUser,
	snapshot LegacyAuthoritySnapshot,
	tupleReader AuthorityCutoverTupleReader,
) (AuthorityCutoverResult, error) {
	status, err := s.AuthorityCutoverStatus(ctx)
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	if status.Completed {
		return AuthorityCutoverResult{
			Changed:            false,
			SourceDigest:       status.SourceDigest,
			ImportedGrantCount: status.ImportedGrantCount,
		}, nil
	}
	if tupleReader == nil {
		return AuthorityCutoverResult{}, fmt.Errorf("%w: tuple reader is required", ErrAuthorityCutoverConflict)
	}
	organization := strings.TrimSpace(snapshot.Organization)
	if organization == "" {
		return AuthorityCutoverResult{}, fmt.Errorf("%w: Casdoor organization is required", ErrAuthorityCutoverConflict)
	}
	if len(users) > 0 && len(snapshot.Users) == 0 {
		return AuthorityCutoverResult{}, fmt.Errorf("%w: Casdoor snapshot is empty for a non-empty user database", ErrAuthorityCutoverConflict)
	}
	identitiesByAlias, err := indexLegacyAuthorityIdentities(organization, snapshot.Users)
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	roleMembers := normalizeLegacyRoleMembers(snapshot.RoleMembers)
	localIdentities := make(map[int64]LegacyAuthorityIdentity, len(users))
	matchedIdentityCount := 0
	for _, user := range users {
		identity, ok := identitiesByAlias[strings.TrimSpace(user.ProviderSubject)]
		if !ok {
			continue
		}
		localIdentities[user.InternalUserID] = identity
		matchedIdentityCount++
	}
	if len(users) > 0 && matchedIdentityCount == 0 {
		return AuthorityCutoverResult{}, fmt.Errorf("%w: no PostgreSQL user matched the Casdoor organization snapshot", ErrAuthorityCutoverConflict)
	}

	grantByKey := make(map[string]AuthorityCutoverGrant)
	for userID, identity := range localIdentities {
		if identity.OrganizationAdmin && !identity.ForbiddenOrDeleted {
			addAuthorityCutoverGrant(grantByKey, AuthorityCutoverGrant{
				SubjectUserID: userID,
				Role:          RoleSuperAdmin,
				Source:        GrantSourceCasdoorOrganizationAdmin,
			})
		}
	}

	schools, err := s.repo.ListAuthorityCutoverSchoolIDs(ctx)
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	skippedTuples := 0
	for _, schoolID := range schools {
		schoolObject := "school:" + strconv.FormatInt(schoolID, 10)
		skipped, readErr := collectLegacyScopedTupleGrants(
			ctx,
			tupleReader,
			schoolObject,
			"admin",
			RoleSchoolAdmin,
			schoolID,
			"",
			localIdentities,
			roleMembers[RoleSchoolAdmin],
			grantByKey,
		)
		if readErr != nil {
			return AuthorityCutoverResult{}, readErr
		}
		skippedTuples += skipped

		sectionID := fga.ReviewModerationSectionID(strconv.FormatInt(schoolID, 10))
		sectionObject := "section:" + sectionID
		for _, role := range []Role{RoleSectionAdmin, RoleSectionModerator, RoleSectionReviewer} {
			skipped, readErr = collectLegacyScopedTupleGrants(
				ctx,
				tupleReader,
				sectionObject,
				string(role),
				role,
				schoolID,
				sectionID,
				localIdentities,
				roleMembers[role],
				grantByKey,
			)
			if readErr != nil {
				return AuthorityCutoverResult{}, readErr
			}
			skippedTuples += skipped
		}
	}

	grants := make([]AuthorityCutoverGrant, 0, len(grantByKey))
	for _, grant := range grantByKey {
		grants = append(grants, grant)
	}
	sortAuthorityCutoverGrants(grants)
	result, err := s.CompleteAuthorityCutover(ctx, AuthorityCutoverInput{Grants: grants})
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	result.SkippedTupleCount = skippedTuples
	return result, nil
}

func (s *Service) CompleteAuthorityCutover(
	ctx context.Context,
	input AuthorityCutoverInput,
) (AuthorityCutoverResult, error) {
	if s.projection == nil {
		return AuthorityCutoverResult{}, fmt.Errorf("%w: projection client is required", ErrAuthorityCutoverConflict)
	}
	normalized, err := normalizeAuthorityCutoverInput(input)
	if err != nil {
		return AuthorityCutoverResult{}, err
	}
	tuples := make([]fga.Tuple, 0, len(normalized.Grants))
	for _, inputGrant := range normalized.Grants {
		tuple, err := tupleForGrant(Grant{
			SubjectUserID: inputGrant.SubjectUserID,
			Role:          inputGrant.Role,
			Source:        inputGrant.Source,
			SchoolID:      inputGrant.SchoolID,
			SectionID:     inputGrant.SectionID,
		})
		if err != nil {
			return AuthorityCutoverResult{}, fmt.Errorf("%w: %v", ErrAuthorityCutoverConflict, err)
		}
		tuples = append(tuples, tuple)
	}
	if err := s.projection.WriteMissingTuples(ctx, tuples); err != nil {
		return AuthorityCutoverResult{}, fmt.Errorf("verify cutover tuples: write missing tuples: %w", err)
	}
	for _, tuple := range tuples {
		exists, err := s.projection.TupleExists(ctx, tuple)
		if err != nil {
			return AuthorityCutoverResult{}, fmt.Errorf("verify cutover tuple: %w", err)
		}
		if !exists {
			return AuthorityCutoverResult{}, fmt.Errorf(
				"%w: verified tuple is absent for %s#%s@%s",
				ErrAuthorityCutoverConflict,
				tuple.Object,
				tuple.Relation,
				tuple.User,
			)
		}
	}
	return s.repo.ApplyAuthorityCutover(ctx, normalized)
}

func normalizeAuthorityCutoverInput(input AuthorityCutoverInput) (AuthorityCutoverInput, error) {
	grants := append([]AuthorityCutoverGrant(nil), input.Grants...)
	seen := make(map[string]struct{}, len(grants))
	for index := range grants {
		grant := &grants[index]
		if grant.SubjectUserID <= 0 {
			return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid subject user", ErrAuthorityCutoverConflict)
		}
		switch grant.Role {
		case RoleSuperAdmin:
			if grant.Source != GrantSourceCasdoorOrganizationAdmin || grant.SchoolID != nil || grant.SectionID != nil {
				return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid super_admin source or scope", ErrAuthorityCutoverConflict)
			}
		case RoleSchoolAdmin:
			if grant.Source != GrantSourceManual || !validPositiveID(grant.SchoolID) || grant.SectionID != nil {
				return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid school_admin source or scope", ErrAuthorityCutoverConflict)
			}
		case RoleSectionAdmin, RoleSectionModerator, RoleSectionReviewer:
			if grant.Source != GrantSourceManual || !validPositiveID(grant.SchoolID) || grant.SectionID == nil {
				return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid section role source or scope", ErrAuthorityCutoverConflict)
			}
			expectedSection := fga.ReviewModerationSectionID(strconv.FormatInt(*grant.SchoolID, 10))
			sectionID := strings.TrimSpace(*grant.SectionID)
			if sectionID != expectedSection {
				return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid section scope", ErrAuthorityCutoverConflict)
			}
			grant.SectionID = &sectionID
		default:
			return AuthorityCutoverInput{}, fmt.Errorf("%w: unsupported role %q", ErrAuthorityCutoverConflict, grant.Role)
		}
		key := authorityCutoverGrantKey(*grant)
		if _, ok := seen[key]; ok {
			return AuthorityCutoverInput{}, fmt.Errorf("%w: duplicate grant", ErrAuthorityCutoverConflict)
		}
		seen[key] = struct{}{}
	}
	sortAuthorityCutoverGrants(grants)
	digest, err := authorityCutoverDigest(grants)
	if err != nil {
		return AuthorityCutoverInput{}, err
	}
	if provided := strings.TrimSpace(input.SourceDigest); provided != "" {
		if len(provided) != sha256.Size*2 {
			return AuthorityCutoverInput{}, fmt.Errorf("%w: invalid source digest", ErrAuthorityCutoverConflict)
		}
		if _, err := hex.DecodeString(provided); err != nil || provided != strings.ToLower(provided) || provided != digest {
			return AuthorityCutoverInput{}, fmt.Errorf("%w: source digest does not match grants", ErrAuthorityCutoverConflict)
		}
	}
	return AuthorityCutoverInput{SourceDigest: digest, Grants: grants}, nil
}

func indexLegacyAuthorityIdentities(
	organization string,
	identities []LegacyAuthorityIdentity,
) (map[string]LegacyAuthorityIdentity, error) {
	indexed := make(map[string]LegacyAuthorityIdentity)
	for _, identity := range identities {
		identity.ID = strings.TrimSpace(identity.ID)
		identity.Owner = strings.TrimSpace(identity.Owner)
		identity.Name = strings.TrimSpace(identity.Name)
		if identity.Owner != organization || identity.Name == "" {
			continue
		}
		for _, alias := range []string{identity.ID, identity.Name, identity.Owner + "/" + identity.Name} {
			alias = strings.TrimSpace(alias)
			if alias == "" {
				continue
			}
			if existing, ok := indexed[alias]; ok && (existing.Owner != identity.Owner || existing.Name != identity.Name) {
				return nil, fmt.Errorf("%w: ambiguous Casdoor user alias", ErrAuthorityCutoverConflict)
			}
			indexed[alias] = identity
		}
	}
	return indexed, nil
}

func normalizeLegacyRoleMembers(input map[Role][]string) map[Role]map[string]struct{} {
	result := make(map[Role]map[string]struct{}, len(scopedCutoverRoles))
	for _, role := range scopedCutoverRoles {
		members := make(map[string]struct{})
		for _, raw := range input[role] {
			member := strings.TrimSpace(raw)
			if member != "" {
				members[member] = struct{}{}
			}
		}
		result[role] = members
	}
	return result
}

func collectLegacyScopedTupleGrants(
	ctx context.Context,
	reader AuthorityCutoverTupleReader,
	object,
	relation string,
	role Role,
	schoolID int64,
	sectionID string,
	localIdentities map[int64]LegacyAuthorityIdentity,
	roleMembers map[string]struct{},
	grantByKey map[string]AuthorityCutoverGrant,
) (int, error) {
	tuples, err := reader.ReadTuples(ctx, object, relation)
	if err != nil {
		return 0, fmt.Errorf("read legacy authorization tuples for %s#%s: %w", object, relation, err)
	}
	skipped := 0
	for _, tuple := range tuples {
		if tuple.Object != object || tuple.Relation != relation {
			return 0, fmt.Errorf(
				"%w: OpenFGA returned a tuple outside the requested %s#%s filter",
				ErrAuthorityCutoverConflict,
				object,
				relation,
			)
		}
		userID, ok := parseInternalFGAUser(tuple.User)
		if !ok {
			return 0, fmt.Errorf(
				"%w: OpenFGA tuple for %s#%s uses unsupported indirect or malformed subject %q",
				ErrAuthorityCutoverConflict,
				object,
				relation,
				tuple.User,
			)
		}
		identity, ok := localIdentities[userID]
		if !ok {
			return 0, fmt.Errorf(
				"%w: OpenFGA tuple for local user %d has no matching Casdoor identity",
				ErrAuthorityCutoverConflict,
				userID,
			)
		}
		if identity.ForbiddenOrDeleted || !legacyIdentityHasRole(identity, roleMembers) {
			skipped++
			continue
		}
		grant := AuthorityCutoverGrant{
			SubjectUserID: userID,
			Role:          role,
			Source:        GrantSourceManual,
			SchoolID:      &schoolID,
		}
		if sectionID != "" {
			sectionCopy := sectionID
			grant.SectionID = &sectionCopy
		}
		addAuthorityCutoverGrant(grantByKey, grant)
	}
	return skipped, nil
}

func legacyIdentityHasRole(identity LegacyAuthorityIdentity, members map[string]struct{}) bool {
	for _, candidate := range []string{
		strings.TrimSpace(identity.ID),
		strings.TrimSpace(identity.Name),
		strings.TrimSpace(identity.Owner) + "/" + strings.TrimSpace(identity.Name),
	} {
		if _, ok := members[candidate]; ok {
			return true
		}
	}
	return false
}

func parseInternalFGAUser(raw string) (int64, bool) {
	rawID, ok := strings.CutPrefix(strings.TrimSpace(raw), "user:")
	if !ok {
		return 0, false
	}
	userID, err := strconv.ParseInt(rawID, 10, 64)
	return userID, err == nil && userID > 0 && strconv.FormatInt(userID, 10) == rawID
}

func addAuthorityCutoverGrant(target map[string]AuthorityCutoverGrant, grant AuthorityCutoverGrant) {
	target[authorityCutoverGrantKey(grant)] = grant
}

func authorityCutoverGrantKey(grant AuthorityCutoverGrant) string {
	schoolID := ""
	if grant.SchoolID != nil {
		schoolID = strconv.FormatInt(*grant.SchoolID, 10)
	}
	sectionID := ""
	if grant.SectionID != nil {
		sectionID = *grant.SectionID
	}
	return strconv.FormatInt(grant.SubjectUserID, 10) + "|" + string(grant.Role) + "|" + schoolID + "|" + sectionID
}

func sortAuthorityCutoverGrants(grants []AuthorityCutoverGrant) {
	sort.Slice(grants, func(i, j int) bool {
		return authorityCutoverGrantKey(grants[i]) < authorityCutoverGrantKey(grants[j])
	})
}

func authorityCutoverDigest(grants []AuthorityCutoverGrant) (string, error) {
	canonical := make([]map[string]any, 0, len(grants))
	for _, grant := range grants {
		canonical = append(canonical, map[string]any{
			"subjectUserId": grant.SubjectUserID,
			"role":          grant.Role,
			"source":        grant.Source,
			"schoolId":      grant.SchoolID,
			"sectionId":     grant.SectionID,
		})
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal authorization cutover digest: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
