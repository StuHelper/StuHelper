<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="visible" class="fixed inset-0 bg-bg-overlay z-50 flex items-center justify-center p-4" @click.self="handleCancel">
        <div
          ref="modalRef"
          class="relative w-full max-w-[660px] max-h-[85vh] bg-bg-card border border-border rounded-xl shadow-xl flex flex-col overflow-hidden animate-modal-in"
          role="dialog"
          aria-modal="true"
          aria-labelledby="review-dialog-title"
          tabindex="-1"
          @keydown.esc="handleCancel"
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
            <button class="py-2 px-4 text-sm text-text-secondary border border-border rounded-full cursor-pointer" @click="handleCancel">
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
      <div v-if="showCancelConfirm" class="fixed inset-0 bg-black/30 z-[60] flex items-center justify-center p-4" @click.self="showCancelConfirm = false">
        <div class="bg-bg-card border border-border rounded-xl shadow-2xl w-full max-w-[340px] p-6 flex flex-col items-center gap-4 animate-modal-in">
          <div class="w-11 h-11 rounded-full bg-warning/10 flex items-center justify-center">
            <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-warning"><path d="M14 3v4a1 1 0 0 0 1 1h4"/><path d="M17 21H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h7l5 5v11a2 2 0 0 1-2 2z"/><path d="M12 10v4"/><path d="M12 17h.01"/></svg>
          </div>
          <p class="text-sm font-medium text-text-primary text-center">{{ t('review.draft.confirmSave') }}</p>
          <div class="flex gap-3 w-full">
            <button
              class="flex-1 py-2 px-3 text-sm text-text-secondary border border-border rounded-lg cursor-pointer transition-colors duration-fast hover:bg-bg-secondary"
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
import { searchCourses } from '@/api/course'
import { postReview } from '@/api/review'
import { saveDraft, getDraft, deleteDraft } from '@/api/draft'
import { useToast } from '@/composables/useToast'
import { useReviewPost } from '@/composables/useReviewPost'
import { useAuthStore } from '@/stores/auth'
import RatingGroup from './RatingGroup.vue'
import type { Course } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; posted: [] }>()

const { t } = useI18n()
const toast = useToast()
const router = useRouter()
const authStore = useAuthStore()
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
const redirectCountdown = ref(0)
const showCancelConfirm = ref(false)
const savingDraft = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null

// --- localStorage 草稿工具 ---
const LOCAL_DRAFT_KEY = 'review_draft'
const LOCAL_DRAFT_CLEARED_KEY = 'review_draft_cleared'

interface LocalDraft {
  courseID: number
  courseName: string
  title: string
  content: string
  grade: string
  ratings: ReviewRatings
  updatedAt: number
}

function saveLocalDraft() {
  if (!selectedCourse.value) return
  const draft: LocalDraft = {
    courseID: selectedCourse.value.id,
    courseName: selectedCourse.value.name,
    title: title.value,
    content: content.value,
    grade: grade.value,
    ratings: ratings.value,
    updatedAt: Date.now()
  }
  localStorage.setItem(LOCAL_DRAFT_KEY, JSON.stringify(draft))
  localStorage.removeItem(LOCAL_DRAFT_CLEARED_KEY)
}

function loadLocalDraft(): LocalDraft | null {
  try {
    const raw = localStorage.getItem(LOCAL_DRAFT_KEY)
    if (!raw) return null
    return JSON.parse(raw) as LocalDraft
  } catch { return null }
}

function clearLocalDraft() {
  localStorage.removeItem(LOCAL_DRAFT_KEY)
  localStorage.setItem(LOCAL_DRAFT_CLEARED_KEY, String(Date.now()))
}

