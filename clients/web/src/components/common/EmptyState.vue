<template>
  <div class="flex flex-col items-center justify-center py-8 px-4 text-center">
    <div class="w-16 h-16 text-text-muted mb-4 opacity-30">
      <slot name="icon">
        <svg class="w-full h-full" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
      </slot>
    </div>
    <p class="text-base font-medium text-text-secondary m-0 mb-2">{{ title }}</p>
    <p v-if="description" class="text-sm text-text-muted m-0 max-w-[300px]">{{ description }}</p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
}>()
</script>
