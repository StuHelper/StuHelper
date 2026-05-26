<template>
  <div class="min-h-screen py-8 px-4">
    <div class="max-w-2xl mx-auto">
      <!-- Back button + title -->
      <div class="flex items-center gap-3 mb-6">
        <button
          class="flex items-center justify-center w-9 h-9 rounded-full bg-transparent border-none cursor-pointer transition-all duration-200 text-text-secondary hover:text-primary hover:bg-bg-hover"
          @click="router.back()"
        >
          <ArrowLeft :size="20" />
        </button>
        <h1 class="m-0 text-2xl font-bold text-text-primary gradient-text">
          {{ t('review.postForm.title') }}
        </h1>
      </div>

      <!-- Form container -->
      <div class="bg-bg-card rounded-2xl shadow-md p-6 md:p-8">
        <DraftIndicator
          v-if="hasMeaningfulDraftInput || draftStore.lastSavedAt"
          class="mb-5 bg-bg-elevated"
          :saving="draftStore.saving"
          :last-saved="draftStore.lastSavedAt"
          :has-draft="draftStore.hasDraft"
          @restore="promptRestoreExistingDraft"
          @delete="discardCurrentDraft"
        />

        <!-- Course search -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.selectCourse') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <div class="relative" ref="searchContainerRef">
            <div
              v-if="selectedCourse && !showDropdown"
              data-testid="review-course-selected"
              class="flex items-center justify-between gap-3 p-3 bg-primary/[0.06] rounded-lg border border-primary/15"
            >
              <span class="min-w-0 truncate font-semibold text-sm text-text-primary">{{ selectedCourse.name }}</span>
              <button
                type="button"
                class="shrink-0 text-xs text-primary cursor-pointer"
                :aria-label="t('common.actions.edit')"
                @click="editCourseSelection"
              >
                {{ t('common.actions.edit') }}
              </button>
            </div>
            <div v-else class="relative">
              <Search
                :size="18"
                class="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted"
              />
              <input
                ref="courseSearchInputRef"
                v-model="courseSearch.query.value"
                type="text"
                class="w-full pl-10 pr-10 px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors"
                :class="{ 'border-danger focus:border-danger focus:ring-danger/20': showErrors && !selectedCourse }"
                :placeholder="t('review.postForm.searchCoursePlaceholder')"
                autocomplete="off"
                @focus="showDropdown = true"
                @keydown="handleCourseKeyDown"
              />
              <button
                v-if="courseSearch.query.value || selectedCourse"
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 bg-transparent border-none cursor-pointer p-0 flex items-center text-text-muted hover:text-text-secondary transition-colors"
                @click="clearCourseSelection"
              >
                <X :size="16" />
              </button>
            </div>

            <!-- Autocomplete dropdown -->
            <div
              v-if="showDropdown && courseSearch.query.value.trim().length > 0"
              class="absolute z-50 left-0 right-0 mt-1 bg-bg-card rounded-lg max-h-[240px] overflow-y-auto shadow-lg"
            >
              <div
                v-if="courseSearchLoading"
                class="px-4 py-3 text-sm text-text-muted"
              >
                {{ t('common.actions.loading') }}
              </div>
              <div
                v-else-if="courseSearchError"
                role="alert"
                class="px-4 py-3 text-sm text-danger"
              >
                {{ courseSearchError }}
              </div>
              <template v-else-if="courseSearch.results.value.length > 0">
                <div
                  v-for="(item, index) in courseSearch.results.value"
                  :key="item.id"
                  class="px-4 py-2.5 text-sm text-text-primary cursor-pointer transition-colors duration-150 hover:bg-bg-hover"
                  :class="{
                    'bg-primary/10': courseSearch.selectedIndex.value === index,
                  }"
                  @click="selectCourse(item)"
                  @mouseenter="courseSearch.selectedIndex.value = index"
                >
                  <div class="font-medium">{{ item.name }}</div>
                  <div class="text-xs text-text-muted mt-0.5">
                    <span v-if="item.departmentName">{{ item.departmentName }}</span>
                    <span v-if="item.departmentName && item.code"> &middot; </span>
                    <span v-if="item.code">{{ item.code }}</span>
                  </div>
                </div>
              </template>
              <div
                v-else
                class="px-4 py-3 text-sm text-text-muted"
              >
                {{ t('review.home.courseNotFound') }}
                <a
                  href="mailto:stuhelper@protonmail.com"
                  class="text-primary underline ml-0.5"
                >{{ t('review.home.contactUs') }}</a>
              </div>
            </div>
          </div>
          <span v-if="showErrors && !selectedCourse" class="block text-danger text-xs mt-1.5">
            {{ t('review.postForm.errors.course') }}
          </span>
          <span v-else-if="courseLoadError" role="alert" class="block text-danger text-xs mt-1.5">
            {{ courseLoadError }}
          </span>
        </div>

        <!-- Teacher selector -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.selectTeacher') }}
            <span class="inline-flex items-center ml-2 px-2 py-0.5 rounded-full text-xs font-bold bg-primary/15 text-primary">
              {{ t('review.post.teacherOptional') }}
            </span>
          </label>
          <div v-if="teachersLoading" class="px-4 py-3 text-sm text-text-muted bg-bg-elevated rounded-lg">
            {{ t('review.post.teacherLoading') }}
          </div>
          <select
            v-else
            v-model="selectedTeacherID"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors disabled:opacity-60"
            :disabled="!selectedCourse"
          >
            <option :value="null">
              {{ selectedCourse ? t('review.post.teacherNone') : t('review.postForm.selectCourse') }}
            </option>
            <option v-for="teacher in teachers" :key="teacher.teacherID" :value="teacher.teacherID">
              {{ teacher.teacherName }}
              <template v-if="teacher.departmentName"> · {{ teacher.departmentName }}</template>
            </option>
          </select>
          <span v-if="teachersLoadError" role="alert" class="block text-danger text-xs mt-1.5">
            {{ teachersLoadError }}
          </span>
        </div>

        <!-- Semester dropdown -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.semester') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <select
            v-model="termID"
            data-testid="review-term"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors appearance-none bg-no-repeat"
            :class="{ 'border-danger focus:border-danger focus:ring-danger/20': showErrors && !termID.trim() }"
            :style="selectArrowStyle"
          >
            <option value="" disabled>{{ t('review.post.termPlaceholder') }}</option>
            <option
              v-for="term in termOptions"
              :key="term.id"
              :value="term.id"
            >
              {{ term.name }}
            </option>
          </select>
          <span v-if="termsLoadError" role="alert" class="block text-danger text-xs mt-1.5">
            {{ termsLoadError }}
          </span>
          <span v-else-if="showErrors && !termID.trim()" class="block text-danger text-xs mt-1.5">
            {{ t('review.postForm.errors.semester') }}
          </span>
        </div>

        <!-- Divider -->
        <hr class="border-border my-6" />

        <!-- Emoji rating section -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.rating') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <p class="text-xs text-text-muted mt-0 mb-3">
            {{ t('review.postForm.ratingTip') }}
          </p>

          <!-- Rating dimension rows -->
          <div class="flex flex-col gap-3">
            <template v-if="ratingDimensionsLoading">
              <div v-for="i in 4" :key="i" class="flex items-center justify-between animate-pulse">
                <div class="h-4 w-24 rounded bg-bg-secondary"></div>
                <div class="h-8 w-36 rounded bg-bg-secondary"></div>
              </div>
            </template>
            <p v-else-if="ratingDimensionsLoadFailed" class="text-sm text-danger">
              {{ t('review.post.ratingLoadFailed') }}
            </p>
            <p v-else-if="ratingDimensions.length === 0" class="text-sm text-text-muted">
              {{ t('review.post.ratingUnavailable') }}
            </p>
            <template v-else>
              <div
                v-for="dim in ratingDimensions"
                :key="dim.key"
                class="flex items-center justify-between"
              >
                <span class="text-sm text-text-secondary shrink-0">
                  {{ dim.name }}
                </span>
                <EmojiRatingInput
                  :model-value="ratings[dim.key] ?? 0"
                  :error="showErrors && !ratings[dim.key] ? t('review.post.ratingMissing') : undefined"
                  :label="dim.name"
                  :test-id-prefix="`rating-${dim.key}`"
                  @update:model-value="(val: number) => updateRating(dim.key, val)"
                />
              </div>
            </template>
          </div>
        </div>

        <!-- Divider -->
        <hr class="border-border my-6" />

        <!-- Title input -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.reviewTitle') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <input
            v-model="title"
            data-testid="review-title"
            type="text"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors"
            :class="{ 'border-danger focus:border-danger focus:ring-danger/20': showErrors && !title.trim() }"
            :placeholder="t('review.postForm.reviewTitlePlaceholder')"
            :maxlength="TITLE_MAX"
          />
          <div class="flex justify-between items-center mt-1.5">
            <span v-if="showErrors && !title.trim()" class="text-danger text-xs">
              {{ t('review.postForm.errors.title') }}
            </span>
            <span v-else />
            <span class="text-xs text-text-muted">
              {{ t('review.validation.charCount', { current: title.length, max: TITLE_MAX }) }}
            </span>
          </div>
        </div>

        <!-- Content textarea -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-1">
            {{ t('review.postForm.detailedReview') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <p class="text-xs text-text-muted mt-0 mb-2">
            {{ t('review.postForm.detailedReviewHint') }}
          </p>
          <textarea
            v-model="content"
            data-testid="review-content"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors resize-y min-h-[200px]"
            :class="{ 'border-danger focus:border-danger focus:ring-danger/20': showErrors && contentError }"
            :placeholder="t('review.postForm.detailedReviewPlaceholder')"
            :maxlength="CONTENT_MAX"
            rows="8"
          />
          <div class="flex justify-between items-center mt-1.5">
            <span
              v-if="showErrors && contentError"
              class="text-danger text-xs"
            >
              {{ contentError }}
            </span>
            <span v-else />
            <span
              class="text-xs"
              :class="content.trim().length > 0 && content.trim().length < CONTENT_MIN
                ? 'text-danger'
                : 'text-text-muted'"
            >
              {{ t('review.validation.charCount', { current: content.length, max: CONTENT_MAX }) }}
            </span>
          </div>
          <p class="text-xs text-text-muted/60 mt-1 mb-0 italic leading-relaxed">
            {{ t('review.postForm.detailedReviewTip') }}
          </p>
        </div>

        <!-- Grade input (optional) -->
        <div class="mb-6">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.grade') }}
            <span class="text-xs font-normal text-text-muted ml-1.5">
              ({{ t('review.post.gradeOptional') }})
            </span>
          </label>
          <select
            v-model="grade"
            data-testid="review-grade"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors appearance-none bg-no-repeat"
            :style="selectArrowStyle"
          >
            <option value="">{{ t('review.postForm.gradePlaceholder') }}</option>
            <option
              v-for="gradeOption in gradeOptions"
              :key="gradeOption"
              :value="gradeOption"
            >
              {{ gradeOption }}
            </option>
          </select>
          <p class="text-xs text-text-muted/60 mt-1.5 mb-0">
            {{ t('review.postForm.gradeHint') }}
          </p>
        </div>

        <!-- Submit button -->
        <div class="flex flex-col items-center gap-3">
          <p
            v-if="submitError"
            role="alert"
            class="w-full m-0 px-4 py-3 rounded-lg bg-danger/10 text-danger text-sm leading-relaxed"
          >
            {{ submitError }}
          </p>
          <button
            data-testid="review-submit"
            class="w-full bg-primary hover:bg-primary-dark text-white rounded-xl py-3 font-semibold text-base cursor-pointer transition-all duration-200 border-none hover:-translate-y-px hover:shadow-lg disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0 disabled:hover:shadow-none"
            :disabled="submitting"
            @click="handleSubmit"
          >
            {{ submitting ? t('review.postForm.submitting') : t('review.postForm.submitBtn') }}
          </button>
          <p class="text-xs text-text-muted text-center m-0 max-w-[400px]">
            {{ t('review.postForm.submitDisclaimer') }}
          </p>
        </div>
      </div>
    </div>

    <DraftPromptDialog
      visible
      v-if="restorePromptDraft"
      title-id="review-draft-restore-title"
      :title="t('review.draft.restoreTitle')"
      :description="t('review.draft.restoreDescription')"
      :confirm-text="t('review.draft.restoreConfirm')"
      :danger-text="t('review.draft.discard')"
      @confirm="restorePromptDraft && applyDraft(restorePromptDraft)"
      @discard="discardRestorePromptDraft"
    />

    <DraftPromptDialog
      :visible="leavePromptVisible"
      title-id="review-draft-leave-title"
      :title="t('review.draft.leaveTitle')"
      :description="t('review.draft.confirmSave')"
      :confirm-text="t('review.draft.keepDraft')"
      :danger-text="t('review.draft.discard')"
      @confirm="resolveLeavePrompt(true)"
      @keep="resolveLeavePrompt(true)"
      @discard="resolveLeavePrompt(false)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave, useRouter } from 'vue-router'
import { ArrowLeft, Search, X } from 'lucide-vue-next'
import type { Draft, SaveDraftParams } from '@stuhelper/shared/draft'
import DraftIndicator from '@/components/business/review/DraftIndicator.vue'
import DraftPromptDialog from '@/components/business/review/DraftPromptDialog.vue'
import EmojiRatingInput from '@/components/business/review/EmojiRatingInput.vue'
import { areRatingsComplete, useRatingDimensions } from '@/components/business/review/composables/useRatingDimensions'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import { usePinyinSearch, type PinyinSearchItem } from '@/composables/usePinyinSearch'
import { buildCreateReviewPayload } from '@/components/business/review/reviewPayload'
import { buildTermOptions } from '@/modules/course/termOptions'
import {
  readCourseListPayload,
  readCoursePayload,
  readTeacherStatsArrayPayload,
  readTermArrayPayload,
} from '@/modules/course/coursePayload'
import { useReviewPost } from '@/composables/useReviewPost'
import { getErrorMessage } from '@/api/errors'
import { consumeReviewPostCourseID } from '@/modules/review/reviewPostNavigation'
import { useDraftStore } from '@/stores/draft'
import {
  REVIEW_TITLE_MAX_LENGTH,
  REVIEW_CONTENT_MIN_LENGTH,
  REVIEW_CONTENT_MAX_LENGTH,
  REVIEW_GRADES,
  normalizeReviewGrade,
} from '@stuhelper/shared/constants'
import type { Course, TeacherStats, Term } from '@stuhelper/shared/course'
import type { ReviewRatings } from '@stuhelper/shared/review'

const router = useRouter()
const { t } = useI18n()
const toast = useToast()
const { ensureCanPostReview } = useReviewPost()
const draftStore = useDraftStore()

// ── Constants ────────────────────────────────────────────
const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH
const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH
const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH
const gradeOptions = REVIEW_GRADES

const defaultTemplate = computed(() => t('review.postForm.defaultTemplate'))

// ── Select arrow (theme-aware via currentColor) ──────────
const selectArrowStyle = computed(() => ({
  backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='currentColor' viewBox='0 0 16 16'%3E%3Cpath d='M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z'/%3E%3C/svg%3E")`,
  backgroundRepeat: 'no-repeat',
  backgroundPosition: 'right 12px center',
  paddingRight: '36px',
}))

const {
  dimensions: ratingDimensions,
  loading: ratingDimensionsLoading,
  loadFailed: ratingDimensionsLoadFailed,
} = useRatingDimensions()

// ── Form state ───────────────────────────────────────────
const selectedCourse = ref<Course | null>(null)
const teachers = ref<TeacherStats[]>([])
const teachersLoading = ref(false)
const teachersLoadError = ref('')
const selectedTeacherID = ref<number | null>(null)
const courseLoadError = ref('')
const termID = ref('')
const ratings = ref<ReviewRatings>({})
const title = ref('')
const content = ref(defaultTemplate.value)
const grade = ref('')
const submitting = ref(false)
const showErrors = ref(false)
const submitError = ref('')
const restorePromptDraft = ref<Draft | null>(null)
const leavePromptVisible = ref(false)
const suppressAutosave = ref(true)
const submittingSuccessfully = ref(false)
const draftTeacherIDToRestore = ref<number | null>(null)
const discardedDraftSignature = ref<string | null>(null)
const restorePromptDiscarded = ref(false)
let autosaveTimer: ReturnType<typeof setTimeout> | null = null
let leavePromptResolver: ((keepDraft: boolean) => void) | null = null
const AUTOSAVE_DELAY_MS = 700

// ── Term options ─────────────────────────────────────────
const terms = ref<Term[]>([])
const termsLoadError = ref('')
const termOptions = computed(() => buildTermOptions(terms.value))

async function fetchTerms() {
  termsLoadError.value = ''
  try {
    const res = await api.course.getTerms()
    terms.value = readTermArrayPayload(
      res.data?.data,
      'Invalid terms response',
    )
    if (!termID.value && terms.value.length > 0) {
      const options = buildTermOptions(terms.value)
      termID.value = options[0]?.id || ''
    }
  } catch (_error) { void _error;
    terms.value = []
    termID.value = ''
    termsLoadError.value = t('common.loadFailed')
  }
}

async function fetchTeachers(courseID: number) {
  teachersLoading.value = true
  teachersLoadError.value = ''
  teachers.value = []
  selectedTeacherID.value = null
  try {
    const res = await api.rating.getCourseTeachers(courseID)
    teachers.value = readTeacherStatsArrayPayload(
      res.data?.data,
      'Invalid course teachers response',
    )
  } catch (_error) { void _error;
    teachers.value = []
    teachersLoadError.value = t('common.loadFailed')
  } finally {
    teachersLoading.value = false
  }
}

// ── Course search ────────────────────────────────────────
const courses = ref<Course[]>([])
const courseSearchLoading = ref(false)
const courseSearchError = ref('')
const showDropdown = ref(false)
const searchContainerRef = ref<HTMLDivElement | null>(null)
const courseSearchInputRef = ref<HTMLInputElement | null>(null)

const courseItems = computed<PinyinSearchItem[]>(() =>
  courses.value.map((c) => ({
    ...c,
    id: c.id,
    name: c.name,
  })),
)

const courseSearch = usePinyinSearch({
  items: courseItems,
  maxResults: 15,
})

// 搜索词变化后延迟请求接口，避免频繁查询
let searchAbortController: AbortController | null = null
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => courseSearch.query.value,
  (q) => {
    const trimmed = q.trim()
    if (!trimmed) {
      courses.value = []
      courseSearchError.value = ''
      return
    }

    courseLoadError.value = ''
    if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
    if (searchAbortController) searchAbortController.abort()

    searchDebounceTimer = setTimeout(async () => {
      courseSearchLoading.value = true
      courseSearchError.value = ''
      searchAbortController = new AbortController()
      try {
        const res = await api.course.searchCourses(trimmed, { pageSize: 30 }, { signal: searchAbortController.signal })
        courses.value = readCourseListPayload(
          res.data?.data,
          'Invalid course search response',
        )
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === 'AbortError') return
        courses.value = []
        courseSearchError.value = t('common.loadFailed')
      } finally {
        courseSearchLoading.value = false
      }
    }, 300)
  },
)