function getLocalClearedAt(): number {
  return Number(localStorage.getItem(LOCAL_DRAFT_CLEARED_KEY)) || 0
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
      await saveDraft({
        courseID: selectedCourse.value.id,
        title: title.value.trim() || undefined,
        content: content.value.trim() || undefined,
        grade: grade.value.trim() || undefined,
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
  clearCountdownTimer()
  window.removeEventListener('beforeunload', onBeforeUnload)
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

// 对话框打开时重置表单并聚焦，锁定 body 滚动
watch(() => props.visible, async (val) => {
  if (val) {
    document.body.style.overflow = 'hidden'
    window.addEventListener('beforeunload', onBeforeUnload)
    courseQuery.value = ''
    courseResults.value = []
    selectedCourse.value = preselectedCourse.value ?? null
    title.value = ''
    content.value = contentTemplate.value
    grade.value = ''
    ratings.value = {}
    submitting.value = false
    attempted.value = false
    showCancelConfirm.value = false

    // 尝试恢复草稿
    await nextTick()
    await tryRestoreDraft()

    nextTick(() => modalRef.value?.focus())
  } else {
    document.body.style.overflow = ''
    if (searchTimer) clearTimeout(searchTimer)
    clearCountdownTimer()
    window.removeEventListener('beforeunload', onBeforeUnload)
  }
})

function selectCourse(course: Course) {
  selectedCourse.value = course
  courseQuery.value = ''
  courseResults.value = []
}

/** 尝试恢复草稿：两端 + 清除时间戳三方比较，取最新的 */
async function tryRestoreDraft() {
  const local = loadLocalDraft()
  const clearedAt = getLocalClearedAt()
  let serverDraft: { title?: string; content?: string; grade?: string; ratings?: ReviewRatings; updatedAt: string } | null = null

  // 已登录时尝试获取服务端草稿
  if (authStore.isAuthenticated && (selectedCourse.value || local)) {
    const courseID = selectedCourse.value?.id ?? local?.courseID
    if (courseID) {
      try {
        const res = await getDraft(courseID)
        serverDraft = res.data ?? null
      } catch { /* 静默忽略 */ }
    }
  }

  // 收集各端时间戳，取最新事件
  const localTime = local?.updatedAt ?? 0
  const serverTime = serverDraft ? new Date(serverDraft.updatedAt).getTime() : 0
  const latest = Math.max(localTime, serverTime, clearedAt)

  // 最新事件是清除操作，或两端都没有 → 无草稿
  if (latest === 0 || latest === clearedAt) return

  if (latest === localTime && local) {
    if (!selectedCourse.value) {
      selectedCourse.value = { id: local.courseID, name: local.courseName } as Course
    }
    if (local.title) title.value = local.title
    if (local.content) content.value = local.content
    if (local.grade) grade.value = local.grade
    if (local.ratings && Object.keys(local.ratings).length > 0) ratings.value = local.ratings
  } else if (serverDraft) {
    if (serverDraft.title) title.value = serverDraft.title
    if (serverDraft.content) content.value = serverDraft.content
    if (serverDraft.grade) grade.value = serverDraft.grade
    if (serverDraft.ratings && Object.keys(serverDraft.ratings).length > 0) ratings.value = serverDraft.ratings
  }

  toast.info(t('review.draft.restored'))
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
    await postReview({
      courseID: selectedCourse.value.id,
      title: title.value.trim() || undefined,
      content: content.value.trim(),
      grade: grade.value.trim() || undefined,
      ratings: ratings.value
    })
    // 发布成功后清除草稿
    clearLocalDraft()
    try { await deleteDraft(selectedCourse.value.id) } catch { /* 忽略 */ }
    toast.success(t('review.post.success'))
    emit('posted')
    emit('close')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}

/** 取消按钮：有内容时弹出自定义确认面板 */
function handleCancel() {
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
  } finally {
    savingDraft.value = false
    showCancelConfirm.value = false
    emit('close')
  }
}

function confirmDiscard() {
  // 丢弃时清除已有草稿，避免下次打开仍恢复
  clearLocalDraft()
  if (authStore.isAuthenticated && selectedCourse.value) {
    deleteDraft(selectedCourse.value.id).catch(() => {})
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
