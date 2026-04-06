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
        <!-- Course search -->
        <div class="mb-5">
          <label class="block text-sm font-medium text-text-secondary mb-2">
            {{ t('review.postForm.selectCourse') }}
            <span class="text-danger ml-0.5">*</span>
          </label>
          <div class="relative" ref="searchContainerRef">
            <div class="relative">
              <Search
                :size="18"
                class="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none text-text-muted"
              />
              <input
                ref="searchInputRef"
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
                class="absolute right-3 top-1/2 -translate-y-1/2 bg-transparent border-none cursor-pointer p-0 flex items-center text-text-muted hover:text-text-secondary transition-colors"
                @click="clearCourseSelection"
              >
                <X :size="16" />
              </button>
            </div>

            <!-- Selected course display -->
            <div
              v-if="selectedCourse && !showDropdown"
              class="mt-2 bg-primary/10 text-primary rounded-lg px-3 py-2 text-sm flex items-center justify-between"
            >
              <span>{{ selectedCourse.name }}</span>
              <button
                class="bg-transparent border-none cursor-pointer p-0 flex items-center text-primary/60 hover:text-primary transition-colors"
                @click="clearCourseSelection"
              >
                <X :size="14" />
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
          <span v-if="showErrors && !termID.trim()" class="block text-danger text-xs mt-1.5">
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
            <div
              v-for="dim in ratingDimensions"
              :key="dim.key"
              class="flex items-center justify-between"
            >
              <span class="text-sm text-text-secondary shrink-0">
                {{ dim.label }}
              </span>
              <EmojiRatingInput
                :model-value="ratings[dim.key] ?? 0"
                :error="showErrors && !ratings[dim.key] ? dim.errorMsg : undefined"
                :test-id-prefix="`rating-${dim.key}`"
                @update:model-value="(val: number) => updateRating(dim.key, val)"
              />
            </div>
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
          <input
            v-model="grade"
            type="text"
            class="w-full px-4 py-3 bg-bg-elevated rounded-lg text-text-primary placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 outline-none transition-colors"
            :placeholder="t('review.postForm.gradePlaceholder')"
          />
          <p class="text-xs text-text-muted/60 mt-1.5 mb-0">
            {{ t('review.postForm.gradeHint') }}
          </p>
        </div>

        <!-- Submit button -->
        <div class="flex flex-col items-center gap-3">
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Search, X } from 'lucide-vue-next'
import EmojiRatingInput from '@/components/business/review/EmojiRatingInput.vue'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import { usePinyinSearch, type PinyinSearchItem } from '@/composables/usePinyinSearch'
import { buildCreateReviewPayload } from '@/components/business/review/reviewPayload'
import { buildTermOptions } from '@/modules/course/termOptions'
import {
  REVIEW_TITLE_MAX_LENGTH,
  REVIEW_CONTENT_MIN_LENGTH,
  REVIEW_CONTENT_MAX_LENGTH,
} from '@/constants/review'
import type { Course, TeacherStats, Term } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const toast = useToast()

// ── Constants ────────────────────────────────────────────
const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH
const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH
const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH

const defaultTemplate = computed(() => t('review.postForm.defaultTemplate'))

// ── Select arrow (theme-aware via currentColor) ──────────
const selectArrowStyle = computed(() => ({
  backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='currentColor' viewBox='0 0 16 16'%3E%3Cpath d='M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z'/%3E%3C/svg%3E")`,
  backgroundRepeat: 'no-repeat',
  backgroundPosition: 'right 12px center',
  paddingRight: '36px',
}))

// ── Rating dimensions (fixed 4, matching old React design) ──
const ratingDimensions = computed(() => [
  {
    key: 'recommendation',
    label: t('review.postForm.overall'),
    errorMsg: t('review.postForm.errors.overall'),
  },
  {
    key: 'content_quality',
    label: t('review.postForm.contentQuality'),
    errorMsg: t('review.postForm.errors.content'),
  },
  {
    key: 'workload',
    label: t('review.postForm.workload'),
    errorMsg: t('review.postForm.errors.workload'),
  },
  {
    key: 'grading',
    label: t('review.postForm.exam'),
    errorMsg: t('review.postForm.errors.exam'),
  },
])

// ── Form state ───────────────────────────────────────────
const selectedCourse = ref<Course | null>(null)
const teachers = ref<TeacherStats[]>([])
const teachersLoading = ref(false)
const selectedTeacherID = ref<number | null>(null)
const termID = ref('')
const ratings = ref<ReviewRatings>({})
const title = ref('')
const content = ref(defaultTemplate.value)
const grade = ref('')
const submitting = ref(false)
const showErrors = ref(false)

// ── Term options ─────────────────────────────────────────
const terms = ref<Term[]>([])
const termOptions = computed(() => buildTermOptions(terms.value))