function selectCourse(item: PinyinSearchItem) {
  const found = courses.value.find((c) => c.id === item.id)
  if (found) {
    restorePromptDiscarded.value = false
    courseLoadError.value = ''
    selectedTeacherID.value = null
    teachers.value = []
    teachersLoadError.value = ''
    selectedCourse.value = found
    courseSearch.query.value = found.name
    courseSearchError.value = ''
    showDropdown.value = false
  }
}

function clearCourseSelection() {
  restorePromptDiscarded.value = false
  courseLoadError.value = ''
  selectedCourse.value = null
  selectedTeacherID.value = null
  teachers.value = []
  teachersLoadError.value = ''
  courseSearch.query.value = ''
  courseSearch.clear()
  courses.value = []
  courseSearchError.value = ''
  showDropdown.value = false
}

async function editCourseSelection() {
  clearCourseSelection()
  showDropdown.value = true
  await nextTick()
  courseSearchInputRef.value?.focus()
}

function handleCourseKeyDown(e: KeyboardEvent) {
  if (!showDropdown.value) return
  const selected = courseSearch.handleKeyDown(e)
  if (selected) {
    selectCourse(selected)
  }
}

// 点击外部区域时关闭下拉面板
function handleClickOutside(e: MouseEvent) {
  if (searchContainerRef.value && !searchContainerRef.value.contains(e.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside, true)
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  if (searchAbortController) searchAbortController.abort()
  if (autosaveTimer) clearTimeout(autosaveTimer)
  window.removeEventListener('beforeunload', handleBeforeUnload)
})

