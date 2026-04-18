import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { ADMIN_REVIEWS_MANAGE, hasCapability } from '@stuhelper/shared/constants'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

import type { Review } from '@stuhelper/shared/review'

export function useReviewAdmin(onRefresh: () => void) {
  const { t } = useI18n()
  const toast = useToast()
  const authStore = useAuthStore()

  const canManageReviews = computed(() =>
    hasCapability(authStore.globalCapabilities, ADMIN_REVIEWS_MANAGE),
  )

  const showModerationDialog = ref(false)
  const showEditDialog = ref(false)
  const moderatingReviewID = ref('')
  const editingReview = ref<Review | null>(null)

  function openModeration(r: Review) {
    moderatingReviewID.value = r.id
    showModerationDialog.value = true
  }

  function openEdit(r: Review) {
    editingReview.value = r
    showEditDialog.value = true
  }

  async function handleModerate(reason: string) {
    showModerationDialog.value = false
    try {
      await api.admin.updateReview(moderatingReviewID.value, { action: 'hide', reason })
      toast.success(t('review.admin.moderateSuccess'))
      onRefresh()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
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
    if (!editingReview.value) return
    showEditDialog.value = false
    try {
      await api.admin.editReview(editingReview.value.id, payload)
      toast.success(t('review.admin.editSuccess'))
      onRefresh()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.admin.actionFailed')))
    }
  }

  return {
    canManageReviews,
    showModerationDialog,
    showEditDialog,
    moderatingReviewID,
    editingReview,
    openModeration,
    openEdit,
    handleModerate,
    handleRestore,
    handleAdminEdit,
  }
}
