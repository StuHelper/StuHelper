<template>
  <nav class="sh-policy-nav" aria-label="策略分类">
    <button
      v-for="item in items"
      :key="item.id"
      type="button"
      class="sh-policy-nav__item"
      :data-active="item.id === activeId"
      @click="handleSelect(item.id)"
    >
      <span class="sh-policy-nav__label">{{ item.label }}</span>
      <span class="sh-policy-nav__count">{{ item.count }}</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import type { PolicyCategoryDefinition, PolicyCategoryId } from '../../policy/categories'

interface PolicyCategoryNavItem extends PolicyCategoryDefinition {
  count: number
}

const props = defineProps<{
  items: readonly PolicyCategoryNavItem[]
  activeId: PolicyCategoryId
}>()

const emit = defineEmits<{
  select: [category: PolicyCategoryId]
}>()

function handleSelect(category: PolicyCategoryId) {
  if (category === props.activeId) return
  emit('select', category)
}
</script>

<style scoped>
.sh-policy-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sh-policy-nav__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sh-s-3);
  min-height: var(--sh-control-h);
  width: 100%;
  padding: 0 var(--sh-s-3);
  border: 0;
  border-radius: var(--sh-r-2);
  background: transparent;
  color: var(--sh-fg-1);
  cursor: pointer;
  text-align: left;
  transition:
    background var(--sh-dur-fast) var(--sh-ease),
    color var(--sh-dur-fast) var(--sh-ease);
}

.sh-policy-nav__item:hover {
  background: var(--sh-surface-hover);
  color: var(--sh-fg);
}

.sh-policy-nav__item[data-active='true'] {
  background: var(--sh-primary-soft);
  color: var(--sh-primary);
}

.sh-policy-nav__label {
  min-width: 0;
  font-size: var(--sh-t-meta);
  font-weight: var(--sh-w-medium);
}

.sh-policy-nav__count {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--sh-r-full);
  background: var(--sh-surface-1);
  color: inherit;
  display: inline-grid;
  place-items: center;
  font-size: var(--sh-t-meta);
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 960px) {
  .sh-policy-nav {
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .sh-policy-nav__item {
    min-width: 140px;
  }
}
</style>