async function loadCourseSelection(courseID: number): Promise<boolean> {
  courseLoadError.value = ''
  try {
    const res = await api.course.getCourse(courseID)
    const data = readCoursePayload(res.data?.data, 'Invalid course response')
    selectedCourse.value = data
    courseSearch.query.value = data.name
    return true
  } catch (_error) { void _error;
    selectedCourse.value = null
    selectedTeacherID.value = null
    teachers.value = []
    teachersLoadError.value = ''
    courseSearch.query.value = ''
    courseSearch.clear()
    courses.value = []
    courseLoadError.value = t('common.loadFailed')
    return false
  }
}

// ── 从入口携带的临时状态恢复默认课程 ─────────
onMounted(async () => {
  if (!(await ensureCanPostReview())) {
    return
  }

  await fetchTerms()

  const courseID = consumeReviewPostCourseID()
  if (courseID) {
    await loadCourseSelection(courseID)
  }

  await promptDraftRestoreOnEntry()
  suppressAutosave.value = false
})

watch(selectedCourse, async (course) => {
  if (!course) {
    selectedTeacherID.value = null
    teachers.value = []
    return
  }

  await fetchTeachers(course.id)
  restoreDraftTeacherSelection(course)
})

function buildDraftPayload() {
  const nextGrade = normalizeReviewGrade(grade.value)
  return {
    ...(selectedCourse.value?.id ? { courseID: selectedCourse.value.id } : {}),
    ...(selectedTeacherID.value ? { teacherID: selectedTeacherID.value } : {}),
    ...(termID.value.trim() ? { termID: termID.value.trim() } : {}),
    ...(title.value ? { title: title.value } : {}),
    ...(content.value ? { content: content.value } : {}),
    ...(nextGrade ? { grade: nextGrade } : {}),
    ...(Object.keys(ratings.value).length > 0 ? { ratings: ratings.value } : {}),
  }
}

