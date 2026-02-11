<template>
  <div class="teacher-detail-page">
    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="skeleton-header"></div>
      <div class="skeleton-chart"></div>
    </div>

    <!-- Content -->
    <template v-else-if="teacher">
      <!-- Teacher Header -->
      <header class="teacher-header">
        <div class="avatar-ring">
          <div class="avatar">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
          </div>
        </div>
        <div class="teacher-info">
          <h1 class="teacher-name">{{ teacher.teacherName }}</h1>
          <span class="department-pill">{{ teacher.departmentName }}</span>
        </div>
        <div class="overall-rating">
          <RatingCircle
            :value="teacher.avgRating || 0"
            :size="72"
            :stroke-width="5"
            :subtitle="t('teaching.profile.overallRating')"
          />
        </div>
      </header>

      <!-- Stats Cards -->
      <section class="stats-section">
        <div class="stat-card">
          <span class="stat-icon courses">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
              <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
            </svg>
          </span>
          <span class="stat-value">{{ teacher.courseCount }}</span>
          <span class="stat-label">{{ t('teaching.profile.courseCount') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon reviews">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
            </svg>
          </span>
          <span class="stat-value">{{ teacher.reviewCount }}</span>
          <span class="stat-label">{{ t('teaching.profile.reviewCount') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon trend">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/>
              <polyline points="17 6 23 6 23 12"/>
            </svg>
          </span>
          <span class="stat-value">{{ trendText }}</span>
          <span class="stat-label">{{ t('teaching.profile.ratingTrend') }}</span>
        </div>
      </section>

      <!-- Rating Trend Chart -->
      <section class="chart-section">
        <h2 class="section-title">{{ t('teaching.profile.ratingTrend') }}</h2>
        <div ref="chartRef" class="chart-container"></div>
      </section>

      <!-- Courses List -->
      <section class="courses-section">
        <h2 class="section-title">{{ t('teaching.profile.courseList') }}</h2>
        <div class="course-list">
          <router-link
            v-for="course in teacher.courses"
            :key="course.id"
            :to="`/review/courses/${course.id}`"
            class="course-item"
          >
            <div class="course-info">
              <span class="course-name">{{ course.name }}</span>
              <span class="course-meta">{{ t('teaching.profile.reviewsCount', { count: course.reviewCount }) }}</span>
            </div>
            <div class="course-rating">
              <span class="rating">{{ course.avgRating?.toFixed(1) || '-' }}</span>
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
              </svg>
            </div>
          </router-link>
        </div>
      </section>
    </template>

    <!-- Error State -->
    <div v-else class="error-state">
      <p>{{ t('teaching.profile.notFound') }}</p>
      <router-link to="/" class="back-link">{{ t('teaching.profile.backToHome') }}</router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { init as echartsInit, graphic } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import type { ECharts } from 'echarts/core'

// 注册按需组件
import { use } from 'echarts/core'
use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])
import { getTeacherStats } from '@/api/course'
import RatingCircle from '@/components/common/RatingCircle.vue'

interface TeacherCourse {
  id: number
  name: string
  avgRating: number | null
  reviewCount: number
}

interface RatingTrendItem {
  termID: string
  termName: string
  avgRating: number
}

interface TeacherDetail {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating: number | null
  courseCount: number
  reviewCount: number
  courses: TeacherCourse[]
  ratingTrend: RatingTrendItem[]
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const teacherID = computed(() => Number(route.params.id))

const loading = ref(true)
const teacher = ref<TeacherDetail | null>(null)
const chartRef = ref<HTMLElement>()
let chartInstance: ECharts | null = null

const trendText = computed(() => {
  if (!teacher.value?.ratingTrend?.length) return '-'
  const trend = teacher.value.ratingTrend
  if (trend.length < 2) return t('teaching.profile.trend.stable')
  const diff = trend[trend.length - 1].avgRating - trend[0].avgRating
  if (diff > 0.2) return t('teaching.profile.trend.up')
  if (diff < -0.2) return t('teaching.profile.trend.down')
  return t('teaching.profile.trend.stable')
})

const fetchTeacher = async () => {
  loading.value = true
  try {
    const res = await getTeacherStats(teacherID.value)
    teacher.value = res.data as TeacherDetail
  } catch {
    // 加载失败时 teacher 保持 null，UI 显示错误状态
  } finally {
    loading.value = false
  }
}

// 解析 CSS 变量为实际颜色值（ECharts 不支持 CSS 变量）
function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

const initChart = () => {
  if (!chartRef.value || !teacher.value?.ratingTrend?.length) return

  chartInstance = echartsInit(chartRef.value)
  const trend = teacher.value.ratingTrend

  const borderColor = cssVar('--border')
  const borderLightColor = cssVar('--border-light')
  const textMutedColor = cssVar('--text-muted')
  const accentColor = cssVar('--accent')

  chartInstance.setOption({
    grid: { top: 20, right: 20, bottom: 30, left: 40 },
    xAxis: {
      type: 'category',
      data: trend.map(t => t.termName),
      axisLine: { lineStyle: { color: borderColor } },
      axisLabel: { color: textMutedColor, fontSize: 12 }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 5,
      axisLine: { show: false },
      splitLine: { lineStyle: { color: borderLightColor } },
      axisLabel: { color: textMutedColor, fontSize: 12 }
    },
    series: [{
      type: 'line',
      data: trend.map(t => t.avgRating),
      smooth: true,
      symbol: 'circle',
      symbolSize: 8,
      lineStyle: { color: accentColor, width: 2 },
      itemStyle: { color: accentColor },
      areaStyle: {
        color: new graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: accentColor + '4D' },
          { offset: 1, color: accentColor + '00' }
        ])
      }
    }],
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const p = params[0]
        return `${p.name}<br/>${t('teaching.profile.ratingLabel')}: ${p.value.toFixed(1)}`
      }
    }
  })
}

