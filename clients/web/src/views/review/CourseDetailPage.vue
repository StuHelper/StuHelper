<template>
  <div class="course-detail-page">
    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="skeleton-header">
        <div class="skeleton-title"></div>
        <div class="skeleton-meta"></div>
      </div>
      <div class="skeleton-chart"></div>
    </div>

    <!-- Content -->
    <template v-else-if="course">
      <!-- Course Header -->
      <header class="course-header">
        <div class="course-info">
          <h1 class="course-name">{{ course.name }}</h1>
          <div class="course-meta">
            <span v-if="course.teacherName" class="teacher">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              {{ course.teacherName }}
            </span>
            <span class="department">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 21h18"/>
                <path d="M5 21V7l8-4v18"/>
                <path d="M19 21V11l-6-4"/>
              </svg>
              {{ course.departmentName }}
            </span>
            <span v-if="course.credits" class="credits">
              {{ course.credits }} 学分
            </span>
          </div>
        </div>
        <button class="post-btn" @click="handlePostClick">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          发布测评
        </button>
      </header>

      <!-- Rating Chart -->
      <section class="rating-section">
        <CourseRatingChart :course-id="courseId" />
      </section>

      <!-- Reviews Section -->
      <section class="reviews-section">
        <div class="section-header">
          <h2 class="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
            课程测评
            <span class="review-count">({{ total }})</span>
          </h2>
        </div>

        <!-- Review List -->
        <div v-if="reviews.length" class="review-list">
          <ReviewCard v-for="r in reviews" :key="r.id" :review="r" />
        </div>

        <!-- Empty State -->
        <div v-else class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p class="empty-text">暂无测评</p>
          <p class="empty-hint">成为第一个评价者吧</p>
        </div>

        <!-- Pagination -->
        <nav v-if="totalPages > 1" class="pagination">
          <button class="page-btn" :disabled="page <= 1" @click="handlePageChange(page - 1)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 18l-6-6 6-6"/>
            </svg>
          </button>
          <span class="page-info">{{ page }} / {{ totalPages }}</span>
          <button class="page-btn" :disabled="page >= totalPages" @click="handlePageChange(page + 1)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 18l6-6-6-6"/>
            </svg>
          </button>
        </nav>
      </section>
    </template>

    <!-- Post Review Dialog -->
    <PostReviewDialog
      v-model="showPostDialog"
      :course-id="courseId"
      @success="handlePostSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ReviewCard from '@/components/review/ReviewCard.vue'
import PostReviewDialog from '@/components/review/PostReviewDialog.vue'
import CourseRatingChart from '@/components/review/CourseRatingChart.vue'
import { getCourse } from '@/api/course'
import { getCourseReviews } from '@/api/review'
import type { Course } from '@/types/course'
import type { Review } from '@/types/review'

const route = useRoute()
const router = useRouter()
const courseId = Number(route.params.id)

// 验证 courseId 是否有效，无效则重定向
if (isNaN(courseId) || courseId <= 0) {
  router.replace({ name: 'CourseList' })
}

const loading = ref(false)
const course = ref<Course | null>(null)
const reviews = ref<Review[]>([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const showPostDialog = ref(false)

const totalPages = computed(() => Math.ceil(total.value / pageSize))

const fetchCourse = async () => {
  try {
    const res = await getCourse(courseId)
    course.value = res.data
  } catch (err) {
    console.error('Failed to fetch course:', err)
  }
}

const fetchReviews = async () => {
  try {
    const res = await getCourseReviews(courseId, page.value, pageSize)
    reviews.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch (err) {
    console.error('Failed to fetch reviews:', err)
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  fetchReviews()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

const handlePostClick = () => {
  showPostDialog.value = true
}

const handlePostSuccess = () => {
  showPostDialog.value = false
  fetchReviews()
  fetchCourse()
}

onMounted(async () => {
  if (isNaN(courseId) || courseId <= 0) {
    router.replace({ name: 'courses' })
    return
  }

  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchReviews()])
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.course-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-6);
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.skeleton-header {
  padding: var(--space-6);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.skeleton-title {
  width: 60%;
  height: 28px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-meta {
  width: 40%;
  height: 18px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-chart {
  height: 300px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Course Header */
.course-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--space-4);
  padding: var(--space-6);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-6);
}

.course-name {
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 var(--space-3) 0;
}

.course-meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-4);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.course-meta span {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.course-meta svg {
  width: 16px;
  height: 16px;
}

.teacher {
  color: var(--accent);
}

.post-btn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5);
  background: var(--accent);
  color: var(--bg-primary);
  font-weight: 600;
  border-radius: var(--radius-md);
  transition: all var(--duration-fast);
  flex-shrink: 0;
}

.post-btn:hover {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.post-btn svg {
  width: 18px;
  height: 18px;
}

/* Rating Section */
.rating-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-6);
}

/* Reviews Section */
.reviews-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
}

.section-header {
  margin-bottom: var(--space-5);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-display);
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.section-title svg {
  width: 20px;
  height: 20px;
  color: var(--accent);
}

.review-count {
  color: var(--text-muted);
  font-weight: 400;
}

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
  padding: var(--space-12) var(--space-4);
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
  margin: 0 0 var(--space-2) 0;
}

.empty-hint {
  font-size: var(--text-sm);
  color: var(--text-muted);
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
  background: var(--bg-elevated);
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

/* Responsive */
@media (max-width: 640px) {
  .course-header {
    flex-direction: column;
  }

  .post-btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
