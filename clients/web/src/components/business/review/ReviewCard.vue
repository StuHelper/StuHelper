<template>
  <div
    ref="cardRef"
    class="relative rounded-xl p-5 glass-card hover-lift"
    :style="tiltStyle"
    :class="{
      'border-primary/30 shadow-glow-sm': isExpanded,
      'animate-shake': shaking,
      'border-warning/30': isHidden
    }"
  >
    <!-- 管理员工具栏 -->
    <div v-if="canManageReviews" class="absolute top-3 right-3 flex items-center gap-1">
      <button
        v-if="!isHidden"
        class="p-1.5 rounded-lg text-text-muted hover:text-warning hover:bg-warning/10 cursor-pointer transition-colors"
        :title="t('review.admin.hide')"
        @click="showModerationDialog = true"
      >
        <EyeOff :size="16" />
      </button>
      <template v-else>
        <button
          class="p-1.5 rounded-lg text-text-muted hover:text-primary hover:bg-primary/10 cursor-pointer transition-colors"
          :title="t('review.admin.restore')"
          @click="handleRestore"
        >
          <Eye :size="16" />
        </button>
        <button
          class="p-1.5 rounded-lg text-text-muted hover:text-primary hover:bg-primary/10 cursor-pointer transition-colors"
          :title="t('review.admin.edit')"
          @click="showEditDialog = true"
        >
          <Pencil :size="16" />
        </button>
      </template>
    </div>

    <!-- 头部：课程名 + 评分徽章 -->
    <div class="flex items-center gap-2 mb-2 pr-20">
      <router-link
        :to="`/courses/${review.courseID}/reviews`"
        class="text-base font-bold text-text-primary no-underline overflow-hidden text-ellipsis whitespace-nowrap hover:text-primary"
      >
        {{ review.title || review.courseName || `Course #${review.courseID}` }}
      </router-link>
      <span
        v-if="avgRating > 0"
        class="text-[11px] font-bold font-mono py-px px-2 rounded-full shrink-0"
        :style="{
          '--rating-clr': ratingColor,
          background: 'color-mix(in srgb, var(--rating-clr, var(--color-primary)) 12%, transparent)',
          color: 'var(--rating-clr, var(--color-primary))'
        }"
      >
        {{ avgRating.toFixed(1) }}
      </span>
    </div>

    <!-- 元数据：教师 + 时间 -->
    <div class="flex items-center gap-1.5 mb-3 text-xs">
      <span v-if="review.teacherName" class="text-teacher-tag font-medium">{{ review.teacherName }}</span>
      <span v-if="review.teacherName" class="text-text-muted">&middot;</span>
      <span class="text-text-muted">{{ formattedTime }}</span>
    </div>

    <!-- 内容区：三态显示 -->
    <!-- 状态1: 屏蔽锁定（非管理员看到锁定提示） -->
    <div v-if="isHidden && !canManageReviews" class="flex items-start gap-2 py-4 px-3 bg-warning/5 border border-warning/20 rounded-lg">
      <ShieldAlert :size="18" class="text-warning shrink-0 mt-0.5" />
      <div>
        <span class="text-sm font-medium text-text-secondary">{{ t('review.card.contentHidden') }}</span>
        <p v-if="review.moderationReason" class="text-xs text-text-muted mt-1" v-text="review.moderationReason" />
      </div>
    </div>

    <!-- 状态2: 未登录锁定 -->
    <div v-else-if="!isAuthenticated && !isHidden" class="flex flex-col items-center gap-2 py-6 text-text-muted">
      <Lock :size="24" />
      <span class="text-sm">{{ t('review.card.loginToView') }}</span>
      <button class="text-primary text-sm font-medium cursor-pointer" @click="handleLogin">
        {{ t('review.card.loginBtn') }}
      </button>
    </div>

    <!-- 状态2.5: 已登录未认证 — 预览模式（D8: 模糊 + 锁图标） -->
    <div v-else-if="isPreviewMode" class="relative">
      <div class="text-sm text-text-secondary leading-relaxed break-words" v-text="previewContent" />
      <div class="relative mt-1 h-16 overflow-hidden select-none" aria-hidden="true">
        <div class="absolute inset-0 bg-bg-card/60 backdrop-blur-md flex flex-col items-center justify-center gap-1.5 rounded-lg">
          <Lock :size="16" class="text-text-muted" />
          <span class="text-xs text-text-muted">{{ t('review.card.verifyToView') }}</span>
          <button
            class="text-primary text-xs font-medium cursor-pointer hover:underline"
            @click="$router.push({ name: 'student-verification' })"
          >
            {{ t('review.card.goVerify') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 状态3: 正常显示（已登录 + published，或管理员查看 hidden） -->
    <template v-else>
      <!-- 使用 v-text 渲染用户内容，不解析 HTML -->
      <div
        class="text-sm text-text-secondary leading-relaxed cursor-pointer break-words"
        :class="{ 'line-clamp-3': !isExpanded && shouldTruncate }"
        role="button"
        tabindex="0"
        :aria-label="t('review.review.expandContent')"
        :aria-expanded="isExpanded"
        @click="toggleExpand"
        @keydown.enter="toggleExpand"
        @keydown.space.prevent="toggleExpand"
        v-text="review.content"
      />
    </template>

    <!-- 表情评分指标 -->
    <div v-if="displayRatings.length > 0" class="flex flex-wrap gap-3 mt-4 pt-3 border-t border-border-light">
      <span
        v-for="dim in displayRatings"
        :key="dim.key"
        class="text-xs text-text-secondary flex items-center gap-1"
      >
        <span>{{ dim.emoji }}</span>
        <span>{{ dim.label }}</span>
        <span class="font-mono font-medium text-text-primary">{{ dim.value.toFixed(1) }}</span>
      </span>
    </div>

    <!-- 操作栏（未登录或屏蔽时隐藏） -->
    <div
      v-if="showActions"
      class="flex items-center gap-1 mt-3"
      :class="{ 'pt-3 border-t border-border-light': displayRatings.length === 0 }"
    >
      <button
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover press-spring"
        :class="{ '!text-primary bg-primary/[0.08]': userVote === 'like' }"
        :aria-label="t('review.vote.like')"
        :aria-pressed="userVote === 'like'"
        @click="handleVote('like')"
      >
        <Heart :size="16" :fill="userVote === 'like' ? 'currentColor' : 'none'" />
        <span class="text-xs font-mono" :class="{ 'animate-vote-bounce': likeBounce }">
          {{ displayLikes }}
        </span>
      </button>

      <button
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover press-spring"
        :class="{ '!text-primary bg-primary/[0.08]': userVote === 'dislike' }"
        :aria-label="t('review.vote.dislike')"
        :aria-pressed="userVote === 'dislike'"
        @click="handleVote('dislike')"
      >
        <ThumbsDown :size="16" />
        <span class="text-xs font-mono">{{ displayDislikes }}</span>
      </button>

      <button
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover press-spring"
        :aria-label="t('review.review.commentBtn')"
        @click="toggleExpand"
      >
        <MessageCircle :size="16" />
        <span class="text-xs font-mono">{{ replyCount }}</span>
      </button>

      <!-- 举报按钮（非自己的评价） -->
      <button
        v-if="!props.isOwnReview"
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-warning hover:bg-warning/10 press-spring ml-auto"
        :aria-label="t('review.review.reportBtn')"
        @click="showReportMenu = !showReportMenu"
      >
        <Flag :size="16" />
      </button>

      <!-- 编辑按钮（自己的评价） -->
      <button
        v-if="props.isOwnReview && !editing"
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-primary hover:bg-primary/10 press-spring ml-auto"
        :aria-label="t('review.review.editBtn')"
        @click="startEditing"
      >
        <Pencil :size="16" />
      </button>

      <!-- 删除按钮（自己的评价） -->
      <button
        v-if="props.isOwnReview"
        v-ripple
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-danger hover:bg-danger/10 press-spring"
        :class="{ 'ml-auto': editing }"
        :aria-label="t('review.review.deleteBtn')"
        :disabled="deleting"
        @click="handleDeleteOwn"
      >
        <Trash2 :size="16" />
      </button>
    </div>

    <!-- 举报下拉菜单 -->
    <div
      v-if="showReportMenu"
      class="mt-2 p-3 bg-bg-card rounded-lg border border-border-light shadow-card animate-fade-in"
    >
      <p class="text-xs font-medium text-text-primary m-0 mb-2">{{ t('review.review.reportReason') }}</p>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="reason in reportReasons"
          :key="reason"
          class="text-xs px-3 py-1.5 rounded-full bg-bg-hover text-text-secondary cursor-pointer transition-colors hover:bg-warning/10 hover:text-warning"
          :disabled="reporting"
          @click="handleReport(reason)"
        >
          {{ t(`review.review.reportReasons.${reason}`) }}
        </button>
      </div>
    </div>

    <!-- 内联编辑表单 -->
    <div v-if="editing" class="mt-3 p-3 bg-bg-base rounded-lg border border-border-light animate-fade-in">
      <textarea
        v-model="editContent"
        class="w-full px-3 py-2 bg-transparent rounded-lg text-sm text-text-primary outline-none border border-border-light focus:border-primary resize-y min-h-[100px]"
        :placeholder="t('review.review.editPlaceholder')"
      />
      <div class="flex justify-end gap-2 mt-2">
        <button
          class="px-4 py-1.5 text-xs rounded-full bg-transparent text-text-muted cursor-pointer transition-colors hover:text-text-primary"
          @click="cancelEditing"
        >
          {{ t('common.actions.cancel') }}
        </button>
        <button
          class="px-4 py-1.5 text-xs rounded-full bg-primary text-white cursor-pointer transition-opacity hover:opacity-90 disabled:opacity-50"
          :disabled="saving || !editContent.trim()"
          @click="handleSaveEdit"
        >
          {{ saving ? t('common.actions.saving') : t('common.actions.save') }}
        </button>
      </div>
    </div>

    <!-- 展开区域：回复列表 + 回复输入 -->
    <div v-if="isExpanded" class="mt-4 pt-4 border-t border-border-light animate-fade-in-up">
      <div v-if="repliesLoading" class="flex justify-center py-4">
        <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin" />
      </div>
      <div v-else-if="repliesError" class="text-center text-accent text-sm py-4 flex items-center justify-center gap-2">
        {{ t('review.review.replyLoadFailed') }}
        <button class="text-primary text-sm cursor-pointer underline" @click="loadReplies">
          {{ t('common.actions.retry') }}
        </button>
      </div>
      <div v-else-if="replies.length > 0" class="mb-2">
        <ReplyCard
          v-for="reply in replies"
          :key="reply.id"
          :reply="reply"
          @delete="handleDeleteReply"
        />
      </div>
      <div v-else class="text-center text-text-muted text-sm py-4">
        {{ t('review.review.noReplies') }}
      </div>

      <ReplyForm
        ref="replyFormRef"
        :submitting="replySubmitting"
        @submit="handleReplySubmit"
        @cancel="isExpanded = false"
      />
    </div>

    <!-- 屏蔽弹窗 -->
    <ModerationDialog
      :visible="showModerationDialog"
      :review-i-d="review.id"
      @confirm="handleModerate"
      @close="showModerationDialog = false"
    />

    <!-- 编辑弹窗 -->
    <AdminEditDialog
      :visible="showEditDialog"
      :review="review"
      @confirm="handleAdminEdit"
      @close="showEditDialog = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Heart, ThumbsDown, MessageCircle, EyeOff, Eye, Pencil, Lock, ShieldAlert, Flag, Trash2 } from 'lucide-vue-next'
import type { Review } from '@/types/review'
import type { Reply } from '@/types/reply'
import { api } from '@/api'
import { ADMIN_REVIEWS_MANAGE, hasCapability } from '@stuhelper/shared/constants'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import { use3DTilt } from '@/composables/use3DTilt'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'
import ModerationDialog from './ModerationDialog.vue'
import AdminEditDialog from './AdminEditDialog.vue'
import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
  type VoteType,
} from './reviewVoteState'

const props = defineProps<{
  review: Review
  isOwnReview?: boolean
}>()

const emit = defineEmits<{
  moderated: []
  deleted: [id: string]
  updated: [id: string, content: string]
}>()

const { t, locale } = useI18n()
const toast = useToast()
const authStore = useAuthStore()

// 3D tilt effect
const cardRef = ref<HTMLElement>()
const { style: tiltStyle } = use3DTilt(cardRef, { maxTilt: 4, scale: 1.01, speed: 500 })

const isAuthenticated = computed(() => authStore.isAuthenticated)
const canManageReviews = computed(() =>
  hasCapability(authStore.globalCapabilities, ADMIN_REVIEWS_MANAGE),
)
const isHidden = computed(() => props.review.status === 'hidden')
const showActions = computed(() => isAuthenticated.value && !isHidden.value)

// Graded display - preview mode for logged-in but unverified users
const verificationStore = useVerificationStore()
const canViewFull = computed(() => canManageReviews.value || verificationStore.canViewFullReviews)
const isPreviewMode = computed(() => isAuthenticated.value && !isHidden.value && !canViewFull.value)

const PREVIEW_CHARS = 20
const PREVIEW_PERCENT = 20
const previewContent = computed(() => {
  const content = props.review.content || ''
  const byChars = PREVIEW_CHARS
  const byPercent = Math.floor(content.length * PREVIEW_PERCENT / 100)
  const previewLen = Math.min(byChars, byPercent)
  if (content.length <= previewLen) return content
  return content.slice(0, previewLen)
})

const isExpanded = ref(false)
const shouldTruncate = computed(() => (props.review.content?.length ?? 0) > 200)

// 管理员弹窗状态
const showModerationDialog = ref(false)
const showEditDialog = ref(false)

// 举报 & 删除状态
const showReportMenu = ref(false)
const reporting = ref(false)
const deleting = ref(false)
const reportReasons = ['spam', 'inappropriate', 'harassment', 'false_info', 'other'] as const
type ReportReason = typeof reportReasons[number]

// 编辑状态
const editing = ref(false)
const editContent = ref('')
const saving = ref(false)

function startEditing() {
  editContent.value = props.review.content
  editing.value = true
}

function cancelEditing() {
  editing.value = false
  editContent.value = ''
}

// 投票状态
const userVote = ref<VoteType | null>(null)
const isVoting = ref(false)
const likeOffset = ref(0)
const dislikeOffset = ref(0)
const likeBounce = ref(false)
const shaking = ref(false)

// 评分值钳位，防止负值导致进度条异常
const displayLikes = computed(() => getDisplayVoteCount(props.review.likeCount, likeOffset.value))
const displayDislikes = computed(() => getDisplayVoteCount(props.review.dislikeCount, dislikeOffset.value))

// 表情评分指标映射，通过 i18n 获取 emoji（支持本地化）
const displayRatings = computed(() => {
  const ratings = props.review.ratings
  if (!ratings || Object.keys(ratings).length === 0) return []
  return Object.entries(ratings).map(([key, value]) => {
    const emoji = t(`review.ratingEmoji.icon.${key}`, '⭐')
    const label = t(`review.ratingEmoji.${key}`, t('review.ratingEmoji.fallback'))
    // 钳位评分值，防止负值或超范围值
    const clampedValue = Math.max(0, Math.min(5, value))
    return { key, emoji, label, value: clampedValue }
  })
})

// 评分
const avgRating = computed(() => {
  const ratings = Object.values(props.review.ratings || {})
  if (ratings.length === 0) return 0
  return ratings.reduce((a, b) => a + b, 0) / ratings.length
})

const ratingColor = computed(() => {
  if (avgRating.value >= 4) return 'var(--color-rating-5)'
  if (avgRating.value >= 3) return 'var(--color-rating-4)'
  if (avgRating.value >= 2) return 'var(--color-rating-3)'
  return 'var(--color-rating-1)'
})

// 回复
const replies = ref<Reply[]>([])
const repliesLoading = ref(false)
const repliesError = ref(false)
const replySubmitting = ref(false)
const replyCount = ref(props.review.replyCount ?? 0)
const replyFormRef = ref<InstanceType<typeof ReplyForm> | null>(null)

// 同步 props 变化（仅在无本地修改时同步，避免覆盖乐观更新）
let replyCountDirty = false
watch(() => props.review.replyCount, (val) => {
  if (val !== undefined && !replyCountDirty) replyCount.value = val
})

// 当 review 数据刷新时重置投票偏移量
watch(() => props.review, () => {
  likeOffset.value = 0
  dislikeOffset.value = 0
  userVote.value = null
  replyCountDirty = false
})

// 定时器清理后置空，释放闭包引用
let bounceTimer: ReturnType<typeof setTimeout> | null = null
let shakeTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => {
  if (bounceTimer) { clearTimeout(bounceTimer); bounceTimer = null }
  if (shakeTimer) { clearTimeout(shakeTimer); shakeTimer = null }
})

