/**
 * 管理单条评测的投票状态、乐观更新与反馈动画。
 */
import { ref, computed, watch, onScopeDispose } from 'vue'
import type { Review } from '@stuhelper/shared/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
  type VoteType,
} from './reviewVoteState'

export function useReviewVote(
  reviewGetter: () => Review,
  t: (key: string) => string,
) {
  const toast = useToast()

  const userVote = ref<VoteType | null>(null)
  const isVoting = ref(false)
  const likeOffset = ref(0)
  const dislikeOffset = ref(0)
  const likeBounce = ref(false)
  const shaking = ref(false)

  const displayLikes = computed(() =>
    getDisplayVoteCount(reviewGetter().likeCount, likeOffset.value),
  )
  const displayDislikes = computed(() =>
    getDisplayVoteCount(reviewGetter().dislikeCount, dislikeOffset.value),
  )

  // 评测数据刷新后，重置本地投票偏移量
  watch(reviewGetter, () => {
    likeOffset.value = 0
    dislikeOffset.value = 0
    userVote.value = null
  })

  // 清理动画定时器
  let bounceTimer: ReturnType<typeof setTimeout> | null = null
  let shakeTimer: ReturnType<typeof setTimeout> | null = null

  onScopeDispose(() => {
    if (bounceTimer) { clearTimeout(bounceTimer); bounceTimer = null }
    if (shakeTimer) { clearTimeout(shakeTimer); shakeTimer = null }
  })

  async function handleVote(type: VoteType) {
    if (isVoting.value) return
    isVoting.value = true

    const review = reviewGetter()
    const previousState = createReviewVoteState({
      userVote: userVote.value,
      likeOffset: likeOffset.value,
      dislikeOffset: dislikeOffset.value,
    })
    const nextState = applyOptimisticVote(previousState, type)
    const shouldBounceLike = previousState.userVote !== 'like' && nextState.userVote === 'like'
    const targetReviewId = review.id

    userVote.value = nextState.userVote
    likeOffset.value = nextState.likeOffset
    dislikeOffset.value = nextState.dislikeOffset

    if (shouldBounceLike) {
      likeBounce.value = true
      bounceTimer = setTimeout(() => { likeBounce.value = false; bounceTimer = null }, 300)
    }

    try {
      await api.review.voteReview(review.id, { voteType: type })
    } catch (err: unknown) {
      if (reviewGetter().id === targetReviewId) {
        userVote.value = previousState.userVote
        likeOffset.value = previousState.likeOffset
        dislikeOffset.value = previousState.dislikeOffset
        shaking.value = true
        shakeTimer = setTimeout(() => { shaking.value = false; shakeTimer = null }, 300)
        toast.error(getErrorMessage(err, t('review.review.voteFailed')))
      }
    } finally {
      isVoting.value = false
    }
  }

  return {
    userVote,
    isVoting,
    likeBounce,
    shaking,
    displayLikes,
    displayDislikes,
    handleVote,
  }
}
