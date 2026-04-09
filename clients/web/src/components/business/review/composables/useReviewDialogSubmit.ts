import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { buildCreateReviewPayload } from '../reviewPayload'
import { clearLocalReviewDraft } from '../reviewDialogDraftState'
import type { ReviewRatings } from '@/types/review'

interface SubmitFormState {
  courseID: number
  teacherID: number | null
  termID: string
  title: string
  content: string
  grade: string
  ratings: ReviewRatings
}

export function useReviewDialogSubmit() {
  const { t } = useI18n()
  const toast = useToast()
  const router = useRouter()
  const authStore = useAuthStore()

  const submitting = ref(false)
  const redirectCountdown = ref(0)

  let countdownTimer: ReturnType<typeof setInterval> | null = null

  function clearCountdownTimer() {
    if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
    redirectCountdown.value = 0
  }

  function startRedirectCountdown(onDone: () => void) {
    clearCountdownTimer()
    redirectCountdown.value = 3
    countdownTimer = setInterval(() => {
      redirectCountdown.value--
      if (redirectCountdown.value <= 0) {
        clearCountdownTimer()
        onDone()
        authStore.login()
      }
    }, 1000)
  }

  async function submit(
    state: SubmitFormState,
    callbacks: {
      saveLocalDraft: () => void
      onSuccess: () => void
      onRedirect: () => void
    },
  ): Promise<void> {
    if (submitting.value) return
    submitting.value = true

    try {
      if (!authStore.isAuthenticated) {
        callbacks.saveLocalDraft()
        sessionStorage.setItem('draft_redirect', router.currentRoute.value.fullPath)
        sessionStorage.setItem('draft_pending', '1')
        startRedirectCountdown(callbacks.onRedirect)
        return
      }

      const checkRes = await api.review.checkContent({ content: state.content.trim() })
      const checkResult = checkRes.data?.data
      if (checkResult && !checkResult.isValid) {
        if (checkResult.level === 'block') {
          toast.error(t('review.post.contentBlocked'))
          return
        }
        if (checkResult.level === 'warn') {
          toast.warning(t('review.post.contentWarning'))
        }
      }

      await api.review.createReview(buildCreateReviewPayload({
        courseID: state.courseID,
        teacherID: state.teacherID ?? undefined,
        termID: state.termID,
        title: state.title.trim(),
        content: state.content.trim(),
        grade: state.grade?.trim() || undefined,
        ratings: state.ratings,
      }))

      clearLocalReviewDraft(state.courseID)
      try { await api.draft.deleteDraft(state.courseID) } catch { /* ignore */ }
      toast.success(t('review.post.success'))
      callbacks.onSuccess()
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.post.failed')))
    } finally {
      submitting.value = false
    }
  }

  return {
    submitting,
    redirectCountdown,
    clearCountdownTimer,
    submit,
  }
}
