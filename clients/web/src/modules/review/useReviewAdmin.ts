import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import {
  canEditReviewContent as canEditReviewContentAccess,
  canManageReviews as canManageReviewAccess,
} from '@/utils/adminAccess'

import type { Review } from '@stuhelper/shared/review'

export function useReviewAdmin(onRefresh: () => void) {
  const { t } = useI18n()
  const toast = useToast()
  const authStore = useAuthStore()

  const canManageReviews = computed(() => canManageReviewAccess(authStore.user))
  const canEditReviewContent = computed(() => canEditReviewContentAccess(authStore.user))

  const showModerationDialog = ref(false)
  const showEditDialog = ref(false)
  const moderatingReviewID = ref('')
  const editingReview = ref<Review | null>(null)
  const moderationSubmitting = ref(false)
  const editSubmitting = ref(false)

  function openModeration(r: Review) {
    moderatingReviewID.value = r.id
    showModerationDialog.value = true
  }

  function openEdit(r: Review) {
    editingReview.value = r
    showEditDialog.value = true
  }

  async function handleModerate(reason: string) {
    if (moderationSubmitting.value) return
    moderationSubmitting.value = true
    try {
      await api.admin.updateReview(moderatingReviewID.value, { action: 'hide', reason })
      showModerationDialog.value = false
      toast.success(t('review.admin.moderateSuccess'))
      onRefresh()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    } finally {
      moderationSubmitting.value = false
    }
  }

  async function handleRestore(r: Review) {
    try {
      await api.admin.updateReview(r.id, { action: 'restore' })
      toast.success(t('review.admin.restoreSuccess'))
      onRefresh()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    }
  }

  async function handleAdminEdit(payload: { title: string; content: string; reason: string }) {
    if (!editingReview.value || editSubmitting.value) return
    editSubmitting.value = true
    try {
      await api.admin.editReview(editingReview.value.id, payload)
      showEditDialog.value = false
      toast.success(t('review.admin.editSuccess'))
      onRefresh()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    } finally {
      editSubmitting.value = false
    }
  }

  return {
    canManageReviews,
    canEditReviewContent,
    showModerationDialog,
    showEditDialog,
    moderatingReviewID,
    editingReview,
    moderationSubmitting,
    editSubmitting,
    openModeration,
    openEdit,
    handleModerate,
    handleRestore,
    handleAdminEdit,
  }
}
