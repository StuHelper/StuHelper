<template>
  <div class="flex items-center gap-1">
    <button
      v-for="level in 5"
      :key="level"
      type="button"
      :aria-label="buttonLabel(level)"
      :data-testid="testIdPrefix ? `${testIdPrefix}-${level}` : undefined"
      :title="buttonLabel(level)"
      class="emoji-btn p-2 m-0.5 rounded-full border-2 transition-all duration-200 cursor-pointer"
      :class="[
        modelValue === level
          ? 'scale-[1.2] selected'
          : 'border-transparent opacity-40 hover:opacity-70 hover:scale-110',
      ]"
      :style="modelValue === level ? { color: getRatingColor(level), borderColor: getRatingColor(level) } : { color: 'var(--color-text-muted)' }"
      @click="toggle(level)"
    >
      <svg class="size-6 block" viewBox="0 0 512 512" fill="currentColor" aria-hidden="true">
        <path :d="getFacePath(level)" />
      </svg>
    </button>
    <span
      v-if="error"
      class="text-xs ml-2 text-danger animate-fade-in"
    >
      {{ error }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { getRatingColor } from '@/design-system/rating'
import { getRatingFacePath } from '@/modules/review/ratingFaces'

const props = defineProps<{
  label?: string
  modelValue: number
  error?: string
  testIdPrefix?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

function getFacePath(level: number) {
  return getRatingFacePath(level)
}

function toggle(level: number) {
  emit('update:modelValue', props.modelValue === level ? 0 : level)
}

function buttonLabel(level: number) {
  return props.label ? `${props.label} ${level}` : String(level)
}
</script>

<style scoped>
.emoji-btn.selected {
  background-color: var(--color-bg-elevated);
  box-shadow: var(--shadow-glow-primary);
}
</style>
