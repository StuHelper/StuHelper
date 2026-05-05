package admission

import (
	"context"
	"strings"
)

func (s *Service) ensureBotSessionMemberAllowed(ctx context.Context, input BotSessionCreateInput) error {
	entry, err := s.repo.GetMemberBlacklistAccess(ctx, MemberBlacklistAccessQuery{
		Platform: input.Platform, SubjectType: BlacklistSubjectQQUser,
		SubjectID: input.QQID, GuildID: botSessionGuildIDPtr(input),
	}, s.now())
	if err != nil {
		return err
	}
	if entry != nil {
		return ErrMemberBlacklisted
	}
	return nil
}

func botSessionGuildIDPtr(input BotSessionCreateInput) *string {
	guildID := strings.TrimSpace(input.GuildID)
	if guildID == "" {
		return nil
	}
	return &guildID
}