async function fetchTerms() {
  try {
    const res = await api.course.getTerms()
    terms.value = res.data?.data || []
    if (!termID.value && terms.value.length > 0) {
      const options = buildTermOptions(terms.value)
      termID.value = options[0]?.id || ''
    }
  } catch {
    terms.value = []
  }
}

async function fetchTeachers(courseID: number) {
  teachersLoading.value = true
  teachers.value = []
  selectedTeacherID.value = null
  try {
    const res = await api.rating.getCourseTeachers(courseID)
    teachers.value = res.data?.data ?? []
  } catch {
    teachers.value = []
  } finally {
    teachersLoading.value = false
  }
}

// ── Course search ────────────────────────────────────────
const courses = ref<Course[]>([])
const courseSearchLoading = ref(false)
const showDropdown = ref(false)
const searchContainerRef = ref<HTMLDivElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)

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

// Debounced API search when query changes
let searchAbortController: AbortController | null = null
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => courseSearch.query.value,
  (q) => {
    const trimmed = q.trim()
    if (!trimmed) {
      courses.value = []
      return
    }

    if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
    if (searchAbortController) searchAbortController.abort()

    searchDebounceTimer = setTimeout(async () => {
      courseSearchLoading.value = true
      searchAbortController = new AbortController()
      try {
        const res = await api.course.searchCourses(trimmed, { pageSize: 30 }, { signal: searchAbortController.signal })
        const data = res.data?.data
        courses.value = Array.isArray(data)
          ? data
          : Array.isArray(data?.list)
            ? data.list
            : []
      } catch (err: unknown) {
        if (err instanceof DOMException && err.name === 'AbortError') return
        courses.value = []
      } finally {
        courseSearchLoading.value = false
      }
    }, 300)
  },
)

function selectCourse(item: PinyinSearchItem) {
  const found = courses.value.find((c) => c.id === item.id)
  if (found) {
    selectedTeacherID.value = null
    teachers.value = []
    selectedCourse.value = found
    courseSearch.query.value = found.name
    showDropdown.value = false
  }
}

function clearCourseSelection() {
  selectedCourse.value = null
  selectedTeacherID.value = null
  teachers.value = []
  courseSearch.query.value = ''
  courseSearch.clear()
  courses.value = []
  showDropdown.value = false
}

function handleCourseKeyDown(e: KeyboardEvent) {
  if (!showDropdown.value) return
  const selected = courseSearch.handleKeyDown(e)
  if (selected) {
    selectCourse(selected)
  }
}

// Close dropdown on click outside
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
})

// ── Pre-select course if coming from route param ─────────
onMounted(async () => {
  await fetchTerms()

  const courseID = Number(route.params.id)
  if (!Number.isNaN(courseID) && courseID > 0) {
    try {
      const res = await api.course.getCourse(courseID)
      const data = res.data?.data as Course | undefined
      if (data) {
        selectedCourse.value = data
        courseSearch.query.value = data.name
      }
    } catch {
      // Course not found via route param, user can search manually
    }
  }
})

watch(selectedCourse, async (course) => {
  if (!course) {
    selectedTeacherID.value = null
    teachers.value = []
    return
  }

  await fetchTeachers(course.id)
})

// ── Ratings ──────────────────────────────────────────────
function updateRating(key: string, value: number) {
  ratings.value = { ...ratings.value, [key]: value } as ReviewRatings
}

// ── Validation ───────────────────────────────────────────
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
  ratingDimensions.value.every((dim) => {
    const v = ratings.value[dim.key]
    return v !== undefined && v >= 1 && v <= 5
  }),
)

const canSubmit = computed(() =>
  !!selectedCourse.value &&
  termID.value.trim().length > 0 &&
  allRatingsProvided.value &&
  title.value.trim().length > 0 &&
  title.value.length <= TITLE_MAX &&
  !contentError.value,
)

// ── Submission ───────────────────────────────────────────
async function handleSubmit() {
  showErrors.value = true

  if (!canSubmit.value) return

  submitting.value = true
  try {
    // 内容预检
    const checkRes = await api.review.checkContent({ content: content.value.trim() })
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

    const payload = buildCreateReviewPayload({
      courseID: selectedCourse.value!.id,
      teacherID: selectedTeacherID.value ?? undefined,
      termID: termID.value,
      title: title.value.trim(),
      content: content.value.trim(),
      ratings: ratings.value,
      ...(grade.value.trim() ? { grade: grade.value.trim() } : {}),
    })

    await api.review.createReview(payload)
    toast.success(t('review.post.success'))

    // Navigate to the course reviews page
    router.push({ name: 'course-reviews', params: { id: selectedCourse.value!.id } })
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
