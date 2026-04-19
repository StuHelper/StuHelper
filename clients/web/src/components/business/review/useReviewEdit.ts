/**
 * 管理单条评测的行内编辑状态与保存流程。
 */
import { ref } from 'vue'
import type { Review } from '@stuhelper/shared/review'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export function useReviewEdit(
  reviewGetter: () => Review,
  t: (key: string) => string,
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
    if (!trimmed) return
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
