<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="visible" class="fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] flex items-center justify-center p-4" @click.self="handleCancel">
        <div
          ref="modalRef"
          class="relative w-full max-w-[660px] max-h-[85vh] bg-bg-card rounded-xl shadow-xl flex flex-col overflow-hidden animate-modal-in"
          role="dialog"
          aria-modal="true"
          aria-labelledby="review-dialog-title"
          tabindex="-1"
          @keydown.esc="handleCancel"
          @keydown="trapFocus"
        >
          <div class="flex items-center justify-between p-5 border-b border-border-light">
            <h2 id="review-dialog-title" class="text-lg font-bold tracking-tight">{{ t('review.hub.postReview') }}</h2>
            <button class="text-2xl text-text-muted leading-none cursor-pointer w-8 h-8 flex items-center justify-center rounded-full transition-all duration-fast hover:text-text-primary hover:bg-bg-secondary" :aria-label="t('common.actions.close')" @click="$emit('close')">&times;</button>
          </div>

          <div class="flex-1 overflow-y-auto p-5 flex flex-col gap-4">
            <!-- 课程搜索 -->
            <div v-if="!selectedCourse">
              <input
                v-model="courseQuery"
                autocomplete="off"
                role="combobox"
                :aria-expanded="courseResults.length > 0"
                aria-haspopup="listbox"
                aria-autocomplete="list"
                :aria-activedescendant="highlightedIndex >= 0 ? `course-option-${courseResults[highlightedIndex]?.id}` : undefined"
                aria-controls="course-search-listbox"
                class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                :placeholder="t('review.post.searchCourse')"
                :aria-label="t('review.post.searchCourseLabel')"
                @keydown="handleSearchKeydown"
              />
              <div v-if="courseResults.length > 0" id="course-search-listbox" role="listbox" class="border-0 rounded-lg max-h-[200px] overflow-y-auto mt-2">
                <button
                  v-for="(c, idx) in courseResults"
                  :id="`course-option-${c.id}`"
                  :key="c.id"
                  role="option"
                  :aria-selected="idx === highlightedIndex"
                  class="flex items-center gap-2 w-full p-3 text-left text-sm text-text-primary cursor-pointer transition-[background] duration-fast hover:bg-bg-hover"
                  :class="{ 'bg-bg-hover': idx === highlightedIndex }"
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
              <!-- 教师选择器 -->
              <div class="relative">
                <label for="review-teacher-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.teacherLabel') }} <span class="text-text-muted font-normal text-xs">（{{ t('review.post.teacherOptional') }}）</span></label>
                <div v-if="loadingTeachers" class="text-xs text-text-muted py-2">{{ t('review.post.teacherLoading') }}</div>
                <div v-else class="relative">
                  <input
                    id="review-teacher-input"
                    v-model="teacherQuery"
                    autocomplete="off"
                    class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                    :placeholder="t('review.post.teacherPlaceholder')"
                    @focus="teacherDropdownOpen = true"
                    @input="selectedTeacherID = null"
                  />
                  <button v-if="teacherQuery" class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary" @click="clearTeacher">&times;</button>
                  <div v-if="teacherDropdownOpen && filteredTeachers.length > 0" class="absolute left-0 right-0 mt-1 rounded-lg bg-bg-card shadow-md max-h-[160px] overflow-y-auto z-10">
                    <button
                      v-for="teacher in filteredTeachers"
                      :key="teacher.teacherID"
                      class="flex items-center justify-between w-full p-2.5 text-left text-sm text-text-primary hover:bg-bg-hover transition-colors duration-fast"
                      @mousedown.prevent="selectTeacher(teacher)"
                    >
                      <span>{{ teacher.teacherName }}</span>
                      <span class="text-xs text-text-muted">{{ teacher.departmentName }}</span>
                    </button>
                  </div>
                </div>
              </div>

              <div class="relative">
                <label for="review-term-select" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.termLabel') }} <span class="text-danger text-xs">*</span></label>
                <select
                  id="review-term-select"
                  v-model="selectedTermID"
                  class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :class="attempted && !selectedTermID ? 'border-danger' : 'border-border'"
                >
                  <option value="" disabled>{{ t('review.post.termPlaceholder') }}</option>
                  <option v-for="term in termOptions" :key="term.id" :value="term.id">
                    {{ term.name }}
                  </option>
                </select>
                <span v-if="attempted && !selectedTermID" class="block text-xs text-danger mt-1">{{ t('review.post.termMissing') }}</span>
              </div>

              <div class="relative">
                <label for="review-title-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.titleRequired') }} <span class="text-danger text-xs">*</span></label>
                <input
                  id="review-title-input"
                  v-model="title"
                  autocomplete="off"
                  class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :class="attempted && titleInvalid ? 'border-danger' : 'border-border'"
                  :placeholder="t('review.post.titlePlaceholder')"
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
                <label for="review-content-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.detailedReview') }} <span class="text-danger text-xs">*</span></label>
                <textarea
                  id="review-content-input"
                  v-model="content"
                  class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[120px] transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :class="attempted && contentInvalid ? 'border-danger' : 'border-border'"
                  :placeholder="t('review.post.contentPlaceholder')"
                  :aria-describedby="contentError ? 'review-dialog-content-error' : undefined"
                  :maxlength="CONTENT_MAX"
                  rows="5"
                />
                <span v-if="contentError || (attempted && contentInvalid)" id="review-dialog-content-error" class="block text-xs text-danger mt-1">
                  {{ contentError || t('review.post.contentMinError', { min: CONTENT_MIN }) }}
                </span>
              </div>

              <div class="relative">
                <label for="review-grade-input" class="font-medium text-text-primary text-sm mb-1.5 block">{{ t('review.post.gradeLabel') }} <span class="text-text-muted font-normal text-xs">（{{ t('review.post.gradeOptional') }}）</span></label>
                <input
                  id="review-grade-input"
                  v-model="grade"
                  autocomplete="off"
                  class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
                  :placeholder="t('review.post.gradePlaceholder')"
                  maxlength="20"
                />
              </div>
            </template>
          </div>

          <div v-if="selectedCourse" class="flex justify-end gap-3 px-5 py-4 border-t border-border-light">
            <button class="py-2 px-4 text-sm text-text-secondary rounded-full cursor-pointer" @click="handleCancel">
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

          <!-- 未登录倒计时遮罩 -->
          <Transition name="overlay">
            <div v-if="redirectCountdown > 0" class="absolute inset-0 bg-bg-card/95 backdrop-blur-sm rounded-xl flex flex-col items-center justify-center gap-4 z-10">
              <div class="w-14 h-14 rounded-full border-3 border-primary flex items-center justify-center text-2xl font-bold text-primary animate-pulse">
                {{ redirectCountdown }}
              </div>
              <p class="text-sm text-text-primary font-medium">{{ t('review.draft.savedAndLogin') }}</p>
              <p class="text-xs text-text-muted">{{ t('review.draft.redirectCountdown', { n: redirectCountdown }) }}</p>
            </div>
          </Transition>
        </div>
      </div>
    </Transition>

    <!-- 取消确认小弹窗（独立于发布弹窗） -->
    <Transition name="overlay">
      <div v-if="showCancelConfirm" class="fixed inset-0 bg-black/30 z-[var(--z-modal)] flex items-center justify-center p-4" @click.self="showCancelConfirm = false">
        <div class="bg-bg-card rounded-xl shadow-2xl w-full max-w-[340px] p-6 flex flex-col items-center gap-4 animate-modal-in">
          <div class="w-11 h-11 rounded-full bg-warning/10 flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-warning"><path d="M14 3v4a1 1 0 0 0 1 1h4"/><path d="M17 21H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h7l5 5v11a2 2 0 0 1-2 2z"/><path d="M12 10v4"/><path d="M12 17h.01"/></svg>
          </div>
          <p class="text-sm font-medium text-text-primary text-center">{{ t('review.draft.confirmSave') }}</p>
          <div class="flex gap-3 w-full">
            <button
              class="flex-1 py-2 px-3 text-sm text-text-secondary rounded-lg cursor-pointer transition-colors duration-fast hover:bg-bg-secondary"
              @click="confirmDiscard"
            >
              {{ t('review.draft.discard') }}
            </button>
            <button
              class="flex-1 py-2 px-3 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-lg cursor-pointer transition-opacity duration-fast disabled:opacity-50"
              :disabled="savingDraft"
              @click="confirmSaveDraft"
            >
              {{ savingDraft ? t('review.draft.saving') : t('review.draft.saveDraft') }}
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
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { useToast } from '@/composables/useToast'
import { useReviewPost } from '@/composables/useReviewPost'
import { useAuthStore } from '@/stores/auth'
import {
  REVIEW_TITLE_MAX_LENGTH,
  REVIEW_CONTENT_MIN_LENGTH,
  REVIEW_CONTENT_MAX_LENGTH
} from '@/constants/review'
import RatingGroup from './RatingGroup.vue'
import type { Course, TeacherStats, Term } from '@/types/course'
import type { ReviewRatings } from '@/types/review'
import { buildCreateReviewPayload } from './reviewPayload'
import { buildTermOptions } from '@/modules/course/termOptions'
import {
  clearLocalReviewDraft,
  createDraftCourse,
  createLocalReviewDraft,
  getLocalReviewDraftClearedAt,
  loadLocalReviewDraft,
  sanitizeDraftRatings,
  saveLocalReviewDraft,
  type ReviewDialogServerDraft,
} from './reviewDialogDraftState'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; posted: [] }>()

