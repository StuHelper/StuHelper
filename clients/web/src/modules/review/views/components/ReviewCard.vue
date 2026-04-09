<template>
  <div
    class="relative bg-bg-card rounded-xl shadow-card p-5 transition-all duration-slow hover:-translate-y-0.5 hover:shadow-lg"
    :class="reviewCardBorderClass(review)"
  >
    <!-- Admin toolbar -->
    <div v-if="canManageReviews" class="absolute top-3 right-3 flex items-center gap-1 z-10">
      <button
        v-if="review.status !== 'hidden'"
        class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
        :title="t('review.admin.hide')"
        @click="$emit('moderate', review)"
      >
        <EyeOff :size="16" />
      </button>
      <template v-else>
        <button
          class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
          :title="t('review.admin.restore')"
          @click="$emit('restore', review)"
        >
          <Eye :size="16" />
        </button>
        <button
          class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
          :title="t('review.admin.edit')"
          @click="$emit('edit', review)"
        >
          <Pencil :size="16" />
        </button>
      </template>
    </div>

    <!-- Title row -->
    <div class="flex items-center gap-2 mb-1" :class="canManageReviews ? 'pr-20' : ''">
      <span class="font-bold text-base text-text-primary">
        {{ review.title || courseName }}
      </span>
    </div>

    <!-- Teacher + Semester + Date -->
    <div class="flex items-center gap-1.5 mb-2 text-xs text-text-muted">
      <span v-if="review.teacherName" class="font-medium text-secondary">
        {{ review.teacherName }}
      </span>
      <span v-if="review.teacherName && review.termName">&middot;</span>
      <span v-if="review.termName">{{ review.termName }}</span>
      <span>&middot;</span>
      <span>{{ formatRelativeTime(review.createdAt, locale, t) }}</span>
    </div>

    <!-- Content -->
    <div
      class="text-sm leading-relaxed break-words whitespace-pre-line mb-2 text-text-secondary"
      v-text="review.content"
    />

    <!-- Grade -->
    <div v-if="review.grade" class="mb-2">
      <span class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-success/15 text-success">
        {{ t('review.post.gradeLabel') }}: {{ review.grade }}
      </span>
    </div>

    <!-- Controversial badge -->
    <ControversialBadge :dislike-count="review.dislikeCount" />

    <!-- Divider -->
    <hr class="border-0 border-t border-border-light my-3" />

    <!-- Bottom: Emoji ratings row -->
    <div class="flex flex-wrap items-center gap-4 mb-3">
      <div
        v-for="(value, key) in review.ratings"
        :key="key"
        class="flex items-center gap-1.5"
      >
        <EmojiRating :value="value" :show-value="false" size="sm" />
        <span class="text-xs text-text-muted">
          {{ dimensionLabel(String(key), t) }}
        </span>
      </div>
    </div>

    <!-- Like / Dislike buttons -->
    <div class="flex items-center gap-2">
      <button
        :data-testid="`review-like-${review.id}`"
        class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-up hover:bg-vote-up/10"
        :class="userVote === 'like' ? '!text-vote-up-active !bg-vote-up/12' : ''"
        @click="$emit('vote', review, 'like')"
      >
        <ThumbsUp :size="16" />
        <span class="text-xs font-mono">{{ displayLikeCount }}</span>
      </button>
      <button
        :data-testid="`review-dislike-${review.id}`"
        class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-down hover:bg-vote-down/10"
        :class="userVote === 'dislike' ? '!text-vote-down-active !bg-vote-down/12' : ''"
        @click="$emit('vote', review, 'dislike')"
      >
        <ThumbsDown :size="16" />
        <span class="text-xs font-mono">{{ displayDislikeCount }}</span>
      </button>
      <button
        class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
        @click="$emit('toggle-expand', review.id)"
      >
        <MessageCircle :size="16" />
        <span class="text-xs font-mono">{{ replyCount }}</span>
      </button>
    </div>

    <!-- Expanded reply area -->
    <div v-if="expanded" class="mt-4 pt-4 border-t border-border-light animate-fade-in">
      <div v-if="repliesLoading" class="flex justify-center py-4">
        <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin" />
      </div>
      <div v-else-if="repliesError" class="text-center text-sm py-4 text-danger">
        {{ t('review.review.replyLoadFailed') }}
        <button class="text-sm cursor-pointer underline ml-2 text-primary" @click="$emit('retry-replies', review.id)">
          {{ t('common.actions.retry') }}
        </button>
      </div>
      <div v-else-if="replies.length > 0" class="mb-2">
        <ReplyCard
          v-for="reply in replies"
          :key="reply.id"
          :reply="reply"
          @delete="$emit('delete-reply', $event)"
        />
      </div>
      <div v-else class="text-center text-sm py-4 text-text-muted">
        {{ t('review.review.noReplies') }}
      </div>

      <ReplyForm
        ref="replyFormRef"
        :submitting="replySubmitting"
        @submit="(content: string) => $emit('reply-submit', review.id, content)"
        @cancel="$emit('reply-cancel')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  ThumbsUp,
  ThumbsDown,
  MessageCircle,
  EyeOff,
  Eye,
  Pencil,
} from 'lucide-vue-next'

import EmojiRating from '@/components/business/review/EmojiRating.vue'
import ControversialBadge from '@/components/business/review/ControversialBadge.vue'
import ReplyCard from '@/components/business/review/ReplyCard.vue'
import ReplyForm from '@/components/business/review/ReplyForm.vue'
import { reviewCardBorderClass, dimensionLabel } from '@/modules/review/ratingHelpers'
import { formatRelativeTime } from '@/utils/date'

import type { Review } from '@/types/review'
import type { Reply } from '@/types/reply'
import type { VoteType } from '@/components/business/review/reviewVoteState'

defineProps<{
  review: Review
  courseName: string
  canManageReviews: boolean
  userVote: VoteType | null
  displayLikeCount: number
  displayDislikeCount: number
  replyCount: number
  expanded: boolean
  replies: Reply[]
  repliesLoading: boolean
  repliesError: boolean
  replySubmitting: boolean
}>()

defineEmits<{
  vote: [review: Review, type: VoteType]
  'toggle-expand': [reviewId: string]
  'retry-replies': [reviewId: string]
  'reply-submit': [reviewId: string, content: string]
  'reply-cancel': []
  'delete-reply': [id: string]
  moderate: [review: Review]
  restore: [review: Review]
  edit: [review: Review]
}>()

const { t, locale } = useI18n()
</script>
