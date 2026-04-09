/**
 * Composable: delete-with-confirmation for a review card
 *
 * Manages the two-click delete flow: first click shows "Confirm",
 * second click performs the deletion.
 * Extracted from ReviewCard.vue for single-responsibility.
 */
import { ref, watch } from 'vue'
import type { Review } from '@/types/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export function useReviewDelete(
  reviewGetter: () => Review,
  t: (key: string) => string,
  emitDeleted: (id: string) => void,
) {
  const toast = useToast()

  const deleting = ref(false)
  const deleteConfirming = ref(false)

  // Reset confirmation when review changes
  watch(() => reviewGetter().id, () => {
    deleteConfirming.value = false
  })

  function handleDeleteOwn() {
    if (!deleteConfirming.value) {
      deleteConfirming.value = true
      return
    }
    confirmDeleteOwn()
  }

  function cancelDelete() {
    deleteConfirming.value = false
  }

  async function confirmDeleteOwn() {
    deleting.value = true
    try {
      const review = reviewGetter()
      await api.review.deleteReview(review.id)
      toast.success(t('review.review.deleteSuccess'))
      emitDeleted(review.id)
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.deleteFailed')))
    } finally {
      deleting.value = false
      deleteConfirming.value = false
    }
  }

  return {
    deleting,
    deleteConfirming,
    handleDeleteOwn,
    cancelDelete,
  }
}
