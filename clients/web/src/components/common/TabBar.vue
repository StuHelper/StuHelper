<template>
  <div class="pill-tabs">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      class="pill-tab"
      :class="{ active: modelValue === tab.value }"
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

<style scoped>
.pill-tabs {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1);
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
}

.pill-tab {
  padding: var(--space-1-5) var(--space-3);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
  border-radius: var(--radius-full);
  transition: all var(--duration-base) var(--ease-smooth);
  cursor: pointer;
  white-space: nowrap;
}

.pill-tab:hover {
  color: var(--text-primary);
}

.pill-tab.active {
  background: var(--bg-card);
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
}
</style>
