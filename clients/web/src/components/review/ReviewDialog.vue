<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="visible" class="fixed inset-0 bg-bg-overlay z-50 flex items-center justify-center p-4" @click.self="$emit('close')">
        <div
          ref="modalRef"
          class="w-full max-w-[660px] max-h-[85vh] bg-bg-card border border-border rounded-xl shadow-xl flex flex-col overflow-hidden animate-modal-in"
          role="dialog"
          aria-modal="true"
          aria-labelledby="review-dialog-title"
          tabindex="-1"
          @keydown.esc="$emit('close')"
          @keydown="trapFocus"
        >
          <div class="flex items-center justify-between p-5 border-b border-border">
            <h2 id="review-dialog-title" class="text-lg font-bold tracking-tight">{{ t('review.hub.postReview') }}</h2>
            <button class="text-2xl text-text-muted leading-none cursor-pointer w-8 h-8 flex items-center justify-center rounded-full transition-all duration-fast hover:text-text-primary hover:bg-bg-secondary" :aria-label="t('common.actions.close')" @click="$emit('close')">&times;</button>
          </div>

          <div class="flex-1 overflow-y-auto p-5 flex flex-col gap-4">
            <!-- 课程搜索 -->
            <div v-if="!selectedCourse">
              <input
                v-model="courseQuery"
                class="w-full p-3 bg-bg-secondary border border-border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                :placeholder="t('review.post.searchCourse')"
                :aria-label="t('review.post.searchCourseLabel')"
              />
              <div v-if="courseResults.length > 0" class="border border-border rounded-lg max-h-[200px] overflow-y-auto mt-2">
                <button
                  v-for="c in courseResults"
                  :key="c.id"
                  class="flex items-center gap-2 w-full p-3 text-left text-sm text-text-primary cursor-pointer transition-[background] duration-fast hover:bg-bg-hover"
                  @click="selectCourse(c)"
                >
                  <span class="font-medium truncate">{{ c.name }}</span>
                  <span class="shrink-0 text-xs text-text-muted"><template v-if="c.credits">{{ t('review.course.creditsBadge', { n: c.credits }) }} · </template>{{ c.departmentName }}</span>
                  <span class="shrink-0 text-xs tabular-nums text-text-muted ml-auto">{{ t('review.course.reviewCountBadge', { count: c.reviewCount ?? 0 }) }}</span>
                </button>
              </div>
            </div>

            <!-- 已选课程 -->
            <div v-else class="flex items-center justify-between p-3 bg-primary/[0.06] rounded-lg border border-primary/15">
              <span class="font-semibold text-sm">{{ selectedCourse.name }}</span>
              <button class="text-xs text-primary cursor-pointer" @click="selectedCourse = null">
                {{ t('common.actions.edit') }}
              </button>
            </div>

            <!-- 表单 -->
            <template v-if="selectedCourse">
              <div class="relative">
                <span class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.titleRequired') }} <span class="text-danger text-xs">*</span></span>
                <input
                  v-model="title"
                  class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :class="attempted && titleInvalid ? 'border-danger' : 'border-border'"
                  :placeholder="t('review.post.titlePlaceholder')"
                  :aria-label="t('review.post.titleLabel')"
                  :maxlength="TITLE_MAX"
                />
                <span v-if="attempted && titleInvalid" class="block text-xs text-danger mt-1">{{ t('review.post.titleMissing') }}</span>
                <span v-else class="block text-right text-xs text-text-muted mt-1">
                  {{ t('review.validation.charCount', { current: title.length, max: TITLE_MAX }) }}
                </span>
              </div>

              <div :class="{ 'ring-1 ring-danger rounded-lg': attempted && ratingsInvalid }">
                <RatingGroup ref="ratingGroupRef" v-model="ratings" />
              </div>
              <span v-if="attempted && ratingsInvalid" class="block text-xs text-danger -mt-2">{{ t('review.post.ratingMissing') }}</span>

              <div class="relative">
                <span class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.detailedReview') }} <span class="text-danger text-xs">*</span></span>
                <textarea
                  v-model="content"
                  class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[120px] transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :class="attempted && contentInvalid ? 'border-danger' : 'border-border'"
                  :placeholder="t('review.post.contentPlaceholder')"
                  :aria-label="t('review.post.contentLabel')"
                  :aria-describedby="contentError ? 'review-dialog-content-error' : undefined"
                  :maxlength="CONTENT_MAX"
                  rows="5"
                />
                <span v-if="contentError || (attempted && contentInvalid)" id="review-dialog-content-error" class="block text-xs text-danger mt-1">
                  {{ contentError || t('review.post.contentMinError', { min: CONTENT_MIN }) }}
                </span>
              </div>

              <div class="relative">
                <span class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.gradeLabel') }} <span class="text-text-muted font-normal text-xs">（{{ t('review.post.gradeOptional') }}）</span></span>
                <input
                  v-model="grade"
                  class="w-full p-3 bg-bg-secondary border border-border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :placeholder="t('review.post.gradePlaceholder')"
                  :aria-label="t('review.post.gradeLabel')"
                  maxlength="20"
                />
              </div>
            </template>
          </div>

          <div v-if="selectedCourse" class="flex justify-end gap-3 px-5 py-4 border-t border-border">
            <button class="py-2 px-4 text-sm text-text-secondary border border-border rounded-full cursor-pointer" @click="$emit('close')">
              {{ t('common.actions.cancel') }}
            </button>
            <button
              class="py-2 px-5 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-full cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
              :disabled="submitting"
              @click="handleSubmit"
            >
              {{ submitting ? t('common.actions.loading') : t('review.post.submit') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { searchCourses } from '@/api/course'
import { postReview } from '@/api/review'
import { useToast } from '@/composables/useToast'
import { useReviewPost } from '@/composables/useReviewPost'
import RatingGroup from './RatingGroup.vue'
import type { Course } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; posted: [] }>()

const { t } = useI18n()
const toast = useToast()
const { preselectedCourse } = useReviewPost()

const TITLE_MAX = 100
const CONTENT_MIN = 10
const CONTENT_MAX = 5000

const templateLabels = computed(() => [
  t('review.post.templateListening'),
  t('review.post.templateWorkload'),
  t('review.post.templateExam')
])
const contentTemplate = computed(() => templateLabels.value.join('\n'))

const modalRef = ref<HTMLElement | null>(null)
const ratingGroupRef = ref<InstanceType<typeof RatingGroup> | null>(null)

const courseQuery = ref('')
const courseResults = ref<Course[]>([])
const selectedCourse = ref<Course | null>(null)
const title = ref('')
const content = ref('')
const grade = ref('')
const ratings = ref<ReviewRatings>({})
const submitting = ref(false)
const attempted = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null

// 焦点陷阱：Tab/Shift+Tab 在对话框内循环
function trapFocus(e: KeyboardEvent) {
  if (e.key !== 'Tab' || !modalRef.value) return
  const focusable = modalRef.value.querySelectorAll<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
  )
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (e.shiftKey) {
    if (document.activeElement === first) { e.preventDefault(); last.focus() }
  } else {
    if (document.activeElement === last) { e.preventDefault(); first.focus() }
  }
}