// 时间格式化（computed 缓存）
const formattedTime = computed(() => formatRelativeTime(props.review.createdAt, locale.value, t))

// 投票
async function handleVote(type: VoteType) {
  if (isVoting.value) return
  isVoting.value = true

  const previousState = createReviewVoteState({
    userVote: userVote.value,
    likeOffset: likeOffset.value,
    dislikeOffset: dislikeOffset.value,
  })
  const nextState = applyOptimisticVote(previousState, type)
  const shouldBounceLike = previousState.userVote !== 'like' && nextState.userVote === 'like'
  const targetReviewId = props.review.id

  userVote.value = nextState.userVote
  likeOffset.value = nextState.likeOffset
  dislikeOffset.value = nextState.dislikeOffset

  if (shouldBounceLike) {
    likeBounce.value = true
    bounceTimer = setTimeout(() => { likeBounce.value = false; bounceTimer = null }, 300)
  }

  try {
    await api.review.voteReview(props.review.id, { voteType: type })
  } catch {
    if (props.review.id === targetReviewId) {
      userVote.value = previousState.userVote
      likeOffset.value = previousState.likeOffset
      dislikeOffset.value = previousState.dislikeOffset
      shaking.value = true
      shakeTimer = setTimeout(() => { shaking.value = false; shakeTimer = null }, 300)
      toast.error(t('review.review.voteFailed'))
    }
  } finally {
    isVoting.value = false
  }
}

