import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import type { Course } from '@/types/course'
import type { ReviewRatings } from '@/types/review'
import {
  clearLocalReviewDraft,
  createDraftCourse,
  createLocalReviewDraft,
  getLocalReviewDraftClearedAt,
  loadLocalReviewDraft,
  sanitizeDraftRatings,
  saveLocalReviewDraft,
  type ReviewDialogServerDraft,
} from '../reviewDialogDraftState'

const AUTO_SAVE_DEBOUNCE_MS = 300

interface DraftFormState {
  selectedCourse: Course | null
  selectedTeacherID: number | null
  teacherQuery: string
  title: string
  content: string
  grade: string
  termID: string
  ratings: ReviewRatings
}

interface DraftRestoreCallbacks {
  setSelectedCourse: (course: Course) => void
  loadCourseData: (courseID: number) => Promise<void>
  setTitle: (v: string) => void
  setContent: (v: string) => void
  setGrade: (v: string) => void
  setTermID: (v: string) => void
  setRatings: (v: ReviewRatings) => void
  applyTeacherDraftSelection: (teacherID?: number | null, teacherName?: string) => void
}

export function useReviewDialogDraft() {
  const { t } = useI18n()
  const authStore = useAuthStore()
  const toast = useToast()

  let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
  let restoreVersion = 0

  function saveLocalDraft(state: DraftFormState) {
    if (!state.selectedCourse) return
    saveLocalReviewDraft(createLocalReviewDraft({
      course: state.selectedCourse,
      teacherID: state.selectedTeacherID,
      teacherName: state.teacherQuery.trim() || undefined,
      title: state.title,
      content: state.content,
      grade: state.grade,
      termID: state.termID,
      ratings: state.ratings,
    }))
  }

  async function saveDraftAuto(state: DraftFormState) {
    if (!state.selectedCourse) return
    if (authStore.isAuthenticated) {
      try {
        await api.draft.saveDraft({
          courseID: state.selectedCourse.id,
          teacherID: state.selectedTeacherID ?? undefined,
          termID: state.termID || undefined,
          title: state.title.trim() || undefined,
          content: state.content.trim() || undefined,
          grade: state.grade?.trim() || undefined,
          ratings: Object.keys(state.ratings).length > 0 ? state.ratings : undefined,
        })
      } catch {
        saveLocalDraft(state)
      }
    } else {
      saveLocalDraft(state)
    }
  }

  function scheduleAutoSave(state: DraftFormState) {
    if (!state.selectedCourse) return
    if (autoSaveTimer) clearTimeout(autoSaveTimer)
    autoSaveTimer = setTimeout(() => { saveLocalDraft(state) }, AUTO_SAVE_DEBOUNCE_MS)
  }

  async function tryRestoreDraft(
    validRatingKeys: ReadonlySet<string> | undefined,
    currentCourse: Course | null,
    callbacks: DraftRestoreCallbacks,
  ): Promise<void> {
    const expectedVersion = ++restoreVersion

    const local = loadLocalReviewDraft(
      validRatingKeys,
      currentCourse?.id,
    )
    const clearedAt = getLocalReviewDraftClearedAt(local?.courseID ?? currentCourse?.id)
    let serverDraft: ReviewDialogServerDraft | null = null

    if (authStore.isAuthenticated && (currentCourse || local)) {
      const courseID = currentCourse?.id ?? local?.courseID
      if (courseID) {
        try {
          const res = await api.draft.getDraft(courseID)
          serverDraft = res.data?.data ?? null
        } catch { /* silent */ }
      }
    }

    // Abort if a newer restore call has started
    if (restoreVersion !== expectedVersion) return

    const localTime = local?.updatedAt ?? 0
    const serverTime = serverDraft ? new Date(serverDraft.updatedAt).getTime() : 0
    const latest = Math.max(localTime, serverTime, clearedAt)

    if (latest === 0 || latest === clearedAt) return

    if (latest === localTime && local) {
      if (!currentCourse || currentCourse.id !== local.courseID) {
        callbacks.setSelectedCourse(createDraftCourse(local))
        await callbacks.loadCourseData(local.courseID)
      }
      if (local.title) callbacks.setTitle(local.title)
      if (local.content) callbacks.setContent(local.content)
      if (local.grade) callbacks.setGrade(local.grade)
      callbacks.setTermID(local.termID)
      if (local.ratings && Object.keys(local.ratings).length > 0) callbacks.setRatings(local.ratings)
      callbacks.applyTeacherDraftSelection(local.teacherID ?? null, local.teacherName)
    } else if (serverDraft) {
      if (serverDraft.title) callbacks.setTitle(serverDraft.title)
      if (serverDraft.content) callbacks.setContent(serverDraft.content)
      if (serverDraft.grade) callbacks.setGrade(serverDraft.grade)
      if (serverDraft.termID) callbacks.setTermID(serverDraft.termID)
      callbacks.applyTeacherDraftSelection(serverDraft.teacherID ?? null)
      const sanitizedRatings = sanitizeDraftRatings(serverDraft.ratings)
      if (sanitizedRatings && Object.keys(sanitizedRatings).length > 0) {
        callbacks.setRatings(sanitizedRatings)
      }
    }

    toast.info(t('review.draft.restored'))
  }

  function clearDraft(courseID?: number) {
    clearLocalReviewDraft(courseID)
    if (authStore.isAuthenticated && typeof courseID === 'number') {
      api.draft.deleteDraft(courseID).catch(() => {})
    }
  }

  function cleanupTimers() {
    if (autoSaveTimer) clearTimeout(autoSaveTimer)
  }

  return {
    saveLocalDraft,
    saveDraftAuto,
    scheduleAutoSave,
    tryRestoreDraft,
    clearDraft,
    cleanupTimers,
  }
}