function currentDraftSignature() {
  const payload: SaveDraftParams = buildDraftPayload()
  const ratings = Object.entries(payload.ratings ?? {})
    .sort(([left], [right]) => left.localeCompare(right))

  return JSON.stringify({
    content: payload.content ?? '',
    courseID: payload.courseID ?? null,
    grade: payload.grade ?? '',
    ratings,
    teacherID: payload.teacherID ?? null,
    termID: payload.termID ?? '',
    title: payload.title ?? '',
  })
}

const hasMeaningfulDraftInput = computed(() => {
  return Boolean(
    selectedCourse.value ||
    selectedTeacherID.value ||
    title.value.trim() ||
    normalizeReviewGrade(grade.value) ||
    Object.keys(ratings.value).length > 0 ||
    (content.value.trim() && content.value.trim() !== defaultTemplate.value.trim()),
  )
})

async function autosaveDraftNow() {
  try {
    if (currentDraftSignature() === discardedDraftSignature.value) return
    if (restorePromptDiscarded.value && !hasMeaningfulDraftInput.value) return
    if (!hasMeaningfulDraftInput.value) {
      if (draftStore.hasDraft) {
        await draftStore.deleteDraft()
      }
      return
    }
    await draftStore.saveDraft(buildDraftPayload())
  } catch (_error) { void _error;
    toast.error(t('review.draft.saveFailed'))
  }
}