// 展开/收起
async function toggleExpand() {
  isExpanded.value = !isExpanded.value
  if (isExpanded.value) {
    await loadReplies()
  }
}

async function loadReplies() {
  repliesLoading.value = true
  repliesError.value = false
  try {
    const res = await api.reply.getReplies(props.review.id)
    replies.value = res.data?.data?.list || []
    replyCount.value = res.data?.data?.total || 0
    replyCountDirty = false
  } catch {
    replies.value = []
    repliesError.value = true
  } finally {
    repliesLoading.value = false
  }
}

async function handleReplySubmit(content: string) {
  replySubmitting.value = true
  try {
    const res = await api.reply.createReply(props.review.id, { content })
    if (res.data?.data) {
      replies.value = [...replies.value, res.data.data]
      replyCount.value++
      replyCountDirty = true
      replyFormRef.value?.clear()
      toast.success(t('review.review.replySuccess'))
    }
  } catch {
    toast.error(t('review.review.replyFailed'))
  } finally {
    replySubmitting.value = false
  }
}

async function handleDeleteReply(id: string) {
  try {
    await api.reply.deleteReply(id)
    replies.value = replies.value.filter((r) => r.id !== id)
    replyCount.value = Math.max(0, replyCount.value - 1)
    replyCountDirty = true
  } catch {
    toast.error(t('review.review.deleteFailed'))
  }
}

