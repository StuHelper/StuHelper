/**
 * 管理管理员对单条评测的审核、恢复与编辑动作。
 */
import { ref } from 'vue'
import type { Review } from '@stuhelper/shared/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export function useReviewModeration(
  reviewGetter: () => Review,
  t: (key: string) => string,
  emitModerated: () => void,
) {
  const toast = useToast()

  const showModerationDialog = ref(false)
  const showEditDialog = ref(false)
  const moderationSubmitting = ref(false)
  const editSubmitting = ref(false)

  async function handleModerate(reason: string) {
    if (moderationSubmitting.value) return
    moderationSubmitting.value = true
    try {
      await api.admin.updateReview(reviewGetter().id, { action: 'hide', reason })
      showModerationDialog.value = false
      toast.success(t('review.admin.moderateSuccess'))
      emitModerated()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    } finally {
      moderationSubmitting.value = false
    }
  }

  async function handleRestore() {
    try {
      await api.admin.updateReview(reviewGetter().id, { action: 'restore' })
      toast.success(t('review.admin.restoreSuccess'))
      emitModerated()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    }
  }

  async function handleAdminEdit(payload: { title: string; content: string; reason: string }) {
    if (editSubmitting.value) return
    editSubmitting.value = true
    try {
      await api.admin.editReview(reviewGetter().id, payload)
      showEditDialog.value = false
      toast.success(t('review.admin.editSuccess'))
      emitModerated()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    } finally {
      editSubmitting.value = false
    }
  }

  return {
    showModerationDialog,
    showEditDialog,
    moderationSubmitting,
    editSubmitting,
    handleModerate,
    handleRestore,
    handleAdminEdit,
  }
}