function scheduleAutosave() {
  if (suppressAutosave.value || submittingSuccessfully.value) return
  if (currentDraftSignature() === discardedDraftSignature.value) return
  if (restorePromptDiscarded.value && !hasMeaningfulDraftInput.value) return
  if (autosaveTimer) clearTimeout(autosaveTimer)
  autosaveTimer = setTimeout(() => {
    void autosaveDraftNow()
  }, AUTOSAVE_DELAY_MS)
}

async function applyDraft(draft: Draft) {
  suppressAutosave.value = true
  discardedDraftSignature.value = null
  restorePromptDiscarded.value = false
  title.value = draft.title ?? ''
  content.value = draft.content ?? defaultTemplate.value
  grade.value = draft.grade ?? ''
  termID.value = draft.termID ?? termID.value
  ratings.value = (draft.ratings ?? {}) as ReviewRatings
  draftTeacherIDToRestore.value = draft.teacherID ?? null
  restorePromptDraft.value = null
  toast.success(t('review.draft.restored'))

  if (draft.courseID) {
    await loadDraftCourse(draft.courseID)
  } else {
    clearCourseSelection()
  }
  if (!draft.courseID) {
    selectedTeacherID.value = draftTeacherIDToRestore.value
    draftTeacherIDToRestore.value = null
  }

  nextTick(() => {
    suppressAutosave.value = false
  })
}

