import { ref, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import ReplyForm from '@/components/business/review/ReplyForm.vue'

import type { Reply } from '@stuhelper/shared/reply'

type ReplyFormExpose = InstanceType<typeof ReplyForm>
type ReplyFormRefValue = ReplyFormExpose | ReplyFormExpose[] | null

function clearReplyForm(refValue: ReplyFormRefValue) {
  const form = Array.isArray(refValue) ? refValue.find(Boolean) : refValue
  form?.clear()
}

export function useReviewReplies() {
  const { t } = useI18n()
  const toast = useToast()
  const router = useRouter()
  const authStore = useAuthStore()

  const expandedReviewID = ref<string | null>(null)
  const replies = ref<Reply[]>([])
  const repliesLoading = ref(false)
  const repliesError = ref(false)
  const replySubmitting = ref(false)
  const replyFormRef = ref<ReplyFormRefValue>(null)
  const replyCountMap = reactive<Record<string, number>>({})
  let repliesRequestSeq = 0

  async function toggleExpand(reviewId: string) {
    if (expandedReviewID.value === reviewId) {
      expandedReviewID.value = null
      return
    }
    expandedReviewID.value = reviewId
    await loadReplies(reviewId)
  }

  async function loadReplies(reviewId: string) {
    const requestSeq = ++repliesRequestSeq
    repliesLoading.value = true
    repliesError.value = false
    try {
      const res = await api.reply.getReplies(reviewId)
      if (requestSeq !== repliesRequestSeq || expandedReviewID.value !== reviewId) return
      replies.value = res.data?.data?.list || []
      replyCountMap[reviewId] = res.data?.data?.total || 0
    } catch (_error) { void _error;
      if (requestSeq !== repliesRequestSeq || expandedReviewID.value !== reviewId) return
      replies.value = []
      repliesError.value = true
    } finally {
      if (requestSeq === repliesRequestSeq) {
        repliesLoading.value = false
      }
    }
  }

  async function handleReplySubmit(reviewId: string, content: string) {
    if (!authStore.bootstrapCompleted) {
      await authStore.bootstrapSession()
    }
    if (!authStore.isAuthenticated) {
      await router.push({
        name: 'login',
        query: { redirect: router.currentRoute.value.fullPath },
      })
      return
    }

    replySubmitting.value = true
    try {
      const res = await api.reply.createReply(reviewId, { content })
      if (res.data?.data) {
        replies.value = [...replies.value, res.data.data]
        replyCountMap[reviewId] = (replyCountMap[reviewId] ?? 0) + 1
        clearReplyForm(replyFormRef.value)
        toast.success(t('review.review.replySuccess'))
      }
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.replyFailed')))
    } finally {
      replySubmitting.value = false
    }
  }

  async function handleDeleteReply(id: string) {
    if (!expandedReviewID.value) return
    try {
      await api.reply.deleteReply(id)
      replies.value = replies.value.filter(r => r.id !== id)
      replyCountMap[expandedReviewID.value] = Math.max(0, (replyCountMap[expandedReviewID.value] ?? 0) - 1)
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.deleteFailed')))
    }
  }

  return {
    expandedReviewID,
    replies,
    repliesLoading,
    repliesError,
    replySubmitting,
    replyFormRef,
    replyCountMap,
    toggleExpand,
    loadReplies,
    handleReplySubmit,
    handleDeleteReply,
  }
}
