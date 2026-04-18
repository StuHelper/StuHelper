/**
 * Composable: delete handling for a review card
 *
 * Manages deletion state and emits the deleted review id on success.
 */
import { ref } from 'vue'
import type { Review } from '@stuhelper/shared/review'
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

  async function handleDeleteOwn() {
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
    }
  }

  return {
    deleting,
    handleDeleteOwn,
  }
}