const { t } = useI18n()
const toast = useToast()
const router = useRouter()
const authStore = useAuthStore()
const { preselectedCourse } = useReviewPost()

const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH
const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH
const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH
const AUTO_SAVE_DEBOUNCE_MS = 300
const GRADE_PATTERN = /^[A-Za-z0-9+\-./\s]*$/

// 硬编码的默认评分维度，API 失败时作为 fallback
const DEFAULT_RATING_DIMENSIONS = ['recommendation', 'content_quality', 'workload', 'grading']

// 合法的评分维度键名白名单
const VALID_RATING_KEYS = new Set(DEFAULT_RATING_DIMENSIONS)

const templateLabels = computed(() => [
  t('review.post.templateListening'),
  t('review.post.templateWorkload'),
  t('review.post.templateExam')
])
const contentTemplate = computed(() => templateLabels.value.join('\n'))

// 追踪用户是否手动编辑过内容，防止语言切换时覆盖用户输入
let userHasEditedContent = false

// 语言切换时，仅当用户未手动编辑且内容仍为旧模板时才更新
watch(contentTemplate, (newTpl, oldTpl) => {
  if (props.visible && !userHasEditedContent && content.value === oldTpl) {
    content.value = newTpl
  }
})

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
const redirectCountdown = ref(0)
const showCancelConfirm = ref(false)
const savingDraft = ref(false)
const teacherList = ref<TeacherStats[]>([])
const selectedTeacherID = ref<number | null>(null)
const teacherQuery = ref('')
const teacherDropdownOpen = ref(false)
const loadingTeachers = ref(false)
const terms = ref<Term[]>([])
const selectedTermID = ref('')
const termOptions = computed(() => buildTermOptions(terms.value))

