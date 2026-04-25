/**
 * 管理单条评测的回复列表、提交与删除。
 */
import { ref, watch } from 'vue'
import type { Reply } from '@stuhelper/shared/reply'
import type { Review } from '@stuhelper/shared/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import type ReplyForm from './ReplyForm.vue'

export function useReviewReply(
  reviewGetter: () => Review,
  t: (key: string) => string,
) {
  const toast = useToast()

  const replies = ref<Reply[]>([])
  const repliesLoading = ref(false)
  const repliesError = ref(false)
  const replySubmitting = ref(false)
  const replyCount = ref(reviewGetter().replyCount ?? 0)
  const replyFormRef = ref<InstanceType<typeof ReplyForm> | null>(null)

  // 父级数据刷新时，仅在本地未改动计数的情况下同步 replyCount
  let replyCountDirty = false
  watch(() => reviewGetter().replyCount, (val) => {
    if (val !== undefined && !replyCountDirty) replyCount.value = val
  })

  // 评测数据整体刷新后，允许重新接收服务端计数
  watch(reviewGetter, () => {
    replyCountDirty = false
  })

  async function loadReplies() {
    repliesLoading.value = true
    repliesError.value = false
    try {
      const res = await api.reply.getReplies(reviewGetter().id)
      replies.value = res.data?.data?.list || []
      replyCount.value = res.data?.data?.total || 0
      replyCountDirty = false
    } catch (_error) { void _error
      replies.value = []
      repliesError.value = true
    } finally {
      repliesLoading.value = false
    }
  }

  async function handleReplySubmit(content: string) {
    replySubmitting.value = true
    try {
      const res = await api.reply.createReply(reviewGetter().id, { content })
      if (res.data?.data) {
        replies.value = [...replies.value, res.data.data]
        replyCount.value++
        replyCountDirty = true
        replyFormRef.value?.clear()
        toast.success(t('review.review.replySuccess'))
      }
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.replyFailed')))
    } finally {
      replySubmitting.value = false
    }
  }

  async function handleDeleteReply(id: string) {
    try {
      await api.reply.deleteReply(id)
      replies.value = replies.value.filter((r) => r.id !== id)
      replyCount.value = Math.max(0, replyCount.value - 1)
      replyCountDirty = true
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.deleteFailed')))
    }
  }

  return {
    replies,
    repliesLoading,
    repliesError,
    replySubmitting,
    replyCount,
    replyFormRef,
    loadReplies,
    handleReplySubmit,
    handleDeleteReply,
  }
}
