<template>
  <div class="sh-empty" :data-tone="tone">
    <p class="sh-empty__title">{{ title }}</p>
    <p v-if="body" class="sh-empty__body">{{ body }}</p>
    <div v-if="$slots.action" class="sh-btn-row" style="margin-top: 12px">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    body?: string
    tone?: 'default' | 'error'
  }>(),
  {
    body: '',
    tone: 'default',
  },
)
</script>