let searchTimer: ReturnType<typeof setTimeout> | null = null
let courseSearchController: AbortController | undefined
let countdownTimer: ReturnType<typeof setInterval> | null = null
// 自动保存防抖 timer
let autoSaveTimer: ReturnType<typeof setTimeout> | null = null
// 草稿恢复版本追踪，防止并发恢复覆盖表单
let restoreVersion = 0
// beforeunload 事件管理
let unloadController: AbortController | null = null

// 课程搜索键盘导航
const highlightedIndex = ref(-1)

function handleSearchKeydown(e: KeyboardEvent) {
  const len = courseResults.value.length
  if (len === 0) return

  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      highlightedIndex.value = (highlightedIndex.value + 1) % len
      break
    case 'ArrowUp':
      e.preventDefault()
      highlightedIndex.value = (highlightedIndex.value - 1 + len) % len
      break
    case 'Enter':
      e.preventDefault()
      if (highlightedIndex.value >= 0 && highlightedIndex.value < len) {
        selectCourse(courseResults.value[highlightedIndex.value])
      }
      break
  }
}

function saveLocalDraft() {
  if (!selectedCourse.value) return
  saveLocalReviewDraft(createLocalReviewDraft({
    course: selectedCourse.value,
    teacherID: selectedTeacherID.value,
    teacherName: teacherQuery.value.trim() || undefined,
    title: title.value,
    content: content.value,
    grade: grade.value,
    termID: selectedTermID.value,
    ratings: ratings.value,
  }))
}

