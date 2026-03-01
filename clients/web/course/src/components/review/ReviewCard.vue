<template>
  <div
    class="relative bg-bg-card border border-border rounded-xl p-5 transition-all duration-200 ease-smooth shadow-card hover:shadow-md hover:-translate-y-px"
    :class="{
      'border-primary/30 shadow-glow-sm': isExpanded,
      'animate-shake': shaking,
      'border-warning/30': isHidden
    }"
  >
    <!-- 管理员工具栏 -->
    <div v-if="isAdmin" class="absolute top-3 right-3 flex items-center gap-1">
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
        :to="`/review/courses/${review.courseID}`"
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
      <span class="text-text-muted">{{ formatTime(review.createdAt) }}</span>
    </div>

    <!-- 内容区：三态显示 -->
    <!-- 状态1: 屏蔽锁定（非管理员看到锁定提示） -->
    <div v-if="isHidden && !isAdmin" class="flex items-start gap-2 py-4 px-3 bg-warning/5 border border-warning/20 rounded-lg">
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

    <!-- 状态3: 正常显示（已登录 + published，或管理员查看 hidden） -->
    <template v-else>
      <!-- H-10: 使用 v-text 防御 XSS，确保用户内容不被解析为 HTML -->
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
    <div v-if="displayRatings.length > 0" class="flex flex-wrap gap-3 mt-4 pt-3 border-t border-border">
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
      :class="{ 'pt-3 border-t border-border': displayRatings.length === 0 }"
    >
      <button
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover"
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
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover"
        :class="{ '!text-primary bg-primary/[0.08]': userVote === 'dislike' }"
        :aria-label="t('review.vote.dislike')"
        :aria-pressed="userVote === 'dislike'"
        @click="handleVote('dislike')"
      >
        <ThumbsDown :size="16" />
        <span class="text-xs font-mono">{{ displayDislikes }}</span>
      </button>

      <button
        class="flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer hover:text-text-primary hover:bg-bg-hover"
        :aria-label="t('review.review.commentBtn')"
        @click="toggleExpand"
      >
        <MessageCircle :size="16" />
        <span class="text-xs font-mono">{{ replyCount }}</span>
      </button>
    </div>

    <!-- 展开区域：回复列表 + 回复输入 -->
    <div v-if="isExpanded" class="mt-4 pt-4 border-t border-border animate-fade-in-up">
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
import { Heart, ThumbsDown, MessageCircle, EyeOff, Eye, Pencil, Lock, ShieldAlert } from 'lucide-vue-next'
import type { Review } from '@/types/review'
import type { Reply } from '@/types/reply'
import { voteReview, getReplies, postReply, deleteReply } from '@/api/review'
import type { VoteType } from '@/api/review'
import { updateReviewStatus, adminEditReview } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'
import ModerationDialog from './ModerationDialog.vue'
import AdminEditDialog from './AdminEditDialog.vue'

const props = defineProps<{
  review: Review
}>()

const emit = defineEmits<{
  moderated: []
}>()

const { t, locale } = useI18n()
const toast = useToast()
const authStore = useAuthStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.user?.isAdmin === true)
const isHidden = computed(() => props.review.status === 'hidden')
const showActions = computed(() => isAuthenticated.value && !isHidden.value)

const isExpanded = ref(false)
const shouldTruncate = computed(() => (props.review.content?.length ?? 0) > 200)

// 管理员弹窗状态
const showModerationDialog = ref(false)
const showEditDialog = ref(false)

// 投票状态
const userVote = ref<VoteType | null>(null)
const isVoting = ref(false)
const likeOffset = ref(0)
const dislikeOffset = ref(0)
const likeBounce = ref(false)
const shaking = ref(false)

// M-69: 评分值钳位，防止负值导致进度条异常
const displayLikes = computed(() => Math.max(0, props.review.likeCount + likeOffset.value))
const displayDislikes = computed(() => Math.max(0, props.review.dislikeCount + dislikeOffset.value))

// M-102: 表情评分指标映射，通过 i18n 获取 emoji（支持本地化）
const displayRatings = computed(() => {
  const ratings = props.review.ratings
  if (!ratings || Object.keys(ratings).length === 0) return []
  return Object.entries(ratings).map(([key, value]) => {
    const emoji = t(`review.ratingEmoji.icon.${key}`, '⭐')
    const label = t(`review.ratingEmoji.${key}`, t('review.ratingEmoji.fallback'))
    // M-69: 钳位评分值，防止负值或超范围值
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
let voteVersion = 0
watch(() => props.review, () => {
  voteVersion++
  likeOffset.value = 0
  dislikeOffset.value = 0
  userVote.value = null
  replyCountDirty = false
})

// L-32: 定时器清理后置空，释放闭包引用
let bounceTimer: ReturnType<typeof setTimeout> | null = null
let shakeTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => {
  if (bounceTimer) { clearTimeout(bounceTimer); bounceTimer = null }
  if (shakeTimer) { clearTimeout(shakeTimer); shakeTimer = null }
})

// 时间格式化
const formatTime = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)

// 投票
async function handleVote(type: VoteType) {
  if (isVoting.value) return
  isVoting.value = true

  const prevVote = userVote.value
  const prevLikeOffset = likeOffset.value
  const prevDislikeOffset = dislikeOffset.value
  // M-103: 使用 review.id 判断是否应回滚，而非 voteVersion
  const targetReviewId = props.review.id

  // 乐观更新
  if (userVote.value === type) {
    userVote.value = null
    if (type === 'like') likeOffset.value--
    else dislikeOffset.value--
  } else {
    if (userVote.value) {
      if (userVote.value === 'like') likeOffset.value--
      else dislikeOffset.value--
    }
    userVote.value = type
    if (type === 'like') {
      likeOffset.value++
      likeBounce.value = true
      bounceTimer = setTimeout(() => { likeBounce.value = false; bounceTimer = null }, 300)
    } else {
      dislikeOffset.value++
    }
  }

  try {
    await voteReview(props.review.id, type)
  } catch {
    // M-103: review.id 变化后跳过回滚，确保只回滚同一条评论的投票
    if (props.review.id === targetReviewId) {
      userVote.value = prevVote
      likeOffset.value = prevLikeOffset
      dislikeOffset.value = prevDislikeOffset
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
    const res = await getReplies(props.review.id)
    replies.value = res.data?.list || []
    replyCount.value = res.data?.total || 0
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
    const res = await postReply({
      reviewID: props.review.id,
      content
    })
    if (res.data) {
      replies.value.push(res.data)
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
    await deleteReply(id)
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

// 管理员屏蔽
async function handleModerate(reason: string) {
  showModerationDialog.value = false
  try {
    await updateReviewStatus(props.review.id, 'hide', reason)
    toast.success(t('review.admin.moderateSuccess'))
    emit('moderated')
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

// 管理员恢复
async function handleRestore() {
  try {
    await updateReviewStatus(props.review.id, 'restore')
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
    await adminEditReview(props.review.id, payload)
    toast.success(t('review.admin.editSuccess'))
    emit('moderated')
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}
</script>