const handleResize = () => {
  chartInstance?.resize()
}

watch(() => teacher.value, () => {
  if (teacher.value) {
    nextTick(initChart)
  }
})

onMounted(async () => {
  if (isNaN(teacherID.value) || teacherID.value <= 0) {
    router.replace({ name: 'teaching-hub' })
    return
  }
  await fetchTeacher()
  window.addEventListener('resize', handleResize)
})

// 路由参数变化时重新加载数据
watch(teacherID, async (newID, oldID) => {
  if (newID === oldID || isNaN(newID) || newID <= 0) return
  chartInstance?.dispose()
  chartInstance = null
  await fetchTeacher()
})

onUnmounted(() => {
  chartInstance?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.teacher-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-6);
  animation: fadeIn var(--duration-base) var(--ease-out);
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.skeleton-header {
  height: 120px;
  background: linear-gradient(90deg, var(--bg-secondary) 25%, var(--bg-tertiary) 50%, var(--bg-secondary) 75%);
  background-size: 200% 100%;
  border-radius: var(--radius-lg);
  animation: shimmer 1.5s ease-in-out infinite;
}

.skeleton-chart {
  height: 300px;
  background: linear-gradient(90deg, var(--bg-secondary) 25%, var(--bg-tertiary) 50%, var(--bg-secondary) 75%);
  background-size: 200% 100%;
  border-radius: var(--radius-lg);
  animation: shimmer 1.5s ease-in-out infinite;
}

/* Teacher Header */
.teacher-header {
  display: flex;
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-6);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  margin-bottom: var(--space-6);
}

.avatar-ring {
  padding: 3px;
  background: var(--gradient-brand);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.avatar {
  width: 64px;
  height: 64px;
  background: var(--bg-card);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.avatar svg {
  width: 32px;
  height: 32px;
}

.teacher-info {
  flex: 1;
}

.teacher-name {
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0 0 var(--space-2) 0;
}

.department-pill {
  display: inline-block;
  font-size: var(--text-xs);
  color: var(--text-secondary);
  background: var(--bg-secondary);
  padding: 2px 10px;
  border-radius: var(--radius-full);
}

.overall-rating {
  flex-shrink: 0;
}

/* Stats Section */
.stats-section {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-8);
  margin-bottom: var(--space-6);
  padding: var(--space-5) 0;
  border-bottom: 1px solid var(--border);
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.stat-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-2);
  color: var(--text-muted);
}

.stat-icon svg {
  width: 20px;
  height: 20px;
}

.stat-icon.courses { color: var(--brand-primary); }
.stat-icon.reviews { color: var(--brand-accent); }
.stat-icon.trend { color: var(--brand-primary); }

.stat-card .stat-value {
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--weight-bold);
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.stat-card .stat-label {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin-top: var(--space-1);
}

/* Chart Section */
.chart-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  margin-bottom: var(--space-6);
}

.section-title {
  font-family: var(--font-sans);
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
  margin: 0 0 var(--space-4) 0;
}

.chart-container {
  height: 250px;
}

/* Courses Section */
.courses-section {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
}

.course-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.course-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
  text-decoration: none;
  transition: all var(--duration-fast);
}

.course-item:last-child {
  border-bottom: none;
}

.course-item:hover {
  background: var(--bg-hover);
  padding-left: var(--space-6);
}

.course-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.course-name {
  font-family: var(--font-sans);
  font-weight: var(--weight-medium);
  color: var(--text-primary);
}

.course-meta {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.course-rating {
  display: flex;
  align-items: center;
  gap: var(--space-1);
}

.course-rating .rating {
  font-weight: var(--weight-semibold);
  color: var(--brand-primary);
  font-variant-numeric: tabular-nums;
}

.course-rating svg {
  width: 14px;
  height: 14px;
  color: var(--brand-primary);
}

/* Error State */
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-12);
  text-align: center;
  color: var(--text-muted);
}

.back-link {
  margin-top: var(--space-4);
  color: var(--brand-primary);
  text-decoration: none;
}

.back-link:hover {
  text-decoration: underline;
}

/* Responsive */
@media (max-width: 640px) {
  .teacher-detail-page {
    padding: var(--space-4);
  }

  .teacher-header {
    flex-direction: column;
    text-align: center;
  }

  .overall-rating {
    width: 100%;
  }

  .stats-section {
    gap: var(--space-4);
  }
}
</style>