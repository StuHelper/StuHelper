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
        {{ review.title || review.courseName || t('review.course.fallbackTitle', { id: review.courseID }) }}
      </router-link>
      <span
        v-if="avgRating > 0"
        class="inline-flex items-center justify-center py-1 px-2 rounded-full shrink-0"
        :style="{
          '--rating-clr': ratingColor,
          background: 'color-mix(in srgb, var(--rating-clr, var(--color-primary)) 12%, transparent)',
          color: 'var(--rating-clr, var(--color-primary))'
        }"
      >
        <EmojiRating :value="avgRating" size="sm" />
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
    <LockedReviewContent
      v-else-if="!isAuthenticated && !isHidden"
      :preview-line="review.content"
      :message="t('review.card.loginToView')"
      :action-label="t('review.card.loginBtn')"
      @action="handleLogin"
    />

    <!-- 状态2.5: 已登录未认证，正文锁定 -->
    <LockedReviewContent
      v-else-if="isPreviewMode"
      :preview-line="review.content"
      :message="t('review.card.verifyToView')"
      :action-label="t('review.card.goVerify')"
      @action="goVerify"
    />

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

    <!-- 评分指标 -->
    <div v-if="displayRatings.length > 0" class="flex flex-wrap gap-3 mt-4 pt-3 border-t border-border-light">
      <span
        v-for="dim in displayRatings"
        :key="dim.key"
        class="text-xs text-text-secondary flex items-center gap-1.5"
      >
        <EmojiRating :value="dim.value" size="sm" />
        <span>{{ dim.label }}</span>
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
        :class="[neutralActionButtonClass, { '!text-primary bg-primary/[0.08]': userVote === 'like' }]"
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
        :class="[neutralActionButtonClass, { '!text-primary bg-primary/[0.08]': userVote === 'dislike' }]"
        :aria-label="t('review.vote.dislike')"
        :aria-pressed="userVote === 'dislike'"
        @click="handleVote('dislike')"
      >
        <ThumbsDown :size="16" />
        <span class="text-xs font-mono">{{ displayDislikes }}</span>
      </button>

      <button
        v-ripple
        :class="neutralActionButtonClass"
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
        :class="[warningActionButtonClass, 'ml-auto']"
        :aria-label="t('review.review.reportBtn')"
        @click="toggleReportMenu"
      >
        <Flag :size="16" />
      </button>

      <!-- 编辑按钮（自己的评价） -->
      <button
        v-if="props.isOwnReview && !editing"
        v-ripple
        :class="[primaryActionButtonClass, 'ml-auto']"
        :aria-label="t('review.review.editBtn')"
        @click="startEditing"
      >
        <Pencil :size="16" />
      </button>

      <!-- 删除按钮（自己的评价） -->
      <button
        v-if="props.isOwnReview"
        v-ripple
        :class="[dangerActionButtonClass, { 'ml-auto': editing }]"
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

      <ReplyLoginPrompt
        v-if="!isAuthenticated"
        @login="handleLogin"
      />
      <ReplyForm
        v-else
        ref="replyFormRef"
        :submitting="replySubmitting"
        @submit="handleReplySubmit"
        @cancel="isExpanded = false"
      />
    </div>

    <!-- 屏蔽弹窗 -->
    <ModerationDialog
      :visible="showModerationDialog"
      :submitting="moderationSubmitting"
      @confirm="handleModerate"
      @close="showModerationDialog = false"
    />

    <!-- 编辑弹窗 -->
    <AdminEditDialog
      :visible="showEditDialog"
      :review="review"
      :submitting="editSubmitting"
      @confirm="handleAdminEdit"
      @close="showEditDialog = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Heart, ThumbsDown, MessageCircle, EyeOff, Eye, Pencil, ShieldAlert, Flag, Trash2 } from 'lucide-vue-next'
