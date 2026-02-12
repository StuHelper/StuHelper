<template>
  <div class="flex items-center gap-1 p-1 bg-bg-card border border-border rounded-full shadow-sm" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      class="px-4 py-1.5 text-sm font-medium text-text-muted rounded-full transition-all duration-base ease-smooth cursor-pointer whitespace-nowrap hover:text-text-primary"
      :class="modelValue === tab.value && 'bg-gradient-to-br from-primary to-accent text-text-inverse shadow-[0_2px_8px_rgba(var(--brand-primary-rgb),0.2)]'"
      role="tab"
      :aria-selected="modelValue === tab.value"
      @click="$emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  tabs: Array<{ label: string; value: string }>
  modelValue: string
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()
</script>
