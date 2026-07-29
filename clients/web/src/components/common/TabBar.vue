<template>
  <div class="flex max-w-full items-center gap-1 overflow-x-auto p-1 bg-bg-card rounded-full shadow-sm" role="tablist">
    <button
      type="button"
      v-for="(tab, index) in tabs"
      :key="tab.value"
      :ref="el => { if (el) tabRefs[index] = el as HTMLButtonElement }"
      class="shrink-0 px-4 py-1.5 text-sm font-medium text-text-muted rounded-full transition-all duration-base ease-spring cursor-pointer whitespace-nowrap hover:text-text-primary press-spring"
      :class="modelValue === tab.value && 'bg-gradient-to-br from-primary to-accent text-text-inverse shadow-glow-sm'"
      role="tab"
      :aria-selected="modelValue === tab.value"
      :aria-label="tab.label"
      :tabindex="modelValue === tab.value ? 0 : -1"
      @click="$emit('update:modelValue', tab.value)"
      @keydown="onKeydown"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onBeforeUpdate } from 'vue'

const props = defineProps<{
  tabs: Array<{ label: string; value: string }>
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const tabRefs = ref<HTMLButtonElement[]>([])

// 每次更新前重置 ref 数组，防止旧引用残留
onBeforeUpdate(() => {
  tabRefs.value = []
})

function onKeydown(e: KeyboardEvent) {
  const currentIndex = props.tabs.findIndex(t => t.value === props.modelValue)
  let nextIndex = -1

  if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % props.tabs.length
  } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
    nextIndex = (currentIndex - 1 + props.tabs.length) % props.tabs.length
  } else if (e.key === 'Home') {
    nextIndex = 0
  } else if (e.key === 'End') {
    nextIndex = props.tabs.length - 1
  }

  if (nextIndex >= 0) {
    e.preventDefault()
    emit('update:modelValue', props.tabs[nextIndex].value)
    tabRefs.value[nextIndex]?.focus()
  }
}
</script>
