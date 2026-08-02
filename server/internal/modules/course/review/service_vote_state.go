package review

import (
	"context"
	"strings"
)

func (s *Service) populateUserReviewState(
	ctx context.Context,
	userHash string,
	reviews []Review,
) error {
	userHash = strings.TrimSpace(userHash)
	populateReviewOwnership(userHash, reviews)
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

func (s *Service) populateGroupedUserReviewState(
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
	if err := s.populateUserReviewState(ctx, userHash, all); err != nil {
		return err
	}

	type reviewState struct {
		isOwner bool
		vote    *string
	}
	states := make(map[string]reviewState, len(all))
	for i := range all {
		states[all[i].ID] = reviewState{isOwner: all[i].IsOwner, vote: all[i].UserVote}
	}
	for courseID, reviews := range grouped {
		for i := range reviews {
			state := states[reviews[i].ID]
			reviews[i].IsOwner = state.isOwner
			reviews[i].UserVote = state.vote
		}
		grouped[courseID] = reviews
	}
	return nil
}

func populateReviewOwnership(userHash string, reviews []Review) {
	userHash = strings.TrimSpace(userHash)
	for i := range reviews {
		reviews[i].IsOwner = userHash != "" && reviews[i].UserHash == userHash
	}
}

func setKnownUserVote(reviews []Review, voteType string) {
	for i := range reviews {
		vote := voteType
		reviews[i].UserVote = &vote
	}
}
