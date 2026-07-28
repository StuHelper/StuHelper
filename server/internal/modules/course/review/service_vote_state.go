package review

import (
	"context"
	"strings"
)

func (s *Service) populateUserVotes(
	ctx context.Context,
	userHash string,
	reviews []Review,
) error {
	userHash = strings.TrimSpace(userHash)
	if userHash == "" || len(reviews) == 0 {
		return nil
	}

	reviewIDs := make([]string, 0, len(reviews))
	seen := make(map[string]struct{}, len(reviews))
	for i := range reviews {
		if reviews[i].ID == "" {
			continue
		}
		if _, exists := seen[reviews[i].ID]; exists {
			continue
		}
		seen[reviews[i].ID] = struct{}{}
		reviewIDs = append(reviewIDs, reviews[i].ID)
	}

	votes, err := s.repo.ListVoteTypes(ctx, userHash, reviewIDs)
	if err != nil {
		return err
	}
	for i := range reviews {
		voteType, ok := votes[reviews[i].ID]
		if !ok {
			reviews[i].UserVote = nil
			continue
		}
		vote := voteType
		reviews[i].UserVote = &vote
	}
	return nil
}

func (s *Service) populateGroupedUserVotes(
	ctx context.Context,
	userHash string,
	grouped map[int64][]Review,
) error {
	if strings.TrimSpace(userHash) == "" || len(grouped) == 0 {
		return nil
	}

	all := make([]Review, 0)
	for _, reviews := range grouped {
		all = append(all, reviews...)
	}
	if err := s.populateUserVotes(ctx, userHash, all); err != nil {
		return err
	}

	votes := make(map[string]*string, len(all))
	for i := range all {
		votes[all[i].ID] = all[i].UserVote
	}
	for courseID, reviews := range grouped {
		for i := range reviews {
			reviews[i].UserVote = votes[reviews[i].ID]
		}
		grouped[courseID] = reviews
	}
	return nil
}

func setKnownUserVote(reviews []Review, voteType string) {
	for i := range reviews {
		vote := voteType
		reviews[i].UserVote = &vote
	}
}