import type { Review } from '@stuhelper/shared/review'
import { canListFullReviews, canManageReviews as canManageReviewAccess } from '@/utils/adminAccess'
import { getRatingColor } from '@/design-system/rating'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { formatRelativeTime } from '@/utils/date'
import { use3DTilt } from '@/composables/use3DTilt'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'
import ReplyLoginPrompt from './ReplyLoginPrompt.vue'
import LockedReviewContent from './LockedReviewContent.vue'
import EmojiRating from './EmojiRating.vue'
import ModerationDialog from './ModerationDialog.vue'
import AdminEditDialog from './AdminEditDialog.vue'
import { useReviewVote } from './useReviewVote'
import { useReviewReply } from './useReviewReply'
import { useReviewReport } from './useReviewReport'
import { useReviewEdit } from './useReviewEdit'
import { useReviewDelete } from './useReviewDelete'
import { useReviewModeration } from './useReviewModeration'
import { ratingDimensionLabel } from '@/modules/review/ratingHelpers'
import { accountCenterURL } from '@/utils/redirect'

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
const authStore = useAuthStore()
const verificationStore = useVerificationStore()

const actionButtonClass = 'flex items-center gap-1.5 text-text-muted text-sm py-1.5 px-3 rounded-full transition-all duration-fast ease-smooth cursor-pointer press-spring'
const neutralActionButtonClass = `${actionButtonClass} hover:text-text-primary hover:bg-bg-hover`
const primaryActionButtonClass = `${actionButtonClass} hover:text-primary hover:bg-primary/10`
const warningActionButtonClass = `${actionButtonClass} hover:text-warning hover:bg-warning/10`
const dangerActionButtonClass = `${actionButtonClass} hover:text-danger hover:bg-danger/10`

const cardRef = ref<HTMLElement>()
const { style: tiltStyle } = use3DTilt(cardRef, { maxTilt: 4, scale: 1.01, speed: 500 })

const isAuthenticated = computed(() => authStore.isAuthenticated)
const canManageReviews = computed(() => canManageReviewAccess(authStore.user))
const hasFullListCapability = computed(() => canListFullReviews(authStore.user))
const isHidden = computed(() => props.review.status === 'hidden')
const showActions = computed(() => isAuthenticated.value && !isHidden.value)
const canViewFull = computed(() =>
  canManageReviews.value || (hasFullListCapability.value && verificationStore.canViewFullReviews)
)
const isPreviewMode = computed(() => isAuthenticated.value && !isHidden.value && !canViewFull.value)

const isExpanded = ref(false)
const shouldTruncate = computed(() => (props.review.content?.length ?? 0) > 200)

const displayRatings = computed(() => {
  const ratings = props.review.ratings
  if (!ratings || Object.keys(ratings).length === 0) return []
  return Object.entries(ratings).map(([key, value]) => {
    const label = ratingDimensionLabel({ key, t })
    const clampedValue = Math.max(0, Math.min(5, value))
    return { key, label, value: clampedValue }
  })
})

const avgRating = computed(() => {
  const ratings = Object.values(props.review.ratings || {})
  if (ratings.length === 0) return 0
  return ratings.reduce((a, b) => a + b, 0) / ratings.length
})
const ratingColor = computed(() => getRatingColor(avgRating.value))
const formattedTime = computed(() => formatRelativeTime(props.review.createdAt, locale.value, t))

function goVerify() {
  window.location.assign(accountCenterURL('/user/student-verification'))
}

const {
  userVote,
  likeBounce,
  shaking,
  displayLikes,
  displayDislikes,
  handleVote,
} = useReviewVote(() => props.review, t)

const {
  replies,
  repliesLoading,
  repliesError,
  replySubmitting,
  replyCount,
  replyFormRef,
  loadReplies,
  handleReplySubmit,
  handleDeleteReply,
} = useReviewReply(() => props.review, t)

const {
  showReportMenu,
  reporting,
  reportReasons,
  toggleReportMenu,
  handleReport,
} = useReviewReport(() => props.review.id, t)

const {
  editing,
  editContent,
  saving,
  startEditing,
  cancelEditing,
  handleSaveEdit,
} = useReviewEdit(() => props.review, t, (id, content) => emit('updated', id, content))

const { deleting, handleDeleteOwn } = useReviewDelete(() => props.review, t, (id) => emit('deleted', id))

const {
  showModerationDialog,
  showEditDialog,
  moderationSubmitting,
  editSubmitting,
  handleModerate,
  handleRestore,
  handleAdminEdit,
} = useReviewModeration(() => props.review, t, () => emit('moderated'))

async function toggleExpand() {
  isExpanded.value = !isExpanded.value
  if (isExpanded.value) {
    await loadReplies()
  }
}

function handleLogin() {
  authStore.login()
}
</script>
