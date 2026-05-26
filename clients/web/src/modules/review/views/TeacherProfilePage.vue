<template>
  <div class="max-w-[900px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col gap-6">
      <div
        class="h-[120px] rounded-lg bg-[length:200%_100%] animate-shimmer"
        :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
      />
      <div
        class="h-[300px] rounded-lg bg-[length:200%_100%] animate-shimmer"
        :style="{ backgroundImage: 'linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-tertiary) 50%, var(--color-bg-secondary) 75%)' }"
      />
    </div>

    <!-- Content -->
    <template v-else-if="teacher">
      <!-- Teacher Header -->
      <header class="flex items-center gap-5 p-6 bg-bg-card rounded-xl mb-6 max-sm:flex-col max-sm:text-center">
        <div class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0">
          <div class="w-16 h-16 bg-bg-card rounded-full flex items-center justify-center text-text-muted">
            <User :size="32" :stroke-width="1.5" />
          </div>
        </div>
        <div class="flex-1">
          <h1 class="text-xl font-extrabold tracking-tight text-text-primary m-0 mb-2">{{ teacher.teacherName }}</h1>
          <span class="inline-block text-xs text-text-secondary bg-bg-secondary py-0.5 px-2.5 rounded-full">{{ teacher.departmentName }}</span>
        </div>
        <div class="shrink-0 max-sm:w-full">
          <RatingCircle
            :value="teacher.avgRating || 0"
            :size="72"
            :stroke-width="5"
            :subtitle="t('teaching.profile.overallRating')"
          >
            <EmojiRating v-if="teacher.avgRating" :value="teacher.avgRating" size="md" />
          </RatingCircle>
        </div>
      </header>

      <!-- Stats Cards -->
      <section class="flex items-center justify-center gap-8 mb-6 py-5 border-b border-border-light max-sm:gap-4">
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-primary">
            <BookOpen :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ teacher.courseCount }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.courseCount') }}</span>
        </div>
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-accent">
            <MessageSquare :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ teacher.reviewCount }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.reviewCount') }}</span>
        </div>
        <div class="flex flex-col items-center text-center">
          <span class="w-9 h-9 flex items-center justify-center mb-2 text-primary">
            <TrendingUp :size="20" />
          </span>
          <span class="font-display text-xl font-bold text-text-primary tabular-nums">{{ trendText }}</span>
          <span class="text-xs text-text-muted mt-1">{{ t('teaching.profile.ratingTrend') }}</span>
        </div>
      </section>

      <!-- Rating Trend Chart -->
      <section class="bg-bg-card rounded-lg p-5 mb-6">
        <h2 class="text-base font-semibold text-text-primary m-0 mb-4">{{ t('teaching.profile.ratingTrend') }}</h2>
        <div ref="chartRef" class="h-[250px]" role="img" :aria-label="t('teaching.profile.ratingTrendChartAria')"></div>
      </section>

      <!-- Courses List -->
      <section class="bg-bg-card rounded-lg p-5">
        <h2 class="text-base font-semibold text-text-primary m-0 mb-4">{{ t('teaching.profile.courseList') }}</h2>
        <div class="flex flex-col gap-2">
          <router-link
            v-for="course in teacher.courses"
            :key="course.id"
            :to="`/courses/${course.id}`"
            class="flex items-center justify-between py-3 px-4 border-b border-border-light no-underline transition-all duration-fast last:border-b-0 hover:bg-bg-hover hover:pl-6"
          >
            <div class="flex flex-col gap-1">
              <span class="font-medium text-text-primary">{{ course.name }}</span>
              <span class="text-xs text-text-muted">{{ t('teaching.profile.reviewsCount', { count: course.reviewCount }) }}</span>
            </div>
            <div class="flex items-center gap-1">
              <EmojiRating v-if="course.avgRating" :value="course.avgRating" size="sm" />
              <span v-else class="text-text-muted">-</span>
            </div>
          </router-link>
        </div>
      </section>
    </template>

    <!-- Error State -->
    <div v-else-if="loadError" role="alert" class="flex flex-col items-center justify-center p-12 text-center text-text-muted">
      <p>{{ loadError }}</p>
      <button
        type="button"
        class="mt-4 rounded-lg bg-primary px-4 py-2 text-white hover:bg-primary/90 cursor-pointer"
        @click="fetchTeacher"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>
    <div v-else class="flex flex-col items-center justify-center p-12 text-center text-text-muted">
      <p>{{ t('teaching.profile.notFound') }}</p>
      <router-link to="/" class="mt-4 text-primary no-underline hover:underline">{{ t('teaching.profile.backToHome') }}</router-link>
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
import { api } from '@/api'
import { useThemeStore } from '@/stores/theme'
import RatingCircle from '@/components/common/RatingCircle.vue'
import EmojiRating from '@/components/business/review/EmojiRating.vue'
import { withAlpha } from '@stuhelper/shared/utils'
import { User, BookOpen, MessageSquare, TrendingUp } from 'lucide-vue-next'

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
const themeStore = useThemeStore()
const teacherID = computed(() => Number(route.params.id))

const loading = ref(true)
const teacher = ref<TeacherDetail | null>(null)
const loadError = ref('')
const chartRef = ref<HTMLElement>()
let chartInstance: ECharts | null = null

