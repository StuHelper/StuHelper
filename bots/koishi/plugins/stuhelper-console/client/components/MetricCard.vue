<template>
  <article class="sh-stat" :class="toneClass">
    <span class="sh-stat__label">{{ label }}</span>
    <span class="sh-stat__value sh-num">{{ formatValue(value) }}</span>
    <span v-if="note" class="sh-stat__note">{{ note }}</span>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    label: string
    value: number | string
    note?: string
    tone?: 'neutral' | 'primary' | 'warning' | 'danger' | 'success'
  }>(),
  {
    note: '',
    tone: 'neutral',
  },
)

const toneClass = computed(() => (props.tone === 'neutral' ? '' : `sh-stat--${props.tone}`))

function formatValue(value: number | string) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return new Intl.NumberFormat('en-US').format(value)
  }
  return String(value)
}
</script>