/** 判断表单是否有用户输入（排除模板文字） */
function hasFormContent(): boolean {
  return title.value.trim().length > 0 ||
    getUserContentLength(content.value) > 0 ||
    grade.value.trim().length > 0 ||
    Object.keys(ratings.value).length > 0
}

/** 保存草稿（已登录走服务端，未登录走 localStorage） */
async function saveDraftAuto() {
  if (!selectedCourse.value) return
  if (authStore.isAuthenticated) {
    try {
      await api.draft.saveDraft({
        courseID: selectedCourse.value.id,
        teacherID: selectedTeacherID.value ?? undefined,
        termID: selectedTermID.value || undefined,
        title: title.value.trim() || undefined,
        content: content.value.trim() || undefined,
        grade: grade.value?.trim() || undefined,
        ratings: Object.keys(ratings.value).length > 0 ? ratings.value : undefined
      })
    } catch {
      // 服务端保存失败时回退到 localStorage
      saveLocalDraft()
    }
  } else {
    saveLocalDraft()
  }
}

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
  highlightedIndex.value = -1
  const q = val.trim()
  if (!q) {
    if (courseSearchController) { courseSearchController.abort(); courseSearchController = undefined }
    courseResults.value = []
    return
  }

  searchTimer = setTimeout(async () => {
    if (courseSearchController) courseSearchController.abort()
    const controller = new AbortController()
    courseSearchController = controller

    try {
      const res = await api.course.searchCourses(q, 8, { signal: controller.signal })
      if (controller.signal.aborted) return
      const list = res.data?.data?.list || []
      const seen = new Set<number>()
      courseResults.value = list.filter((c: Course) => {
        if (seen.has(c.id)) return false
        seen.add(c.id)
        return true
      })
      highlightedIndex.value = -1
    } catch (err) {
      if (controller.signal.aborted) return
      console.warn('[ReviewDialog] Course search failed:', err)
      courseResults.value = []
    }
  }, 300)
})

onUnmounted(() => {
  document.body.style.overflow = ''
  if (searchTimer) clearTimeout(searchTimer)
  if (courseSearchController) { courseSearchController.abort(); courseSearchController = undefined }
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  clearCountdownTimer()
  unloadController?.abort()
  unloadController = null
})

function onBeforeUnload(e: BeforeUnloadEvent) {
  if (hasFormContent()) {
    e.preventDefault()
  }
}

function clearCountdownTimer() {
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
  redirectCountdown.value = 0
}

function startRedirectCountdown() {
  clearCountdownTimer()
  redirectCountdown.value = 3
  countdownTimer = setInterval(() => {
    redirectCountdown.value--
    if (redirectCountdown.value <= 0) {
      clearCountdownTimer()
      emit('close')
      authStore.login()
    }
  }, 1000)
}

// 表单变更时防抖自动保存草稿
watch([title, content, grade, ratings], () => {
  // 标记用户已手动编辑内容
  userHasEditedContent = true
  if (!selectedCourse.value) return
  if (autoSaveTimer) clearTimeout(autoSaveTimer)
  autoSaveTimer = setTimeout(() => { saveLocalDraft() }, AUTO_SAVE_DEBOUNCE_MS)
}, { deep: true })

// Grade 格式校验
const gradeInvalid = computed(() => {
  const g = grade.value.trim()
  return g.length > 0 && !GRADE_PATTERN.test(g)
})

