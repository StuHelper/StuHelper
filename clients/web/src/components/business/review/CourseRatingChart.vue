<template>
  <div class="p-5">
    <!-- 加载态 -->
    <div v-if="loading" class="flex flex-col gap-4">
      <div class="flex justify-between items-center">
        <div class="w-[100px] h-6 bg-bg-elevated rounded-sm animate-pulse"></div>
        <div class="w-[160px] h-9 bg-bg-elevated rounded-md animate-pulse"></div>
      </div>
      <div class="h-[280px] bg-bg-elevated rounded-lg animate-pulse"></div>
      <div class="flex flex-col gap-3">
        <div v-for="i in 5" :key="i" class="flex items-center gap-3">
          <div class="w-[60px] h-4 bg-bg-elevated rounded-sm animate-pulse"></div>
          <div class="w-[120px] h-5 bg-bg-elevated rounded-sm animate-pulse"></div>
          <div class="w-[50px] h-3.5 bg-bg-elevated rounded-sm animate-pulse"></div>
        </div>
      </div>
    </div>

    <!-- 内容区 -->
    <template v-else-if="ratingStats">
      <div class="flex justify-between items-center mb-4 flex-wrap gap-3 max-sm:flex-col max-sm:items-start">
        <h3 class="flex items-center gap-2 m-0 font-serif text-base font-semibold text-text-primary">
          <Star class="w-5 h-5 text-accent" />
          {{ t('review.chart.title') }}
        </h3>

        <!-- 自定义多选下拉 -->
        <div class="relative" ref="selectorRef">
          <button
            class="flex items-center gap-2 py-2 px-3 bg-transparent rounded-sm text-text-primary text-sm cursor-pointer transition-all duration-fast min-w-[140px] hover:border-text-primary max-sm:w-full"
            @click="toggleDropdown"
          >
            <span class="flex items-center gap-1 flex-1 flex-wrap">
              <span v-if="selectedTerms.length === 0" class="text-text-muted">{{ t('review.chart.selectTerm') }}</span>
              <template v-else>
                <span v-for="term in selectedTermLabels.slice(0, 2)" :key="term" class="py-px px-2 bg-bg-elevated rounded-sm text-xs text-text-secondary">
                  {{ term }}
                </span>
                <span v-if="selectedTerms.length > 2" class="py-px px-2 bg-accent text-bg-base rounded-sm text-xs">
                  +{{ selectedTerms.length - 2 }}
                </span>
              </template>
            </span>
            <ChevronDown
              class="w-4 h-4 text-text-muted shrink-0 transition-transform duration-fast"
              :class="{ 'rotate-180': dropdownOpen }"
            />
          </button>

          <Transition name="dropdown">
            <div v-if="dropdownOpen" class="absolute top-full mt-1 right-0 min-w-[160px] bg-bg-card rounded-sm shadow-lg z-40 overflow-hidden max-sm:left-0">
              <label
                class="dropdown-item"
                :class="{ selected: selectedTerms.includes('overall') }"
              >
                <input type="checkbox" value="overall" v-model="selectedTerms" class="hidden" />
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
                <input type="checkbox" :value="term.termID" v-model="selectedTerms" class="hidden" />
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

      <!-- 雷达图 -->
      <div class="h-[280px] mb-4" role="img" :aria-label="t('review.chart.radarAria')">
        <v-chart :option="chartOption" autoresize />
      </div>

      <!-- 维度列表 -->
      <div class="flex flex-col gap-3 pt-4 border-t border-border-light">
        <div
          v-for="dim in ratingStats.overall.dimensions"
          :key="dim.key"
          class="flex items-center gap-3 py-2 px-3 bg-bg-card rounded-md transition-[background] duration-fast hover:bg-bg-elevated max-sm:flex-wrap"
        >
          <span class="w-[70px] text-sm font-medium text-text-secondary shrink-0 max-sm:w-full max-sm:mb-1">
            {{ dimensionLabel(dim.key, dim.name) }}
          </span>
          <div class="flex-1">
            <RatingDisplay :value="dim.avgRating" />
          </div>
          <span class="flex items-center gap-1 text-xs text-text-muted shrink-0">
            <Users class="w-3.5 h-3.5" />
            {{ dim.ratingCount }}
          </span>
        </div>
      </div>
    </template>

    <!-- 空状态 -->
    <div v-else class="flex flex-col items-center justify-center py-10 px-4 text-center">
      <Star class="w-16 h-16 text-text-muted opacity-50 mb-4" :stroke-width="1.5" />
      <p class="text-base text-text-secondary m-0 mb-2">{{ t('review.chart.emptyTitle') }}</p>
      <p class="text-sm text-text-muted m-0">{{ t('review.chart.emptyHint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { use } from 'echarts/core'
import { RadarChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { Star, Users, ChevronDown } from 'lucide-vue-next'
import { api } from '@/api'
import type { CourseRatingStatsResponse } from '@stuhelper/shared/course'
import { useThemeStore } from '@/stores/theme'
import RatingDisplay from './RatingDisplay.vue'
import { withAlpha } from '@stuhelper/shared/utils'
import { ratingDimensionLabel } from '@/modules/review/ratingHelpers'

use([RadarChart, TooltipComponent, LegendComponent, CanvasRenderer])

const { t } = useI18n()
const themeStore = useThemeStore()

// 缓存 CSS 变量值，避免每次 computed 求值时调用 getComputedStyle
function readCssVars() {
  const style = getComputedStyle(document.documentElement)
  return {
    bgCard: style.getPropertyValue('--color-bg-card').trim(),
    border: style.getPropertyValue('--color-border').trim(),
    borderLight: style.getPropertyValue('--color-border-light').trim(),
    textPrimary: style.getPropertyValue('--color-text-primary').trim(),
    textMuted: style.getPropertyValue('--color-text-muted').trim(),
  }
}

const cssVarCache = ref(readCssVars())

// 主题切换时刷新缓存
watch(() => themeStore.resolvedTheme, () => {
  nextTick(() => {
    cssVarCache.value = readCssVars()
  })
})

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
    const term = ratingStats.value?.byTerm.find(item => item.termID === id)
    return term?.termName || id
  })
})

