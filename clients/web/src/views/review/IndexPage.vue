<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="hero-section">
      <div class="hero-bg">
        <div class="hero-glow"></div>
      </div>
      <div class="hero-content">
        <h1 class="hero-title">
          <span class="title-accent">课程测评</span>社区
        </h1>
        <p class="hero-subtitle">分享你的课程体验，帮助同学选课</p>
        <div class="hero-search">
          <SearchBar placeholder="搜索课程、教师..." @select="handleCourseSelect" />
        </div>
      </div>
    </section>

    <!-- Stats Section -->
    <section class="stats-section">
      <div class="stats-grid">
        <div class="stat-card" v-for="stat in statItems" :key="stat.key">
          <div class="stat-icon">
            <component :is="stat.icon" />
          </div>
          <div class="stat-info">
            <span class="stat-value">{{ stats[stat.key] }}</span>
            <span class="stat-label">{{ stat.label }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Quick Actions -->
    <section class="actions-section">
      <router-link to="/review/courses" class="action-card primary">
        <div class="action-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
          </svg>
        </div>
        <div class="action-content">
          <span class="action-title">浏览全部课程</span>
          <span class="action-desc">按院系分类查看所有课程</span>
        </div>
        <svg class="action-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M5 12h14M12 5l7 7-7 7"/>
        </svg>
      </router-link>

      <router-link to="/review/latest" class="action-card">
        <div class="action-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12,6 12,12 16,14"/>
          </svg>
        </div>
        <div class="action-content">
          <span class="action-title">查看最新测评</span>
          <span class="action-desc">浏览同学们的最新评价</span>
        </div>
        <svg class="action-arrow" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M5 12h14M12 5l7 7-7 7"/>
        </svg>
      </router-link>
    </section>

    <!-- Recent Reviews -->
    <section v-if="randomReviews.length" class="reviews-section">
      <div class="section-header">
        <h2 class="section-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          最新测评
        </h2>
        <router-link to="/review/latest" class="view-all">
          查看全部
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M5 12h14M12 5l7 7-7 7"/>
          </svg>
        </router-link>
      </div>
      <div class="reviews-grid">
        <ReviewCard
          v-for="review in randomReviews"
          :key="review.id"
          :review="review"
          show-course
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import SearchBar from '@/components/review/SearchBar.vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import { getLatestReviews } from '@/api/review'
import { getStats } from '@/api/course'
import type { Course } from '@/types/course'
import type { Review } from '@/types/review'

const router = useRouter()

// 统计数据图标组件
const BookIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
  h('path', { d: 'M4 19.5A2.5 2.5 0 0 1 6.5 17H20' }),
  h('path', { d: 'M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z' })
])

const MessageIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
  h('path', { d: 'M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z' })
])

const BuildingIcon = () => h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
  h('path', { d: 'M3 21h18' }),
  h('path', { d: 'M5 21V7l8-4v18' }),
  h('path', { d: 'M19 21V11l-6-4' }),
  h('path', { d: 'M9 9v.01' }),
  h('path', { d: 'M9 12v.01' }),
  h('path', { d: 'M9 15v.01' }),
  h('path', { d: 'M9 18v.01' })
])

const statItems = [
  { key: 'courseCount', label: '课程数量', icon: BookIcon },
  { key: 'reviewCount', label: '测评数量', icon: MessageIcon },
  { key: 'departmentCount', label: '院系数量', icon: BuildingIcon }
]

const stats = ref<Record<string, number>>({
  courseCount: 0,
  reviewCount: 0,
  departmentCount: 0
})

const randomReviews = ref<Review[]>([])

const handleCourseSelect = (course: Course) => {
  router.push(`/review/courses/${course.id}`)
}

onMounted(async () => {
  document.title = '课程测评社区 - StuHelper'
  try {
    const [reviewsRes, statsRes] = await Promise.all([
      getLatestReviews(1, 3),
      getStats()
    ])
    randomReviews.value = reviewsRes.data?.list || []
    if (statsRes.data) {
      stats.value = statsRes.data
    }
  } catch (e) {
    console.error('Failed to load data:', e)
  }
})
</script>

<style scoped>
.home-page {
  min-height: 100vh;
}

/* Hero Section */
.hero-section {
  position: relative;
  padding: var(--space-16) var(--space-4);
  text-align: center;
  overflow: hidden;
}

.hero-bg {
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%);
}

.hero-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 600px;
  height: 600px;
  background: radial-gradient(circle, rgba(201, 162, 39, 0.15) 0%, transparent 70%);
  pointer-events: none;
}

.hero-content {
  position: relative;
  z-index: 1;
  max-width: 600px;
  margin: 0 auto;
}

.hero-title {
  font-family: var(--font-display);
  font-size: clamp(2rem, 5vw, 3rem);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 var(--space-3) 0;
  letter-spacing: -0.02em;
}

.title-accent {
  color: var(--accent);
}

.hero-subtitle {
  font-size: var(--text-lg);
  color: var(--text-secondary);
  margin: 0 0 var(--space-8) 0;
}

.hero-search {
  max-width: 480px;
  margin: 0 auto;
}

/* Stats Section */
.stats-section {
  padding: var(--space-8) var(--space-4);
  max-width: 1000px;
  margin: 0 auto;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-4);
}

.stat-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  transition: all var(--duration-base) var(--ease-out);
}

.stat-card:hover {
  border-color: var(--border-accent);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(201, 162, 39, 0.1);
  border-radius: var(--radius-md);
  color: var(--accent);
}

.stat-icon svg {
  width: 24px;
  height: 24px;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--text-primary);
}

.stat-label {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

/* Actions Section */
.actions-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: 0 var(--space-4) var(--space-8);
  max-width: 600px;
  margin: 0 auto;
}

.action-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-decoration: none;
  transition: all var(--duration-base) var(--ease-out);
}

.action-card:hover {
  border-color: var(--border-light);
  background: var(--bg-elevated);
  transform: translateX(4px);
}

.action-card.primary {
  border-color: var(--border-accent);
  background: rgba(201, 162, 39, 0.05);
}

.action-card.primary:hover {
  background: rgba(201, 162, 39, 0.1);
}

.action-icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-elevated);
  border-radius: var(--radius-md);
  color: var(--accent);
  flex-shrink: 0;
}

.action-icon svg {
  width: 22px;
  height: 22px;
}

.action-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.action-title {
  font-weight: 600;
  color: var(--text-primary);
}

.action-desc {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

.action-arrow {
  width: 20px;
  height: 20px;
  color: var(--text-muted);
  transition: all var(--duration-fast);
  flex-shrink: 0;
}

.action-card:hover .action-arrow {
  color: var(--accent);
  transform: translateX(4px);
}

/* Reviews Section */
.reviews-section {
  padding: var(--space-8) var(--space-4);
  max-width: 1200px;
  margin: 0 auto;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}

.section-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.section-title svg {
  width: 24px;
  height: 24px;
  color: var(--accent);
}

.view-all {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-sm);
  color: var(--accent);
  text-decoration: none;
  transition: all var(--duration-fast);
}

.view-all:hover {
  gap: var(--space-2);
}

.view-all svg {
  width: 16px;
  height: 16px;
}

.reviews-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--space-4);
}

/* Responsive */
@media (max-width: 640px) {
  .hero-section {
    padding: var(--space-10) var(--space-4);
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .reviews-grid {
    grid-template-columns: 1fr;
  }
}
</style>
