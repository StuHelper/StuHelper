/**
 * Composable: admin moderation actions for a review card
 *
 * Manages moderation dialog, admin edit dialog, restore, and admin edit actions.
 * Extracted from ReviewCard.vue for single-responsibility.
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

  async function handleModerate(reason: string) {
    showModerationDialog.value = false
    try {
      await api.admin.updateReview(reviewGetter().id, { action: 'hide', reason })
      toast.success(t('review.admin.moderateSuccess'))
      emitModerated()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
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
    showEditDialog.value = false
    try {
      await api.admin.editReview(reviewGetter().id, payload)
      toast.success(t('review.admin.editSuccess'))
      emitModerated()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    }
  }

  return {
    showModerationDialog,
    showEditDialog,
    handleModerate,
    handleRestore,
    handleAdminEdit,
  }
}