function restoreDraftTeacherSelection(course: Course) {
  const teacherID = draftTeacherIDToRestore.value
  if (teacherID === null) return
  if (selectedCourse.value?.id !== course.id) return
  selectedTeacherID.value = teacherID
  draftTeacherIDToRestore.value = null
}

async function loadDraftCourse(courseID: number) {
  const loaded = await loadCourseSelection(courseID)
  if (!loaded) {
    draftTeacherIDToRestore.value = null
  }
}

async function promptDraftRestoreOnEntry() {
  try {
    const draft = await draftStore.loadDraft(true)
    if (draft) {
      restorePromptDraft.value = draft
    }
  } catch (_error) { void _error;
    toast.error(t('review.draft.saveFailed'))
  }
}

async function promptRestoreExistingDraft() {
  try {
    const draft = await draftStore.loadDraft(true)
    if (draft) {
      restorePromptDraft.value = draft
    }
  } catch (_error) { void _error;
    toast.error(t('review.draft.saveFailed'))
  }
}

async function discardCurrentDraft() {
  try {
    if (autosaveTimer) clearTimeout(autosaveTimer)
    await draftStore.deleteDraft()
    discardedDraftSignature.value = currentDraftSignature()
    restorePromptDraft.value = null
  } catch (_error) { void _error;
    toast.error(t('review.draft.saveFailed'))
  }
}