// 缓存 CSS 变量值，避免每次 initChart 时调用 getComputedStyle
function readCssVars() {
  const style = getComputedStyle(document.documentElement)
  return {
    border: style.getPropertyValue('--color-border').trim(),
    borderLight: style.getPropertyValue('--color-border-light').trim(),
    textMuted: style.getPropertyValue('--color-text-muted').trim(),
    accent: style.getPropertyValue('--color-accent').trim(),
  }
}

const cssVarCache = ref(readCssVars())

// 主题切换时刷新缓存并重绘图表
watch(() => themeStore.resolvedTheme, () => {
  nextTick(() => {
    cssVarCache.value = readCssVars()
    if (chartInstance) {
      initChart()
    }
  })
})

const trendText = computed(() => {
  if (!teacher.value?.ratingTrend?.length) return '-'
  const trend = teacher.value.ratingTrend
  if (trend.length < 2) return t('teaching.profile.trend.stable')
  const diff = trend[trend.length - 1].avgRating - trend[0].avgRating
  if (diff > 0.2) return t('teaching.profile.trend.up')
  if (diff < -0.2) return t('teaching.profile.trend.down')
  return t('teaching.profile.trend.stable')
})

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readPositiveInteger(
  record: Record<string, unknown>,
  key: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readNonNegativeInteger(
  record: Record<string, unknown>,
  key: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readString(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readRating(record: Record<string, unknown>, key: string): number {
  const value = record[key]
  if (!isNonNegativeNumber(value) || value > 5) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readOptionalRating(record: Record<string, unknown>, key: string): number | null {
  const value = record[key]
  if (value === undefined || value === null) {
    return null
  }
  if (!isNonNegativeNumber(value) || value > 5) {
    throw new Error('Invalid teacher stats response')
  }
  return value
}

function readTeacherCourse(payload: unknown): TeacherCourse {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    id: readPositiveInteger(payload, 'id'),
    name: readString(payload, 'name'),
    avgRating: readOptionalRating(payload, 'avgRating'),
    reviewCount: readNonNegativeInteger(payload, 'reviewCount'),
  }
}

function readRatingTrendItem(payload: unknown): RatingTrendItem {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    termID: readString(payload, 'termID'),
    termName: readString(payload, 'termName'),
    avgRating: readRating(payload, 'avgRating'),
  }
}

function readTeacherPayload(payload: unknown): TeacherDetail {
  if (!isRecord(payload)) {
    throw new Error('Invalid teacher stats response')
  }

  if (!Array.isArray(payload.courses) || !Array.isArray(payload.ratingTrend)) {
    throw new Error('Invalid teacher stats response')
  }

  return {
    teacherID: readPositiveInteger(payload, 'teacherID'),
    teacherName: readString(payload, 'teacherName'),
    departmentName: readString(payload, 'departmentName'),
    avgRating: readOptionalRating(payload, 'avgRating'),
    courseCount: readNonNegativeInteger(payload, 'courseCount'),
    reviewCount: readNonNegativeInteger(payload, 'reviewCount'),
    courses: payload.courses.map(readTeacherCourse),
    ratingTrend: payload.ratingTrend.map(readRatingTrendItem),
  }
}

const fetchTeacher = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await api.rating.getTeacherStats(teacherID.value)
    teacher.value = readTeacherPayload(res.data?.data)
  } catch (_error) { void _error;
    teacher.value = null
    loadError.value = t('common.loadFailed')
  } finally {
    loading.value = false
  }
}

// 解析 CSS 变量为实际颜色值（ECharts 不支持 CSS 变量）— 使用缓存
const initChart = () => {
  if (!chartRef.value || !teacher.value?.ratingTrend?.length) return

  // 销毁旧实例，防止内存泄漏
  chartInstance?.dispose()
  chartInstance = echartsInit(chartRef.value)
  const trend = teacher.value.ratingTrend

  const cv = cssVarCache.value

  chartInstance.setOption({
    grid: { top: 20, right: 20, bottom: 30, left: 40 },
    xAxis: {
      type: 'category',
      data: trend.map(item => item.termName),
      axisLine: { lineStyle: { color: cv.border } },
      axisLabel: { color: cv.textMuted, fontSize: 12 }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 5,
      axisLine: { show: false },
      splitLine: { lineStyle: { color: cv.borderLight } },
      axisLabel: { show: false }
    },
    series: [{
      type: 'line',
      data: trend.map(item => item.avgRating),
      smooth: true,
      symbol: 'circle',
      symbolSize: 8,
      lineStyle: { color: cv.accent, width: 2 },
      itemStyle: { color: cv.accent },
      areaStyle: {
        color: new graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: withAlpha(cv.accent, 0.3) },
          { offset: 1, color: withAlpha(cv.accent, 0) }
        ])
      }
    }],
    tooltip: {
      trigger: 'axis',
      formatter: (params: { name: string; value: number }[]) => {
        if (!params || params.length === 0) return ''
        const p = params[0]
        return `${p.name}<br/>${t('teaching.profile.ratingLabel')}`
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
    router.replace({ name: 'course-hub' })
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