// 对话框打开时重置表单并聚焦，锁定 body 滚动
watch(() => props.visible, async (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
    // 使用 AbortController 管理 beforeunload，防止监听器累积
    unloadController?.abort()
    unloadController = new AbortController()
    window.addEventListener('beforeunload', onBeforeUnload, { signal: unloadController.signal })
    courseQuery.value = ''
    courseResults.value = []
    selectedCourse.value = preselectedCourse.value ?? null
    title.value = ''
    content.value = contentTemplate.value
    grade.value = ''
    ratings.value = {}
    selectedTermID.value = ''
    selectedTeacherID.value = null
    teacherQuery.value = ''
    teacherDropdownOpen.value = false
    submitting.value = false
    attempted.value = false
    showCancelConfirm.value = false
    userHasEditedContent = false
    if (selectedCourse.value) {
      await Promise.all([
        loadTeachers(selectedCourse.value.id),
        loadTerms(),
      ])
    } else {
      teacherList.value = []
      terms.value = []
    }
    // 版本追踪，防止并发恢复覆盖表单
    const currentVersion = ++restoreVersion
    await nextTick()
    await tryRestoreDraft(currentVersion)

    // 聚焦首个可交互输入框而非 modal div
    nextTick(() => {
      const firstInput = modalRef.value?.querySelector<HTMLElement>('input, textarea')
      if (firstInput) firstInput.focus()
      else modalRef.value?.focus()
    })
  } else {
    document.body.style.overflow = ''
    if (searchTimer) clearTimeout(searchTimer)
    if (autoSaveTimer) clearTimeout(autoSaveTimer)
    clearCountdownTimer()
    unloadController?.abort()
    unloadController = null
  }
})

function selectCourse(course: Course) {
  selectedCourse.value = course
  courseQuery.value = ''
  courseResults.value = []
  loadTeachers(course.id)
  loadTerms()
}