async function discardRestorePromptDraft() {
  try {
    if (autosaveTimer) clearTimeout(autosaveTimer)
    await draftStore.deleteDraft()
    restorePromptDiscarded.value = true
    discardedDraftSignature.value = currentDraftSignature()
    restorePromptDraft.value = null
  } catch (_error) { void _error;
    toast.error(t('review.draft.saveFailed'))
  }
}

watch(
  [
    selectedCourse,
    selectedTeacherID,
    termID,
    ratings,
    title,
    content,
    grade,
  ],
  scheduleAutosave,
  { deep: true },
)

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!draftStore.hasDraft && !hasMeaningfulDraftInput.value) return
  event.preventDefault()
  event.returnValue = ''
}

window.addEventListener('beforeunload', handleBeforeUnload)

function waitForLeaveDecision() {
  leavePromptVisible.value = true
  return new Promise<boolean>((resolve) => {
    leavePromptResolver = resolve
  })
}

function resolveLeavePrompt(keepDraft: boolean) {
  leavePromptVisible.value = false
  leavePromptResolver?.(keepDraft)
  leavePromptResolver = null
}

async function confirmLeaveWithDraft() {
  if (submittingSuccessfully.value) return true
  if (!draftStore.hasDraft && !hasMeaningfulDraftInput.value) return true
  await autosaveDraftNow()
  if (!draftStore.hasDraft && !hasMeaningfulDraftInput.value) return true
  const keepDraft = await waitForLeaveDecision()
  if (keepDraft) return true
  await discardCurrentDraft()
  return true
}

