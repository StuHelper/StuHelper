import { ref, reactive } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
  type VoteType,
} from '@/components/business/review/reviewVoteState'

import type { Review } from '@stuhelper/shared/review'

export function useReviewVoting() {
  const { t } = useI18n()
  const toast = useToast()

  const reviewVotes = reactive<Record<string, VoteType | null>>({})
  const likeOffsets = reactive<Record<string, number>>({})
  const dislikeOffsets = reactive<Record<string, number>>({})
  const votingReviews = ref(new Set<string>())

  function displayLikeCount(r: Review): number {
    return getDisplayVoteCount(r.likeCount, likeOffsets[r.id] ?? 0)
  }

  function displayDislikeCount(r: Review): number {
    return getDisplayVoteCount(r.dislikeCount, dislikeOffsets[r.id] ?? 0)
  }

  async function handleVote(r: Review, type: VoteType) {
    if (votingReviews.value.has(r.id)) return
    votingReviews.value.add(r.id)

    const id = r.id
    const previousState = createReviewVoteState({
      userVote: reviewVotes[id] ?? null,
      likeOffset: likeOffsets[id] ?? 0,
      dislikeOffset: dislikeOffsets[id] ?? 0,
    })
    const nextState = applyOptimisticVote(previousState, type)

    reviewVotes[id] = nextState.userVote
    likeOffsets[id] = nextState.likeOffset
    dislikeOffsets[id] = nextState.dislikeOffset

    try {
      await api.review.voteReview(id, { voteType: type })
    } catch (err: unknown) {
      reviewVotes[id] = previousState.userVote
      likeOffsets[id] = previousState.likeOffset
      dislikeOffsets[id] = previousState.dislikeOffset
      toast.error(getErrorMessage(err, t('review.review.voteFailed')))
    } finally {
      votingReviews.value.delete(id)
    }
  }

  return {
    reviewVotes,
    displayLikeCount,
    displayDislikeCount,
    handleVote,
  }
}
