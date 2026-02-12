<template>
  <div
    class="bg-bg-card border border-border rounded-xl p-5 transition-all duration-200 ease-smooth shadow-card hover:shadow-md hover:-translate-y-px"
    :class="{
      'border-primary/30 shadow-glow-sm': isExpanded,
      'animate-shake': shaking
    }"
  >
    <!-- 头部：课程名 + 评分徽章 -->
    <div class="flex items-center gap-2 mb-2">
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

    <!-- 内容 -->
    <div
      class="text-sm text-text-secondary leading-relaxed cursor-pointer break-words"
      :class="{ 'line-clamp-3': !isExpanded && shouldTruncate }"
      role="button"
      tabindex="0"
      :aria-label="t('review.review.expandContent')"
      @click="toggleExpand"
      @keydown.enter="toggleExpand"
      @keydown.space.prevent="toggleExpand"
    >
      {{ review.content }}
    </div>

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

    <!-- 操作栏 -->
    <div class="flex items-center gap-1 mt-3" :class="{ 'pt-3 border-t border-border': displayRatings.length === 0 }">
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Heart, ThumbsDown, MessageCircle } from 'lucide-vue-next'
import type { Review } from '@/types/review'
import type { Reply } from '@/types/reply'
import { voteReview, getReplies, postReply, deleteReply } from '@/api/review'
import type { VoteType } from '@/api/review'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'

const props = defineProps<{
  review: Review
}>()

const { t, locale } = useI18n()
const toast = useToast()

const isExpanded = ref(false)
const shouldTruncate = computed(() => props.review.content.length > 200)

// 投票状态
const userVote = ref<VoteType | null>(null)
const isVoting = ref(false)
const likeOffset = ref(0)
const dislikeOffset = ref(0)
const likeBounce = ref(false)
const shaking = ref(false)

const displayLikes = computed(() => props.review.likeCount + likeOffset.value)
const displayDislikes = computed(() => props.review.dislikeCount + dislikeOffset.value)

// 表情评分指标映射
const ratingEmojiMap: Record<string, { emoji: string; label: string }> = {
  recommendation: { emoji: '👍', label: '推荐度' },
  content_quality: { emoji: '📚', label: '内容' },
  workload: { emoji: '⏰', label: '工作量' },
  grading: { emoji: '📊', label: '给分' }
}

const displayRatings = computed(() => {
  const ratings = props.review.ratings
  if (!ratings || Object.keys(ratings).length === 0) return []
  return Object.entries(ratings).map(([key, value]) => {
    const mapped = ratingEmojiMap[key] || { emoji: '⭐', label: key }
    return { key, emoji: mapped.emoji, label: mapped.label, value }
  })
})

// 评分
const avgRating = computed(() => {
  const ratings = Object.values(props.review.ratings)
  if (ratings.length === 0) return 0
  return ratings.reduce((a, b) => a + b, 0) / ratings.length
})

const ratingColor = computed(() => {
  if (avgRating.value >= 4) return 'var(--rating-5)'
  if (avgRating.value >= 3) return 'var(--rating-4)'
  if (avgRating.value >= 2) return 'var(--rating-3)'
  return 'var(--rating-1)'
})

// 回复
const replies = ref<Reply[]>([])
const repliesLoading = ref(false)
const repliesError = ref(false)
const replySubmitting = ref(false)
const replyCount = ref(props.review.replyCount ?? 0)
const replyFormRef = ref<InstanceType<typeof ReplyForm> | null>(null)

// 同步 props 变化
watch(() => props.review.replyCount, (val) => {
  if (val !== undefined) replyCount.value = val
})

// 当 review 数据刷新时重置投票偏移量
watch(() => props.review.id, () => {
  likeOffset.value = 0
  dislikeOffset.value = 0
  userVote.value = null
})

// 定时器清理
let bounceTimer: ReturnType<typeof setTimeout> | null = null
let shakeTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => {
  if (bounceTimer) clearTimeout(bounceTimer)
  if (shakeTimer) clearTimeout(shakeTimer)
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
      bounceTimer = setTimeout(() => { likeBounce.value = false }, 300)
    } else {
      dislikeOffset.value++
    }
  }

  try {
    await voteReview(props.review.id, type)
  } catch {
    // 回滚
    userVote.value = prevVote
    likeOffset.value = prevLikeOffset
    dislikeOffset.value = prevDislikeOffset
    shaking.value = true
    shakeTimer = setTimeout(() => { shaking.value = false }, 300)
    toast.error(t('review.review.voteFailed'))
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
  } catch {
    toast.error(t('review.review.deleteFailed'))
  }
}
</script>
