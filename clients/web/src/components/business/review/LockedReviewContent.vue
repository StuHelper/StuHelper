<template>
  <div class="rounded-lg border border-border-light bg-bg-elevated/60 px-4 py-4">
    <p
      v-if="previewLine"
      class="m-0 text-sm leading-relaxed text-text-secondary break-words"
      v-text="previewLine"
    />
    <div class="relative mt-3 overflow-hidden rounded-md" aria-hidden="true">
      <div class="space-y-2 select-none blur-[3px]">
        <div
          v-for="line in lockedLines"
          :key="line"
          class="h-3 rounded-full bg-text-muted/25"
          :class="line"
        />
      </div>
      <div class="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-bg-elevated/55 to-bg-elevated/90" />
    </div>
    <div class="mt-3 flex flex-col items-center gap-1.5 text-center">
      <Lock :size="18" class="text-text-muted" />
      <span class="text-xs font-medium text-text-secondary">{{ message }}</span>
      <button
        class="text-xs font-medium text-primary hover:underline"
        type="button"
        @click="$emit('action')"
      >
        {{ actionLabel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Lock } from 'lucide-vue-next'

defineProps<{
  previewLine?: string
  message: string
  actionLabel: string
}>()

defineEmits<{
  action: []
}>()

const lockedLines = ['w-full', 'w-11/12', 'w-4/5']
</script>
