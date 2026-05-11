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
      <component :is="getIcon(level)" :size="24" :stroke-width="2.5" />
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
import { Angry, Frown, Meh, Smile, SmilePlus } from 'lucide-vue-next'
import { getRatingColor } from '@/design-system/rating'

const props = defineProps<{
  label?: string
  modelValue: number
  error?: string
  testIdPrefix?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

function getIcon(level: number) {
  switch (level) {
    case 1: return Angry
    case 2: return Frown
    case 3: return Meh
    case 4: return Smile
    case 5: return SmilePlus
    default: return Meh
  }
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
