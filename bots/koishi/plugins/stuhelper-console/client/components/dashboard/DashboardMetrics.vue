<template>
  <div class="sh-dashboard-metrics">
    <article
      v-for="metric in metrics"
      :key="metric.label"
      class="sh-stat sh-dashboard-metric"
      :class="metricClass(metric.tone)"
    >
      <span class="sh-stat__label">{{ metric.label }}</span>
      <span class="sh-stat__value sh-num">{{ formatValue(metric.value) }}</span>
      <span class="sh-stat__note">{{ metric.note }}</span>
    </article>
  </div>
</template>

<script setup lang="ts">
import type { DashboardMetric } from '../../dashboard/model'

defineProps<{
  metrics: readonly DashboardMetric[]
}>()

function metricClass(tone?: DashboardMetric['tone']) {
  return tone ? `sh-stat--${tone}` : ''
}

function formatValue(value: number) {
  return new Intl.NumberFormat('en-US').format(value)
}
</script>
