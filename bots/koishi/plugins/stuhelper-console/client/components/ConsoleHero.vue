<template>
  <header class="sh-topbar">
    <div class="sh-topbar__brand">
      <span class="sh-topbar__mark">SH</span>
      <div class="sh-topbar__title">
        <span class="sh-topbar__eyebrow">{{ eyebrow }}</span>
        <span class="sh-topbar__headline">{{ title }}</span>
      </div>
    </div>
    <div class="sh-topbar__meta">
      <span v-if="generatedAt" class="sh-topbar__sync sh-num">
        同步 {{ formatSyncTime(generatedAt) }}
      </span>
      <button class="sh-btn sh-btn--sm" :disabled="loading" @click="emit('refresh')">
        {{ loading ? '同步中…' : '刷新' }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    eyebrow?: string
    generatedAt?: string
    loading?: boolean
  }>(),
  {
    eyebrow: 'STUHELPER / MODERATION',
    generatedAt: '',
    loading: false,
  },
)

const emit = defineEmits<{ refresh: [] }>()

function formatSyncTime(input: string) {
  if (!input) return ''
  const date = new Date(input)
  if (Number.isNaN(date.getTime())) return input
  const hh = String(date.getHours()).padStart(2, '0')
  const mm = String(date.getMinutes()).padStart(2, '0')
  const ss = String(date.getSeconds()).padStart(2, '0')
  return `${hh}:${mm}:${ss}`
}
</script>
