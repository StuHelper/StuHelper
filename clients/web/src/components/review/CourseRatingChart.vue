<template>
  <div class="course-rating-chart">
    <div v-if="loading" class="loading">
      <el-skeleton :rows="6" animated />
    </div>
    <template v-else-if="ratingStats">
      <div class="chart-header">
        <h3>评分统计</h3>
        <el-select
          v-model="selectedTerms"
          multiple
          collapse-tags
          collapse-tags-tooltip
          placeholder="选择学期"
          size="small"
          style="width: 200px"
        >
          <el-option label="总体" value="overall" />
          <el-option
            v-for="term in ratingStats.byTerm"
            :key="term.termId"
            :label="term.termName"
            :value="term.termId"
          />
        </el-select>
      </div>
      <div class="chart-container">
        <v-chart :option="chartOption" autoresize />
      </div>
      <div class="dimension-list">
        <div
          v-for="dim in ratingStats.overall.dimensions"
          :key="dim.key"
          class="dimension-item"
        >
          <span class="dim-name">{{ dim.name }}</span>
          <RatingDisplay :value="dim.avgRating" show-value />
          <span class="dim-count">{{ dim.ratingCount }}人评价</span>
        </div>
      </div>
    </template>
    <el-empty v-else description="暂无评分数据" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { use } from 'echarts/core'
import { RadarChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { getCourseRatingStats } from '@/api/course'
import type { CourseRatingStatsResponse } from '@/types/course'
import RatingDisplay from './RatingDisplay.vue'

use([RadarChart, TooltipComponent, LegendComponent, CanvasRenderer])

const props = defineProps<{
  courseId: number
}>()

const loading = ref(true)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)
const selectedTerms = ref<string[]>(['overall'])

const termColors = [
  { bg: 'rgba(64, 158, 255, 0.2)', border: '#409EFF' },
  { bg: 'rgba(103, 194, 58, 0.2)', border: '#67C23A' },
  { bg: 'rgba(230, 162, 60, 0.2)', border: '#E6A23C' },
  { bg: 'rgba(245, 108, 108, 0.2)', border: '#F56C6C' },
  { bg: 'rgba(144, 147, 153, 0.2)', border: '#909399' }
]

const chartOption = computed(() => {
  if (!ratingStats.value) return {}

  const labels = ratingStats.value.radarChart.labels
  const indicator = labels.map(name => ({ name, max: 5 }))

  const series: { name: string; value: number[] }[] = []

  selectedTerms.value.forEach((termId, index) => {
    const colorIndex = index % termColors.length
    if (termId === 'overall') {
      series.push({
        name: '总体',
        value: ratingStats.value!.overall.dimensions.map(d => d.avgRating)
      })
    } else {
      const term = ratingStats.value!.byTerm.find(t => t.termId === termId)
      if (term) {
        series.push({
          name: term.termName,
          value: term.dimensions.map(d => d.avgRating)
        })
      }
    }
  })

  return {
    tooltip: { trigger: 'item' },
    legend: {
      data: series.map(s => s.name),
      bottom: 0
    },
    radar: {
      indicator,
      shape: 'polygon',
      splitNumber: 5,
      axisName: { color: '#606266' }
    },
    series: [{
      type: 'radar',
      data: series.map((s, i) => ({
        name: s.name,
        value: s.value,
        areaStyle: { color: termColors[i % termColors.length].bg },
        lineStyle: { color: termColors[i % termColors.length].border }
      }))
    }]
  }
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getCourseRatingStats(props.courseId)
    ratingStats.value = res.data
  } catch (e) {
    console.error('Failed to load rating stats:', e)
  } finally {
    loading.value = false
  }
}

watch(() => props.courseId, fetchData)
onMounted(fetchData)
</script>

<style scoped>
.course-rating-chart {
  padding: 16px;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.chart-header h3 {
  margin: 0;
  font-size: 16px;
  color: #303133;
}

.chart-container {
  height: 300px;
}

.dimension-list {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dimension-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dim-name {
  width: 80px;
  font-size: 14px;
  color: #606266;
}

.dim-count {
  font-size: 12px;
  color: #909399;
}
</style>
