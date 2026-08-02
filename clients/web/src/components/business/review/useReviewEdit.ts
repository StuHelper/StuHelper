/**
 * 管理单条评测的行内编辑状态与保存流程。
 */
import { ref } from 'vue'
import {
  REVIEW_CONTENT_MAX_LENGTH,
  REVIEW_CONTENT_MIN_LENGTH,
} from '@stuhelper/shared/constants'
import type { Review } from '@stuhelper/shared/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export function useReviewEdit(
  reviewGetter: () => Review,
  t: (key: string, params?: Record<string, string | number>) => string,
  onUpdated: (id: string, content: string) => void,
) {
  const toast = useToast()

  const editing = ref(false)
  const editContent = ref('')
  const saving = ref(false)

  function startEditing() {
    editContent.value = reviewGetter().content
    editing.value = true
  }

  function cancelEditing() {
    editing.value = false
    editContent.value = ''
  }

  async function handleSaveEdit() {
    const trimmed = editContent.value.trim()
    if (trimmed.length < REVIEW_CONTENT_MIN_LENGTH) {
      toast.error(t('review.validation.contentTooShort', { min: REVIEW_CONTENT_MIN_LENGTH }))
      return
    }
    if (trimmed.length > REVIEW_CONTENT_MAX_LENGTH) {
      toast.error(t('review.validation.contentTooLong', { max: REVIEW_CONTENT_MAX_LENGTH }))
      return
    }
    saving.value = true
    try {
      const review = reviewGetter()
      await api.review.updateReview(review.id, {
        content: trimmed,
        ratings: review.ratings,
      })
      toast.success(t('review.review.editSuccess'))
      editing.value = false
      onUpdated(review.id, trimmed)
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.editFailed')))
    } finally {
      saving.value = false
    }
  }

  return {
    editing,
    editContent,
    saving,
    startEditing,
    cancelEditing,
    handleSaveEdit,
  }
}
