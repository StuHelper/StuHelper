<template>
  <div class="feed-card" :class="{ expanded: isExpanded, 'animate-shake': shaking }">
    <!-- 头部：课程链接 + 时间 -->
    <div class="feed-card__header">
      <router-link :to="`/review/courses/${review.courseID}`" class="feed-card__course">
        {{ review.courseName || `Course #${review.courseID}` }}
      </router-link>
      <span v-if="review.teacherName" class="feed-card__teacher">
        {{ review.teacherName }}
      </span>
      <span class="feed-card__time">{{ formatTime(review.createdAt) }}</span>
    </div>

    <!-- 标题 -->
    <h3 v-if="review.title" class="feed-card__title">{{ review.title }}</h3>

    <!-- 内容 -->
    <div
      class="feed-card__content"
      :class="{ truncated: !isExpanded && shouldTruncate }"
      @click="toggleExpand"
    >
      {{ review.content }}
    </div>

    <!-- 评分胶囊 -->
    <div v-if="avgRating > 0" class="feed-card__ratings">
      <span class="rating-pill font-mono" :style="{ color: ratingColor }">
        {{ avgRating.toFixed(1) }}
      </span>
    </div>

    <!-- 操作栏 -->
    <div class="feed-card__actions">
      <button
        class="action-btn"
        :class="{ active: userVote === 'like' }"
        @click="handleVote('like')"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" :stroke="userVote === 'like' ? 'var(--brand-primary)' : 'currentColor'" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M7 10v12" /><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z" />
        </svg>
        <span class="action-count font-mono" :class="{ bouncing: likeBounce }">
          {{ displayLikes }}
        </span>
      </button>

      <button
        class="action-btn"
        :class="{ active: userVote === 'dislike' }"
        @click="handleVote('dislike')"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" :stroke="userVote === 'dislike' ? 'var(--accent)' : 'currentColor'" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M17 14V2" /><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z" />
        </svg>
        <span class="action-count font-mono">{{ displayDislikes }}</span>
      </button>

      <button class="action-btn" @click="toggleExpand">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
        <span class="action-count font-mono">{{ replyCount }}</span>
      </button>
    </div>

    <!-- 展开区域：回复列表 + 回复输入 -->
    <div v-if="isExpanded" class="feed-card__expanded">
      <div v-if="repliesLoading" class="replies-loading">
        <div class="spinner" />
      </div>
      <div v-else-if="replies.length > 0" class="replies-list">
        <ReplyCard
          v-for="reply in replies"
          :key="reply.id"
          :reply="reply"
          @delete="handleDeleteReply"
        />
      </div>
      <div v-else class="replies-empty">
        {{ t('review.noReplies') }}
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
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
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
const likeOffset = ref(0)
const dislikeOffset = ref(0)
const likeBounce = ref(false)
const shaking = ref(false)

const displayLikes = computed(() => props.review.likeCount + likeOffset.value)
const displayDislikes = computed(() => props.review.dislikeCount + dislikeOffset.value)

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
const replySubmitting = ref(false)
const replyCount = ref(props.review.replyCount ?? 0)
const replyFormRef = ref<InstanceType<typeof ReplyForm> | null>(null)

// 时间格式化
const formatTime = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)

// 投票
async function handleVote(type: VoteType) {
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
      setTimeout(() => { likeBounce.value = false }, 300)
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
    setTimeout(() => { shaking.value = false }, 300)
    toast.error(t('review.voteFailed'))
  }
}

// 展开/收起
async function toggleExpand() {
  isExpanded.value = !isExpanded.value
  if (isExpanded.value && replies.value.length === 0) {
    await loadReplies()
  }
}

async function loadReplies() {
  repliesLoading.value = true
  try {
    const res = await getReplies(props.review.id)
    replies.value = res.data?.list || []
    replyCount.value = res.data?.total || 0
  } catch {
    replies.value = []
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
    replies.value.push(res.data)
    replyCount.value++
    replyFormRef.value?.clear()
    toast.success(t('review.replySuccess'))
  } catch {
    toast.error(t('review.replyFailed'))
  } finally {
    replySubmitting.value = false
  }
}

async function handleDeleteReply(id: number) {
  try {
    await deleteReply(id)
    replies.value = replies.value.filter((r) => r.id !== id)
    replyCount.value--
  } catch {
    toast.error(t('review.deleteFailed'))
  }
}
</script>

<style scoped>
.feed-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  transition: border-color var(--duration-base) var(--ease-smooth);
}

.feed-card:hover {
  border-color: var(--border-light);
}

.feed-card.expanded {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.feed-card__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
  flex-wrap: wrap;
}

.feed-card__course {
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--brand-primary);
  text-decoration: none;
}

.feed-card__course:hover {
  text-decoration: underline;
}

.feed-card__teacher {
  font-size: var(--text-xs);
  color: var(--text-muted);
  padding: 2px 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
}

.feed-card__time {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-left: auto;
}

.feed-card__title {
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-2);
  letter-spacing: var(--tracking-tight);
}

.feed-card__content {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  line-height: var(--leading-relaxed);
  cursor: pointer;
  word-break: break-word;
}

.feed-card__content.truncated {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.feed-card__ratings {
  margin-top: var(--space-3);
}

.rating-pill {
  font-size: var(--text-sm);
  font-weight: var(--weight-bold);
  padding: 2px 10px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
}

.feed-card__actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--border);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: var(--space-1-5);
  color: var(--text-muted);
  font-size: var(--text-sm);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-full);
  transition: all var(--duration-fast) var(--ease-smooth);
  cursor: pointer;
}

.action-btn:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.action-btn.active {
  color: var(--brand-primary);
}

.action-count {
  font-size: var(--text-xs);
}

.action-count.bouncing {
  animation: voteBounce var(--duration-slow) var(--ease-spring);
}

.feed-card__expanded {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
  animation: fadeInUp var(--duration-base) var(--ease-out);
}

.replies-loading {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--brand-primary);
  border-radius: var(--radius-full);
  animation: spin 0.6s linear infinite;
}

.replies-empty {
  text-align: center;
  color: var(--text-muted);
  font-size: var(--text-sm);
  padding: var(--space-4);
}

.replies-list {
  margin-bottom: var(--space-2);
}
</style>