function dimensionLabel(key: string, fallback?: string): string {
  return ratingDimensionLabel({ key, fallback, t })
}

const chartOption = computed(() => {
  if (!ratingStats.value) return {}

  const labels = ratingStats.value.overall.dimensions.map(d => dimensionLabel(d.key, d.name))
  const indicator = labels.map(name => ({ name, max: 5 }))

  const series: { name: string; value: number[] }[] = []

  selectedTerms.value.forEach((termID) => {
    if (termID === 'overall') {
      series.push({
        name: t('review.chart.overall'),
        value: ratingStats.value!.overall.dimensions.map(d => d.avgRating)
      })
    } else {
      const term = ratingStats.value!.byTerm.find(item => item.termID === termID)
      if (term) {
        series.push({
          name: term.termName,
          value: term.dimensions.map(d => d.avgRating)
        })
      }
    }
  })

  const cv = cssVarCache.value

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: withAlpha(cv.bgCard, 0.95),
      borderColor: cv.border,
      textStyle: { color: cv.textPrimary },
      formatter: (params: { name?: string }) => params.name ?? ''
    },
    legend: {
      data: series.map(s => s.name),
      bottom: 0,
      textStyle: { color: cv.textMuted }
    },
    radar: {
      indicator,
      shape: 'polygon',
      splitNumber: 5,
      axisName: { color: cv.textMuted },
      splitLine: { lineStyle: { color: cv.border } },
      splitArea: { areaStyle: { color: ['transparent', withAlpha(cv.borderLight, 0.1)] } },
      axisLine: { lineStyle: { color: cv.border } }
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
    const res = await api.rating.getCourseStats(props.courseID)
    ratingStats.value = res.data?.data ?? null
  } catch (_error) { void _error;
    // 评分统计加载失败，UI 显示空状态
  } finally {
    loading.value = false
  }
}

watch(() => props.courseID, () => {
  selectedTerms.value = ['overall']
  dropdownOpen.value = false
  fetchData()
})

onMounted(() => {
  fetchData()
  document.addEventListener('click', closeDropdown)
})

onUnmounted(() => {
  document.removeEventListener('click', closeDropdown)
})
</script>

<style scoped>
/* Dropdown 项 — checkbox 复杂选中态（transform + opacity 联动）无法用纯 utility 表达 */
.dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  cursor: pointer;
  transition: background var(--duration-fast);
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}
.dropdown-item:hover {
  background: var(--color-bg-hover);
}
.dropdown-item.selected {
  color: var(--color-accent);
}

.checkbox-icon {
  width: 16px;
  height: 16px;
  border: 1.5px solid var(--color-border);
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
  background: var(--color-accent);
  border-color: var(--color-accent);
}
.dropdown-item.selected .checkbox-icon svg {
  opacity: 1;
  transform: scale(1);
  color: var(--color-bg-base);
}

/* 下拉面板过渡动画 */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-fast) var(--ease-out);
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
