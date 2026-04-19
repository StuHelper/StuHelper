<template>
  <nav class="sh-tabs" role="tablist" :aria-label="ariaLabel">
    <button
      v-for="(item, index) in items"
      :key="item.id"
      type="button"
      role="tab"
      class="sh-tab"
      :aria-selected="item.id === modelValue"
      :aria-controls="`sh-view-${item.id}`"
      @click="onSelect(item.id)"
    >
      <span class="sh-tab__index">{{ String(index + 1).padStart(2, '0') }}</span>
      <span>{{ item.label }}</span>
      <span v-if="typeof item.count === 'number'" class="sh-tab__badge sh-num">
        {{ item.count }}
      </span>
    </button>
  </nav>
</template>

<script setup lang="ts">
export interface TabItem {
  id: string
  label: string
  count?: number
}

withDefaults(
  defineProps<{
    items: TabItem[]
    modelValue: string
    ariaLabel?: string
  }>(),
  {
    ariaLabel: '一级导航',
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

function onSelect(id: string) {
  emit('update:modelValue', id)
}
</script>
