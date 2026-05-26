<template>
  <div class="chart-card rank-card">
    <div class="card-header">
      <k-icon :name="resolvedIcon" />
      <h3>{{ title }}</h3>
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
          :key="rankItemId(item) || index"
          class="h-bar-item"
        >
          <div class="h-bar-label">
            <span class="h-bar-rank">{{ index + 1 }}</span>
            <span class="h-bar-name" :title="rankItemId(item)">{{ rankItemLabel(item) }}</span>
          </div>
          <div class="h-bar-track">
            <div
              class="h-bar-fill"
              :class="type"
              :style="{ width: getBarHeight(item.count, maxCount) + '%' }"
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

import type { ChartGuildRankItem, ChartUserRankItem } from '../../api'

type RankChartItem = ChartGuildRankItem | ChartUserRankItem

const props = defineProps<{
  type: 'guild' | 'user'
  data: RankChartItem[]
  loading?: boolean
  title: string
  icon: string
}>()

// 自动添加 stuhelperGroupCenter 命名空间前缀
const resolvedIcon = computed(() => {
  if (!props.icon) return 'stuhelperGroupCenter:users'
  if (props.icon.includes(':')) return props.icon
  return `stuhelperGroupCenter:${props.icon}`
})

const maxCount = computed(() => {
  if (!props.data || props.data.length === 0) return 1
  return Math.max(...props.data.map(i => i.count), 1)
})

function rankItemId(item: RankChartItem): string {
  if (props.type === 'guild' && 'guildId' in item) return item.guildId
  if (props.type === 'user' && 'userId' in item) return item.userId
  return 'guildId' in item ? item.guildId : item.userId
}

function rankItemLabel(item: RankChartItem): string {
  return item.name || rankItemId(item)
}

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
  color: var(--fg2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 60px;
}

/* 群聊和用户使用克制的 Koishi 颜色 */
.h-bar-fill.guild {
  background: var(--k-color-success);
}
.h-bar-fill.user {
  background: var(--k-color-warning);
}
</style>
