<template>
  <div class="course-rating-chart">
    <!-- Loading State -->
    <div v-if="loading" class="loading-state">
      <div class="skeleton-header">
        <div class="skeleton-title"></div>
        <div class="skeleton-select"></div>
      </div>
      <div class="skeleton-chart"></div>
      <div class="skeleton-list">
        <div v-for="i in 5" :key="i" class="skeleton-item">
          <div class="skeleton-name"></div>
          <div class="skeleton-stars"></div>
          <div class="skeleton-count"></div>
        </div>
      </div>
    </div>

    <!-- Content -->
    <template v-else-if="ratingStats">
      <div class="chart-header">
        <h3 class="chart-title">
          <svg class="title-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
          </svg>
          {{ t('review.chart.title') }}
        </h3>

        <!-- Custom Multi-Select -->
        <div class="term-selector" ref="selectorRef">
          <button class="selector-trigger" @click="toggleDropdown">
            <span class="selected-tags">
              <span v-if="selectedTerms.length === 0" class="placeholder">{{ t('review.chart.selectTerm') }}</span>
              <template v-else>
                <span v-for="term in selectedTermLabels.slice(0, 2)" :key="term" class="tag">
                  {{ term }}
                </span>
                <span v-if="selectedTerms.length > 2" class="tag more">
                  +{{ selectedTerms.length - 2 }}
                </span>
              </template>
            </span>
            <svg class="chevron" :class="{ open: dropdownOpen }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </button>

          <Transition name="dropdown">
            <div v-if="dropdownOpen" class="dropdown-menu">
              <label class="dropdown-item" :class="{ selected: selectedTerms.includes('overall') }">
                <input type="checkbox" value="overall" v-model="selectedTerms" />
                <span class="checkbox-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <path d="M5 12l5 5L20 7"/>
                  </svg>
                </span>
                <span>{{ t('review.chart.overall') }}</span>
              </label>
              <label
                v-for="term in ratingStats.byTerm"
                :key="term.termID"
                class="dropdown-item"
                :class="{ selected: selectedTerms.includes(term.termID ?? '') }"
              >
                <input type="checkbox" :value="term.termID" v-model="selectedTerms" />
                <span class="checkbox-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <path d="M5 12l5 5L20 7"/>
                  </svg>
                </span>
                <span>{{ term.termName }}</span>
              </label>
            </div>
          </Transition>
        </div>
      </div>

      <!-- Radar Chart -->
      <div class="chart-container">
        <v-chart :option="chartOption" autoresize />
      </div>

      <!-- Dimension List -->
      <div class="dimension-list">
        <div
          v-for="dim in ratingStats.overall.dimensions"
          :key="dim.key"
          class="dimension-item"
        >
          <span class="dim-name">{{ dim.name }}</span>
          <div class="dim-rating">
            <RatingDisplay :value="dim.avgRating" show-value />
          </div>
          <span class="dim-count">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            {{ dim.ratingCount }}
          </span>
        </div>
      </div>
    </template>

    <!-- Empty State -->
    <div v-else class="empty-state">
      <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>
      </svg>
      <p class="empty-text">{{ t('review.chart.emptyTitle') }}</p>
      <p class="empty-hint">{{ t('review.chart.emptyHint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { RadarChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { getCourseRatingStats } from '@/api/course'
import type { CourseRatingStatsResponse } from '@/types/course'
import RatingDisplay from './RatingDisplay.vue'

use([RadarChart, TooltipComponent, LegendComponent, CanvasRenderer])

const { t } = useI18n()

function cssVar(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

const props = defineProps<{
  courseID: number
}>()

const loading = ref(true)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)
const selectedTerms = ref<string[]>(['overall'])
const dropdownOpen = ref(false)
const selectorRef = ref<HTMLElement>()

// 设计系统配色 - 深色主题适配
const termColors = [
  { bg: 'rgba(201, 162, 39, 0.2)', border: '#c9a227' },   // 金色 (accent)
  { bg: 'rgba(76, 175, 80, 0.2)', border: '#4caf50' },    // 绿色
  { bg: 'rgba(33, 150, 243, 0.2)', border: '#2196f3' },   // 蓝色
  { bg: 'rgba(255, 152, 0, 0.2)', border: '#ff9800' },    // 橙色
  { bg: 'rgba(156, 39, 176, 0.2)', border: '#9c27b0' }    // 紫色
]

// 下拉菜单逻辑
const toggleDropdown = () => {
  dropdownOpen.value = !dropdownOpen.value
}

const closeDropdown = (e: MouseEvent) => {
  if (selectorRef.value && !selectorRef.value.contains(e.target as Node)) {
    dropdownOpen.value = false
  }
}

// 获取选中项的标签
const selectedTermLabels = computed(() => {
  return selectedTerms.value.map(id => {
    if (id === 'overall') return t('review.chart.overall')
    const term = ratingStats.value?.byTerm.find(t => t.termID === id)
    return term?.termName || id
  })
})

const chartOption = computed(() => {
  if (!ratingStats.value) return {}

  const labels = ratingStats.value.radarChart.labels
  const indicator = labels.map(name => ({ name, max: 5 }))

  const series: { name: string; value: number[] }[] = []

  selectedTerms.value.forEach((termID) => {
    if (termID === 'overall') {
      series.push({
        name: t('review.chart.overall'),
        value: ratingStats.value!.overall.dimensions.map(d => d.avgRating)
      })
    } else {
      const term = ratingStats.value!.byTerm.find(t => t.termID === termID)
      if (term) {
        series.push({
          name: term.termName,
          value: term.dimensions.map(d => d.avgRating)
        })
      }
    }
  })

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: cssVar('--bg-card') + 'F2',
      borderColor: cssVar('--border'),
      textStyle: { color: cssVar('--text-primary') }
    },
    legend: {
      data: series.map(s => s.name),
      bottom: 0,
      textStyle: { color: cssVar('--text-muted') }
    },
    radar: {
      indicator,
      shape: 'polygon',
      splitNumber: 5,
      axisName: { color: cssVar('--text-muted') },
      splitLine: { lineStyle: { color: cssVar('--border') } },
      splitArea: { areaStyle: { color: ['transparent', cssVar('--border-light') + '1A'] } },
      axisLine: { lineStyle: { color: cssVar('--border') } }
    },
    series: [{
      type: 'radar',
      data: series.map((s, i) => ({
        name: s.name,
        value: s.value,
        areaStyle: { color: termColors[i % termColors.length].bg },
        lineStyle: { color: termColors[i % termColors.length].border, width: 2 },
        itemStyle: { color: termColors[i % termColors.length].border },
        symbol: 'circle',
        symbolSize: 6
      }))
    }]
  }
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getCourseRatingStats(props.courseID)
    ratingStats.value = res.data
  } catch {
    // 评分统计加载失败，UI 显示空状态
  } finally {
    loading.value = false
  }
}

watch(() => props.courseID, fetchData)

onMounted(() => {
  fetchData()
  document.addEventListener('click', closeDropdown)
})

onUnmounted(() => {
  document.removeEventListener('click', closeDropdown)
})
</script>

<style scoped>
.course-rating-chart {
  padding: var(--space-5);
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.skeleton-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.skeleton-title {
  width: 100px;
  height: 24px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-select {
  width: 160px;
  height: 36px;
  background: var(--bg-elevated);
  border-radius: var(--radius-md);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-chart {
  height: 280px;
  background: var(--bg-elevated);
  border-radius: var(--radius-lg);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.skeleton-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.skeleton-name {
  width: 60px;
  height: 16px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-stars {
  width: 120px;
  height: 20px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-count {
  width: 50px;
  height: 14px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Chart Header */
.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.chart-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  font-family: var(--font-serif);
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.title-icon {
  width: 20px;
  height: 20px;
  color: var(--accent);
}

/* Term Selector */
.term-selector {
  position: relative;
}

.selector-trigger {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
  min-width: 140px;
}

.selector-trigger:hover {
  border-color: var(--text-primary);
}

.selected-tags {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  flex: 1;
  flex-wrap: wrap;
}

.placeholder {
  color: var(--text-muted);
}

.tag {
  padding: 2px var(--space-2);
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  color: var(--text-secondary);
}

.tag.more {
  background: var(--accent);
  color: var(--bg-primary);
}

.chevron {
  width: 16px;
  height: 16px;
  color: var(--text-muted);
  transition: transform var(--duration-fast);
  flex-shrink: 0;
}

.chevron.open {
  transform: rotate(180deg);
}

/* Dropdown Menu */
.dropdown-menu {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 160px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-lg);
  z-index: var(--z-dropdown);
  overflow: hidden;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  cursor: pointer;
  transition: background var(--duration-fast);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.dropdown-item:hover {
  background: var(--bg-hover);
}

.dropdown-item.selected {
  color: var(--accent);
}

.dropdown-item input {
  display: none;
}

.checkbox-icon {
  width: 16px;
  height: 16px;
  border: 1.5px solid var(--border);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--duration-fast);
}

.checkbox-icon svg {
  width: 12px;
  height: 12px;
  opacity: 0;
  transform: scale(0.5);
  transition: all var(--duration-fast);
}

.dropdown-item.selected .checkbox-icon {
  background: var(--accent);
  border-color: var(--accent);
}

.dropdown-item.selected .checkbox-icon svg {
  opacity: 1;
  transform: scale(1);
  color: var(--bg-primary);
}

/* Chart Container */
.chart-container {
  height: 280px;
  margin-bottom: var(--space-4);
}

/* Dimension List */
.dimension-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.dimension-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-card);
  border-radius: var(--radius-md);
  transition: background var(--duration-fast);
}

.dimension-item:hover {
  background: var(--bg-elevated);
}

.dim-name {
  width: 70px;
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.dim-rating {
  flex: 1;
}

.dim-count {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-xs);
  color: var(--text-muted);
  flex-shrink: 0;
}

.dim-count svg {
  width: 14px;
  height: 14px;
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-10) var(--space-4);
  text-align: center;
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: var(--text-muted);
  opacity: 0.5;
  margin-bottom: var(--space-4);
}

.empty-text {
  font-size: var(--text-base);
  color: var(--text-secondary);
  margin: 0 0 var(--space-2) 0;
}

.empty-hint {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0;
}

/* Dropdown Transition */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-fast) var(--ease-out);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Responsive */
@media (max-width: 480px) {
  .chart-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .term-selector {
    width: 100%;
  }

  .selector-trigger {
    width: 100%;
  }

  .dropdown-menu {
    left: 0;
    right: 0;
  }

  .dimension-item {
    flex-wrap: wrap;
  }

  .dim-name {
    width: 100%;
    margin-bottom: var(--space-1);
  }
}
</style>
