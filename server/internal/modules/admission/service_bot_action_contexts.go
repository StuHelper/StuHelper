package admission

import (
	"context"
	"time"
)

type admissionGuildKey struct {
	Platform string
	GuildID  string
}

type admissionFailureKey struct {
	Platform string
	GuildID  string
	QQID     string
}

type pendingActionLookupKeySet struct {
	guilds   []admissionGuildKey
	failures []admissionFailureKey
}

type pendingActionSeed struct {
	action   BotAction
	deadline time.Time
}

type pendingActionContexts struct {
	policies map[admissionGuildKey]*AdmissionPolicy
	failures map[admissionFailureKey]*AdmissionFailure
	now      func() time.Time
}

func (s *Service) pendingActionContexts(
	ctx context.Context,
	sessions []AdmissionSession,
	seeds []pendingActionSeed,
) (pendingActionContexts, error) {
	keys := pendingActionLookupKeys(sessions, seeds)
	policies, err := s.repo.ListPoliciesByGuildKeys(ctx, keys.guilds)
	if err != nil {
		return pendingActionContexts{}, err
	}
	failures, err := s.repo.ListAdmissionFailuresByKeys(ctx, keys.failures)
	if err != nil {
		return pendingActionContexts{}, err
	}
	return pendingActionContexts{policies: policies, failures: failures, now: s.now}, nil
}

func pendingActionSeeds(sessions []AdmissionSession, now time.Time) []pendingActionSeed {
	seeds := make([]pendingActionSeed, len(sessions))
	for i := range sessions {
		action, deadline := resolvePendingAction(&sessions[i], now)
		seeds[i] = pendingActionSeed{action: action, deadline: deadline}
	}
	return seeds
}

func (c pendingActionContexts) policyFor(session *AdmissionSession) *AdmissionPolicy {
	key := admissionGuildKey{Platform: session.Platform, GuildID: session.GuildID}
	if policy, ok := c.policies[key]; ok {
		return policy
	}
	return defaultAdmissionPolicy(session.Platform, session.GuildID, c.now())
}

func (c pendingActionContexts) failureFor(session *AdmissionSession) *AdmissionFailure {
	key := admissionFailureKey{Platform: session.Platform, GuildID: session.GuildID, QQID: session.QQID}
	if failure, ok := c.failures[key]; ok {
		return failure
	}
	return nil
}

func pendingActionLookupKeys(sessions []AdmissionSession, seeds []pendingActionSeed) pendingActionLookupKeySet {
	guilds := map[admissionGuildKey]struct{}{}
	failures := map[admissionFailureKey]struct{}{}
	for i := range sessions {
		if seeds[i].action != BotActionKick {
			continue
		}
		guilds[admissionGuildKey{Platform: sessions[i].Platform, GuildID: sessions[i].GuildID}] = struct{}{}
		failures[admissionFailureKey{
			Platform: sessions[i].Platform,
			GuildID:  sessions[i].GuildID,
			QQID:     sessions[i].QQID,
		}] = struct{}{}
	}
	return pendingActionLookupKeySet{
		guilds:   admissionGuildKeys(guilds),
		failures: admissionFailureKeys(failures),
	}
}

func admissionGuildKeys(values map[admissionGuildKey]struct{}) []admissionGuildKey {
	keys := make([]admissionGuildKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func admissionFailureKeys(values map[admissionFailureKey]struct{}) []admissionFailureKey {
	keys := make([]admissionFailureKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
