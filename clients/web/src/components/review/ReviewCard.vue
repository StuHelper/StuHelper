<template>
  <article class="review-card" :class="{ 'show-course': showCourse }">
    <!-- Header -->
    <header class="card-header">
      <div class="header-left">
        <h3 v-if="review.title" class="title">{{ review.title }}</h3>
        <div class="meta">
          <router-link
            v-if="showCourse && review.courseName"
            :to="`/review/courses/${review.courseId}`"
            class="course-link"
          >
            {{ review.courseName }}
          </router-link>
          <span v-if="review.teacherName" class="teacher">
            {{ review.teacherName }} 老师
          </span>
          <span v-if="review.termName" class="term">{{ review.termName }}</span>
        </div>
      </div>
      <time class="date">{{ formatDate(review.createdAt) }}</time>
    </header>

    <!-- Content -->
    <div class="card-content">
      <p class="content-text">{{ review.content }}</p>
    </div>

    <!-- Ratings -->
    <div v-if="hasRatings" class="ratings-row">
      <div v-for="(value, key) in review.ratings" :key="key" class="rating-tag">
        <span class="rating-name">{{ getDimensionName(key) }}</span>
        <span class="rating-value" :style="{ color: getRatingColor(value) }">
          {{ value }}
        </span>
      </div>
    </div>

    <!-- Footer -->
    <footer class="card-footer">
      <div class="actions">
        <button
          class="action-btn like-btn"
          :class="{ loading: voting }"
          :disabled="voting"
          @click="handleLike"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3zM7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3"/>
          </svg>
          <span>{{ review.likeCount }}</span>
        </button>
      </div>
      <span v-if="review.grade" class="grade">成绩: {{ review.grade }}</span>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Review } from '@/types/review'
import { formatDate } from '@/utils/date'
import { voteReview } from '@/api/review'

const props = withDefaults(defineProps<{
  review: Review
  showCourse?: boolean
}>(), {
  showCourse: false
})

const emit = defineEmits<{ voted: [] }>()

const voting = ref(false)

const hasRatings = computed(() => {
  return props.review.ratings && Object.keys(props.review.ratings).length > 0
})

const dimensionNames: Record<string, string> = {
  overall: '总体',
  content: '内容',
  workload: '工作量',
  grading: '给分',
  attendance: '考勤'
}

const getDimensionName = (key: string) => {
  return dimensionNames[key] || key
}

const getRatingColor = (value: number) => {
  const colors: Record<number, string> = {
    1: 'var(--rating-1)',
    2: 'var(--rating-2)',
    3: 'var(--rating-3)',
    4: 'var(--rating-4)',
    5: 'var(--rating-5)'
  }
  return colors[value] || 'var(--text-secondary)'
}

const handleLike = async () => {
  const reviewId = Number(props.review.id)
  if (isNaN(reviewId)) return

  voting.value = true
  try {
    await voteReview(Number(props.review.id), 'like')
    emit('voted')
  } catch (e) {
    console.error('Vote failed:', e)
  } finally {
    voting.value = false
  }
}
</script>

<style scoped>
.review-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  transition: all var(--duration-base) var(--ease-out);
}

.review-card:hover {
  border-color: var(--border-light);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.title {
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 var(--space-2) 0;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.course-link {
  color: var(--accent);
  font-weight: 500;
}

.date {
  font-size: var(--text-xs);
  color: var(--text-muted);
  white-space: nowrap;
}

.content-text {
  color: var(--text-secondary);
  line-height: var(--leading-relaxed);
  margin: 0;
}

.ratings-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.rating-tag {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}

.rating-name {
  color: var(--text-muted);
}

.rating-value {
  font-weight: 600;
}

.card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  transition: all var(--duration-fast);
}

.action-btn svg {
  width: 18px;
  height: 18px;
}

.action-btn:hover {
  color: var(--accent);
  background: var(--bg-hover);
}

.action-btn.loading {
  opacity: 0.6;
  pointer-events: none;
}

.grade {
  font-size: var(--text-sm);
  color: var(--text-muted);
}
</style>