onBeforeRouteLeave(async () => {
  return confirmLeaveWithDraft()
})

// ── 评分相关状态 ─────────────────────────────────────────
function updateRating(key: string, value: number) {
  ratings.value = { ...ratings.value, [key]: value } as ReviewRatings
}

// ── 表单校验 ─────────────────────────────────────────────
const contentError = computed(() => {
  const trimmed = content.value.trim()
  if (!trimmed || trimmed === defaultTemplate.value.trim()) {
    return t('review.postForm.errors.review')
  }
  if (trimmed.length < CONTENT_MIN) {
    return t('review.validation.contentTooShort', { min: CONTENT_MIN })
  }
  return ''
})

const allRatingsProvided = computed(() =>
  !ratingDimensionsLoading.value &&
  !ratingDimensionsLoadFailed.value &&
  areRatingsComplete(ratings.value, ratingDimensions.value),
)

const canSubmit = computed(() =>
  !!selectedCourse.value &&
  !termsLoadError.value &&
  termID.value.trim().length > 0 &&
  allRatingsProvided.value &&
  title.value.trim().length > 0 &&
  title.value.length <= TITLE_MAX &&
  !contentError.value,
)

// ── 提交流程 ─────────────────────────────────────────────
async function handleSubmit() {
  showErrors.value = true
  submitError.value = ''
  if (ratingDimensionsLoadFailed.value) {
    submitError.value = t('review.post.ratingLoadFailed')
    toast.error(submitError.value)
    return
  }
  if (termsLoadError.value) {
    submitError.value = t('common.loadFailed')
    toast.error(submitError.value)
    return
  }

  if (!canSubmit.value) return

  submitting.value = true
  try {
    const nextGrade = normalizeReviewGrade(grade.value)
    // 内容预检
    let checkResult: Awaited<ReturnType<typeof api.review.checkContentResult>>
    try {
      checkResult = await api.review.checkContentResult({ content: content.value.trim() })
    } catch (_error) { void _error;
      submitError.value = t('common.loadFailed')
      toast.error(submitError.value)
      return
    }
    if (!checkResult.isValid) {
      if (checkResult.level === 'warn') {
        toast.warning(t('review.post.contentWarning'))
      } else {
        submitError.value = t('review.post.contentBlocked')
        toast.error(submitError.value)
        return
      }
    }

    const payload = buildCreateReviewPayload({
      courseID: selectedCourse.value!.id,
      teacherID: selectedTeacherID.value ?? undefined,
      termID: termID.value,
      title: title.value.trim(),
      content: content.value.trim(),
      ratings: ratings.value,
      ...(nextGrade ? { grade: nextGrade } : {}),
    })

    await api.review.createReview(payload)
    submittingSuccessfully.value = true
    if (draftStore.hasDraft) {
      await draftStore.deleteDraft()
    }
    toast.success(t('review.post.success'))

    // 发布成功后跳转到课程评测页
    router.push({ name: 'course-reviews', params: { id: selectedCourse.value!.id } })
  } catch (error) {
    submitError.value = getErrorMessage(error, t('review.post.failed'))
    toast.error(submitError.value)
  } finally {
    submitting.value = false
  }
}
</script>
