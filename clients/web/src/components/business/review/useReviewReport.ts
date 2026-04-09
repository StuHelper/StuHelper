/**
 * Composable: report handling for a review card
 *
 * Manages report menu visibility, reason selection, and API submission.
 * Extracted from ReviewCard.vue for single-responsibility.
 */
import { ref } from 'vue'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { useToast } from '@/composables/useToast'

export const REPORT_REASONS = ['spam', 'inappropriate', 'harassment', 'false_info', 'other'] as const
export type ReportReason = typeof REPORT_REASONS[number]

export function useReviewReport(
  reviewIdGetter: () => string,
  t: (key: string) => string,
) {
  const toast = useToast()

  const showReportMenu = ref(false)
  const reporting = ref(false)

  function toggleReportMenu() {
    showReportMenu.value = !showReportMenu.value
  }

  async function handleReport(reason: ReportReason) {
    reporting.value = true
    try {
      await api.review.reportReview(reviewIdGetter(), { reason })
      toast.success(t('review.review.reportSuccess'))
      showReportMenu.value = false
    } catch (err: unknown) {
      toast.error(getErrorMessage(err, t('review.review.reportFailed')))
    } finally {
      reporting.value = false
    }
  }

  return {
    showReportMenu,
    reporting,
    reportReasons: REPORT_REASONS,
    toggleReportMenu,
    handleReport,
  }
}