async function loadTeachers(courseID: number) {
  loadingTeachers.value = true
  teacherList.value = []
  selectedTeacherID.value = null
  teacherQuery.value = ''
  try {
    const res = await api.rating.getCourseTeachers(courseID)
    teacherList.value = res.data?.data ?? []
  } catch { /* 静默忽略 */ }
  finally { loadingTeachers.value = false }
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

const filteredTeachers = computed(() => {
  const q = teacherQuery.value.trim().toLowerCase()
  if (!q) return teacherList.value
  return teacherList.value.filter(t => t.teacherName.toLowerCase().includes(q))
})

function selectTeacher(teacher: TeacherStats) {
  selectedTeacherID.value = teacher.teacherID
  teacherQuery.value = teacher.teacherName
  teacherDropdownOpen.value = false
}

function clearTeacher() {
  selectedTeacherID.value = null
  teacherQuery.value = ''
}

function applyTeacherDraftSelection(teacherID?: number | null, teacherName?: string) {
  if (teacherID === undefined || teacherID === null) {
    selectedTeacherID.value = null
    teacherQuery.value = teacherName ?? ''
    teacherDropdownOpen.value = false
    return
  }

  selectedTeacherID.value = teacherID
  const matched = teacherList.value.find((teacher) => teacher.teacherID === teacherID)
  teacherQuery.value = matched?.teacherName ?? teacherName ?? ''
  teacherDropdownOpen.value = false
}

/** 尝试恢复草稿：两端 + 清除时间戳三方比较，取最新的 */
async function tryRestoreDraft(expectedVersion: number) {
  const local = loadLocalReviewDraft(
    VALID_RATING_KEYS,
    selectedCourse.value?.id,
  )
  const clearedAt = getLocalReviewDraftClearedAt(local?.courseID ?? selectedCourse.value?.id)
  let serverDraft: ReviewDialogServerDraft | null = null

  // 已登录时尝试获取服务端草稿
  if (authStore.isAuthenticated && (selectedCourse.value || local)) {
    const courseID = selectedCourse.value?.id ?? local?.courseID
    if (courseID) {
      try {
        const res = await api.draft.getDraft(courseID)
        serverDraft = res.data?.data ?? null
      } catch { /* 静默忽略 */ }
    }
  }

  // 异步操作后检查版本，防止并发恢复覆盖表单
  if (restoreVersion !== expectedVersion) return

  // 收集各端时间戳，取最新事件
  const localTime = local?.updatedAt ?? 0
  const serverTime = serverDraft ? new Date(serverDraft.updatedAt).getTime() : 0
  const latest = Math.max(localTime, serverTime, clearedAt)

  // 最新事件是清除操作，或两端都没有 → 无草稿
  if (latest === 0 || latest === clearedAt) return

  if (latest === localTime && local) {
    if (!selectedCourse.value || selectedCourse.value.id !== local.courseID) {
      selectedCourse.value = createDraftCourse(local)
      await Promise.all([loadTeachers(local.courseID), loadTerms()])
    }
    if (local.title) title.value = local.title
    if (local.content) content.value = local.content
    if (local.grade) grade.value = local.grade
    selectedTermID.value = local.termID
    if (local.ratings && Object.keys(local.ratings).length > 0) ratings.value = local.ratings
    applyTeacherDraftSelection(local.teacherID ?? null, local.teacherName)
  } else if (serverDraft) {
    if (serverDraft.title) title.value = serverDraft.title
    if (serverDraft.content) content.value = serverDraft.content
    if (serverDraft.grade) grade.value = serverDraft.grade
    if (serverDraft.termID) selectedTermID.value = serverDraft.termID
    applyTeacherDraftSelection(serverDraft.teacherID ?? null)
    const sanitizedRatings = sanitizeDraftRatings(serverDraft.ratings)
    if (sanitizedRatings && Object.keys(sanitizedRatings).length > 0) {
      ratings.value = sanitizedRatings
    }
  }

  toast.info(t('review.draft.restored'))
}

// 去掉模板标签后的实际用户输入长度
function getUserContentLength(raw: string): number {
  let text = raw
  for (const label of templateLabels.value) {
    text = text.split(label).join('')
  }
  return text.trim().length
}

const titleInvalid = computed(() => title.value.trim().length === 0)

const ratingsInvalid = computed(() => {
  // API 失败时使用硬编码默认维度作为 fallback
  const dims = ratingGroupRef.value?.dimensions ?? []
  const keys = dims.length > 0 ? dims.map(d => d.key) : DEFAULT_RATING_DIMENSIONS
  return keys.some(k => {
    const v = ratings.value[k]
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
    !gradeInvalid.value &&
    content.value.length <= CONTENT_MAX &&
    title.value.length <= TITLE_MAX &&
    !ratingsInvalid.value
})

async function handleSubmit() {
  // 防重复提交：即使按钮 disabled 未渲染，也阻止快速双击或 Enter 键触发
  if (submitting.value) return
  attempted.value = true
  if (!canSubmit.value || !selectedCourse.value) return

  // 未登录：暂存草稿到 localStorage，倒计时后跳转 SSO 登录
  if (!authStore.isAuthenticated) {
    saveLocalDraft()
    sessionStorage.setItem('draft_redirect', router.currentRoute.value.fullPath)
    sessionStorage.setItem('draft_pending', '1')
    startRedirectCountdown()
    return
  }

  // 已登录：正常发布
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

    await api.review.createReview(buildCreateReviewPayload({
      courseID: selectedCourse.value.id,
      teacherID: selectedTeacherID.value ?? undefined,
      termID: selectedTermID.value,
      title: title.value.trim(),
      content: content.value.trim(),
      grade: grade.value?.trim() || undefined,
      ratings: ratings.value
    }))
    // 发布成功后清除草稿
    clearLocalReviewDraft(selectedCourse.value.id)
    try { await api.draft.deleteDraft(selectedCourse.value.id) } catch { /* 忽略 */ }
    toast.success(t('review.post.success'))
    emit('posted')
    emit('close')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}

/** 取消按钮：倒计时期间禁止关闭，有内容时弹出自定义确认面板 */
function handleCancel() {
  if (redirectCountdown.value > 0) return
  if (selectedCourse.value && hasFormContent()) {
    showCancelConfirm.value = true
    return
  }
  emit('close')
}

async function confirmSaveDraft() {
  savingDraft.value = true
  try {
    await saveDraftAuto()
    toast.success(t('review.draft.saved'))
    showCancelConfirm.value = false
    emit('close')
  } catch {
    toast.error(t('review.draft.saveFailed'))
    showCancelConfirm.value = false
  } finally {
    savingDraft.value = false
  }
}

function confirmDiscard() {
  // 丢弃时清除已有草稿，避免下次打开仍恢复
  if (selectedCourse.value) {
    clearLocalReviewDraft(selectedCourse.value.id)
  } else {
    clearLocalReviewDraft()
  }
  if (authStore.isAuthenticated && selectedCourse.value) {
    api.draft.deleteDraft(selectedCourse.value.id).catch(() => {})
  }
  showCancelConfirm.value = false
  emit('close')
}
</script>

<style scoped>
/* Vue Transition hooks — 无法用 utility 表达 */
.overlay-enter-active { animation: overlayIn var(--duration-base) var(--ease-out); }
.overlay-leave-active { animation: overlayIn var(--duration-fast) var(--ease-out) reverse; }
</style>
