import { ref, computed, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  REVIEW_TITLE_MAX_LENGTH,
  REVIEW_CONTENT_MIN_LENGTH,
  REVIEW_CONTENT_MAX_LENGTH,
} from '@/constants/review'
import { api } from '@/api'
import { buildTermOptions } from '@/modules/course/termOptions'
import type { Course, RatingDimension, Term } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

// Hardcoded fallback rating dimensions when API fails
const DEFAULT_RATING_DIMENSIONS = ['recommendation', 'content_quality', 'workload', 'grading']

const VALID_RATING_KEYS = new Set(DEFAULT_RATING_DIMENSIONS)

const GRADE_PATTERN = /^[A-Za-z0-9+\-./\s]*$/

export function useReviewDialogForm() {
  const { t } = useI18n()

  const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH
  const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH
  const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH

  const selectedCourse = ref<Course | null>(null)
  const title = ref('')
  const content = ref('')
  const grade = ref('')
  const ratings = ref<ReviewRatings>({})
  const attempted = ref(false)

  /** Externally set by the parent after the form sub-component mounts */
  const ratingDimensions: Ref<RatingDimension[]> = ref([])

  const terms = ref<Term[]>([])
  const selectedTermID = ref('')
  const termOptions = computed(() => buildTermOptions(terms.value))

  const templateLabels = computed(() => [
    t('review.post.templateListening'),
    t('review.post.templateWorkload'),
    t('review.post.templateExam'),
  ])
  const contentTemplate = computed(() => templateLabels.value.join('\n'))

  // Track whether the user has manually edited content to avoid
  // overwriting user input when the locale changes.
  let userHasEditedContent = false

  watch(contentTemplate, (newTpl, oldTpl) => {
    if (!userHasEditedContent && content.value === oldTpl) {
      content.value = newTpl
    }
  })

  /** Actual user-typed length, excluding template labels */
  function getUserContentLength(raw: string): number {
    let text = raw
    for (const label of templateLabels.value) {
      text = text.split(label).join('')
    }
    return text.trim().length
  }

  const titleInvalid = computed(() => title.value.trim().length === 0)

  const ratingsInvalid = computed(() => {
    const dims = ratingDimensions.value
    const keys = dims.length > 0 ? dims.map(d => d.key) : DEFAULT_RATING_DIMENSIONS
    return keys.some(k => {
      const v = ratings.value[k]
      return !v || v < 1 || v > 5
    })
  })

  const contentInvalid = computed(() =>
    getUserContentLength(content.value) < CONTENT_MIN,
  )

  const contentError = computed(() => {
    const userLen = getUserContentLength(content.value)
    if (userLen > 0 && userLen < CONTENT_MIN) {
      return t('review.post.contentMinErrorNoTemplate', { min: CONTENT_MIN })
    }
    return ''
  })

  const gradeInvalid = computed(() => {
    const g = grade.value.trim()
    return g.length > 0 && !GRADE_PATTERN.test(g)
  })

  const canSubmit = computed(() => {
    return selectedCourse.value &&
      !titleInvalid.value &&
      !contentInvalid.value &&
      !gradeInvalid.value &&
      content.value.length <= CONTENT_MAX &&
      title.value.length <= TITLE_MAX &&
      !ratingsInvalid.value
  })

  /** Whether the form has any meaningful user input (excluding template text) */
  function hasFormContent(): boolean {
    return title.value.trim().length > 0 ||
      getUserContentLength(content.value) > 0 ||
      grade.value.trim().length > 0 ||
      Object.keys(ratings.value).length > 0
  }

  function markUserEdited() {
    userHasEditedContent = true
  }

  async function loadTerms() {
    try {
      const res = await api.course.getTerms()
      terms.value = res.data?.data ?? []
      if (!selectedTermID.value && terms.value.length > 0) {
        selectedTermID.value = buildTermOptions(terms.value)[0]?.id || ''
      }
    } catch {
      terms.value = []
      selectedTermID.value = ''
    }
  }

  function resetForm(template: string) {
    title.value = ''
    content.value = template
    grade.value = ''
    ratings.value = {}
    selectedTermID.value = ''
    attempted.value = false
    userHasEditedContent = false
  }

  return {
    TITLE_MAX,
    CONTENT_MIN,
    CONTENT_MAX,
    VALID_RATING_KEYS,
    selectedCourse,
    title,
    content,
    grade,
    ratings,
    attempted,
    ratingDimensions,
    terms,
    selectedTermID,
    termOptions,
    templateLabels,
    contentTemplate,
    titleInvalid,
    ratingsInvalid,
    contentInvalid,
    contentError,
    gradeInvalid,
    canSubmit,
    getUserContentLength,
    hasFormContent,
    markUserEdited,
    loadTerms,
    resetForm,
  }
}