// 登录跳转
function handleLogin() {
  authStore.login()
}

// 举报评价
async function handleReport(reason: ReportReason) {
  reporting.value = true
  try {
    await api.review.reportReview(props.review.id, { reason })
    toast.success(t('review.review.reportSuccess'))
    showReportMenu.value = false
  } catch {
    toast.error(t('review.review.reportFailed'))
  } finally {
    reporting.value = false
  }
}

// 删除自己的评价
async function handleDeleteOwn() {
  deleting.value = true
  try {
    await api.review.deleteReview(props.review.id)
    toast.success(t('review.review.deleteSuccess'))
    emit('deleted', props.review.id)
  } catch {
    toast.error(t('review.review.deleteFailed'))
  } finally {
    deleting.value = false
  }
}

// 保存编辑
async function handleSaveEdit() {
  const trimmed = editContent.value.trim()
  if (!trimmed) return
  saving.value = true
  try {
    await api.review.updateReview(props.review.id, {
      content: trimmed,
      ratings: props.review.ratings,
    })
    toast.success(t('review.review.editSuccess'))
    editing.value = false
    emit('updated', props.review.id, trimmed)
  } catch {
    toast.error(t('review.review.editFailed'))
  } finally {
    saving.value = false
  }
}

// 管理员屏蔽
async function handleModerate(reason: string) {
  showModerationDialog.value = false
  try {
    await api.admin.updateReview(props.review.id, { action: 'hide', reason })
    toast.success(t('review.admin.moderateSuccess'))
    emit('moderated')
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

// 管理员恢复
async function handleRestore() {
  try {
    await api.admin.updateReview(props.review.id, { action: 'restore' })
    toast.success(t('review.admin.restoreSuccess'))
    emit('moderated')
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

// 管理员编辑
async function handleAdminEdit(payload: { title: string; content: string; reason: string }) {
  showEditDialog.value = false
  try {
    await api.admin.editReview(props.review.id, payload)
    toast.success(t('review.admin.editSuccess'))
    emit('moderated')
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}
</script>
