<template>
  <div class="latest-reviews-page">
    <header class="page-header">
      <h1 class="page-title">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <polyline points="12,6 12,12 16,14"/>
        </svg>
        最新测评
      </h1>
      <p class="page-desc">查看社区最新发布的课程测评</p>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="loading-list">
      <div v-for="i in 5" :key="i" class="skeleton-review">
        <div class="skeleton-header"></div>
        <div class="skeleton-content"></div>
        <div class="skeleton-footer"></div>
      </div>
    </div>

    <!-- Review List -->
    <div v-else-if="reviews.length" class="review-list">
      <ReviewCard
        v-for="r in reviews"
        :key="r.id"
        :review="r"
        show-course
      />
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state">
      <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <p class="empty-text">暂无测评</p>
    </div>

    <!-- Pagination -->
    <nav v-if="totalPages > 1" class="pagination">
      <button
        class="page-btn"
        :disabled="page <= 1"
        @click="handlePageChange(page - 1)"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M15 18l-6-6 6-6"/>
        </svg>
      </button>
      <span class="page-info">{{ page }} / {{ totalPages }}</span>
      <button
        class="page-btn"
        :disabled="page >= totalPages"
        @click="handlePageChange(page + 1)"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 18l6-6-6-6"/>
        </svg>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import { getLatestReviews } from '@/api/review'
import type { Review } from '@/types/review'

const loading = ref(false)
const reviews = ref<Review[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)

const totalPages = computed(() => Math.ceil(total.value / pageSize))

const fetchReviews = async () => {
  loading.value = true
  try {
    const res = await getLatestReviews(page.value, pageSize)
    reviews.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  fetchReviews()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

onMounted(fetchReviews)
</script>

<style scoped>
.latest-reviews-page {
  max-width: 800px;
  margin: 0 auto;
  padding: var(--space-6);
}

.page-header {
  margin-bottom: var(--space-8);
}

.page-title {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 var(--space-2) 0;
}

.page-title svg {
  width: 28px;
  height: 28px;
  color: var(--accent);
}

.page-desc {
  font-size: var(--text-base);
  color: var(--text-muted);
  margin: 0;
}

/* Loading State */
.loading-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.skeleton-review {
  padding: var(--space-5);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.skeleton-header {
  width: 60%;
  height: 20px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-content {
  width: 100%;
  height: 60px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-footer {
  width: 40%;
  height: 16px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Review List */
.review-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-16) var(--space-4);
  text-align: center;
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: var(--text-muted);
  opacity: 0.4;
  margin-bottom: var(--space-4);
}

.empty-text {
  font-size: var(--text-lg);
  color: var(--text-secondary);
  margin: 0;
}

/* Pagination */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-4);
}

.page-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  transition: all var(--duration-fast);
}

.page-btn:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-btn svg {
  width: 20px;
  height: 20px;
}

.page-info {
  font-size: var(--text-sm);
  color: var(--text-muted);
}
</style>
