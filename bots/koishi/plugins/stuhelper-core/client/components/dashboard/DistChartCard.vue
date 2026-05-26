<template>
  <div class="chart-card distribution-card">
    <div class="card-header">
      <k-icon name="stuhelperGroupCenter:octicons.bar-chart" />
      <h3>命令排行</h3>
      <span class="chart-subtitle">Top 10</span>
    </div>
    <div class="chart-container">
      <div v-if="loading" class="chart-loading">
        <k-icon name="loader" class="spin" />
      </div>
      <div v-else-if="!data || data.length === 0" class="chart-empty">
        暂无数据
      </div>
      <div v-else class="horizontal-bar-chart">
        <div
          v-for="(item, index) in data"
          :key="item.command"
          class="h-bar-item"
        >
          <div class="h-bar-label">
            <span class="h-bar-rank">{{ index + 1 }}</span>
            <span class="h-bar-name">{{ item.command }}</span>
          </div>
          <div class="h-bar-track">
            <div
              class="h-bar-fill"
              :style="{ width: getBarHeight(item.count, maxDistCount) + '%' }"
            ></div>
          </div>
          <span class="h-bar-count">{{ item.count }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { ChartDistributionItem } from '../../api'

const props = defineProps<{
  data: ChartDistributionItem[]
  loading?: boolean
}>()

const maxDistCount = computed(() => {
  if (!props.data || props.data.length === 0) return 1
  return Math.max(...props.data.map(i => i.count), 1)
})

const getBarHeight = (count: number, max: number) => {
  if (max === 0) return 0
  return Math.max((count / max) * 100, 2)
}
</script>

<style scoped>
.chart-container {
  flex: 1;
  display: flex;
  align-items: flex-start;
}

.chart-loading, .chart-empty {
  width: 100%;
  text-align: center;
  color: var(--fg3);
  font-size: 0.8rem;
  padding: 1rem 0;
}

.h-bar-name {
  font-size: 0.75rem;
  font-family: var(--sh-font-mono);
  color: var(--fg2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 60px;
}
</style>
