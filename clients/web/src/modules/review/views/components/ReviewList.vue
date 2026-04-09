<template>
  <section>
    <div class="flex flex-col gap-4 mb-4">
      <ReviewCard
        v-for="r in filteredReviews"
        :key="r.id"
        :review="r"
        :course-name="courseName"
        :can-manage-reviews="canManageReviews"
        :user-vote="reviewVotes[r.id] ?? null"
        :display-like-count="displayLikeCount(r)"
        :display-dislike-count="displayDislikeCount(r)"
        :reply-count="replyCountMap[r.id] ?? r.replyCount"
        :expanded="expandedReviewID === r.id"
        :replies="expandedReviewID === r.id ? replies : []"
        :replies-loading="expandedReviewID === r.id && repliesLoading"
        :replies-error="expandedReviewID === r.id && repliesError"
        :reply-submitting="replySubmitting"
        @vote="handleVote"
        @toggle-expand="toggleExpand"
        @retry-replies="loadReplies"
        @reply-submit="handleReplySubmit"
        @reply-cancel="expandedReviewID = null"
        @delete-reply="handleDeleteReply"
        @moderate="$emit('moderate', $event)"
        @restore="$emit('restore', $event)"
        @edit="$emit('edit', $event)"
      />
    </div>

    <!-- Empty state -->
    <div
      v-if="!reviewsLoading && reviews.length === 0"
      class="bg-bg-card rounded-xl shadow-card p-8 text-center"
    >
      <p class="text-text-muted">{{ t('review.course.noReviews') }}</p>
      <button
        class="mt-2 py-2 px-6 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
        @click="$emit('post')"
      >
        {{ t('review.detail.writeFirst') }}
      </button>
    </div>

    <!-- Load more -->
    <div v-if="hasMore" class="flex justify-center py-6">
      <button
        class="py-2 px-6 text-sm font-medium text-white bg-primary rounded-full opacity-90 transition-opacity duration-fast hover:opacity-100 disabled:opacity-50"
        :disabled="reviewsLoading"
        @click="$emit('load-more')"
      >
        {{ reviewsLoading ? t('common.actions.loading') : t('common.actions.loadMore') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ReviewCard from './ReviewCard.vue'
import { useReviewVoting } from '@/modules/review/useReviewVoting'
import { useReviewReplies } from '@/modules/review/useReviewReplies'

import type { Review } from '@/types/review'
import type { VoteType } from '@/components/business/review/reviewVoteState'

defineProps<{
  reviews: Review[]
  filteredReviews: Review[]
  courseName: string
  canManageReviews: boolean
  reviewsLoading: boolean
  hasMore: boolean
}>()

defineEmits<{
  post: []
  'load-more': []
  moderate: [review: Review]
  restore: [review: Review]
  edit: [review: Review]
}>()

const { t } = useI18n()

const {
  reviewVotes,
  displayLikeCount,
  displayDislikeCount,
  handleVote,
} = useReviewVoting()

const {
  expandedReviewID,
  replies,
  repliesLoading,
  repliesError,
  replySubmitting,
  replyCountMap,
  toggleExpand,
  loadReplies,
  handleReplySubmit,
  handleDeleteReply,
} = useReviewReplies()
</script>