watch(courseQuery, (val) => {
  if (searchTimer) clearTimeout(searchTimer)
  const q = val.trim()
  if (!q) { courseResults.value = []; return }

  const currentQuery = q
  searchTimer = setTimeout(async () => {
    try {
      const res = await searchCourses(currentQuery, 8)
      if (courseQuery.value.trim() === currentQuery) {
        courseResults.value = res.data?.list || []
      }
    } catch { courseResults.value = [] }
  }, 300)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

// 对话框打开时重置表单并聚焦，锁定 body 滚动
watch(() => props.visible, (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
    courseQuery.value = ''
    courseResults.value = []
    selectedCourse.value = preselectedCourse.value ?? null
    title.value = ''
    content.value = contentTemplate.value
    grade.value = ''
    ratings.value = {}
    submitting.value = false
    attempted.value = false
    nextTick(() => modalRef.value?.focus())
  } else {
    document.body.style.overflow = ''
    if (searchTimer) clearTimeout(searchTimer)
  }
})

function selectCourse(course: Course) {
  selectedCourse.value = course
  courseQuery.value = ''
  courseResults.value = []
}

// 去掉模板标签后的实际用户输入长度
function getUserContentLength(raw: string): number {
  let text = raw
  for (const label of templateLabels.value) {
    text = text.replaceAll(label, '')
  }
  return text.trim().length
}

const titleInvalid = computed(() => title.value.trim().length === 0)

const ratingsInvalid = computed(() => {
  const dims = ratingGroupRef.value?.dimensions ?? []
  if (dims.length === 0) return Object.keys(ratings.value).length === 0
  // 每个维度都必须评分
  return dims.some(d => {
    const v = ratings.value[d.key]
    return !v || v < 1 || v > 5
  })
})

const contentInvalid = computed(() =>
  getUserContentLength(content.value) < CONTENT_MIN
)

const contentError = computed(() => {
  const userLen = getUserContentLength(content.value)
  if (userLen > 0 && userLen < CONTENT_MIN) {
    return t('review.post.contentMinErrorNoTemplate', { min: CONTENT_MIN })
  }
  return ''
})

const canSubmit = computed(() => {
  return selectedCourse.value &&
    !titleInvalid.value &&
    !contentInvalid.value &&
    content.value.length <= CONTENT_MAX &&
    title.value.length <= TITLE_MAX &&
    !ratingsInvalid.value
})

async function handleSubmit() {
  attempted.value = true
  if (!canSubmit.value || !selectedCourse.value) return
  submitting.value = true
  try {
    await postReview({
      courseID: selectedCourse.value.id,
      title: title.value.trim() || undefined,
      content: content.value.trim(),
      grade: grade.value.trim() || undefined,
      ratings: ratings.value
    })
    toast.success(t('review.post.success'))
    emit('posted')
    emit('close')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
/* Vue Transition hooks — 无法用 utility 表达 */
.overlay-enter-active { animation: overlayIn var(--duration-base) var(--ease-out); }
.overlay-leave-active { animation: overlayIn var(--duration-fast) var(--ease-out) reverse; }
</style>
