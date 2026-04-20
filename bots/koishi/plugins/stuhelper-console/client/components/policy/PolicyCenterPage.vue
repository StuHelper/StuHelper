<template>
  <div id="sh-view-policy" class="sh-view sh-policy-center" role="tabpanel">
    <aside class="sh-policy-center__nav">
      <h1 class="sh-policy-center__nav-title">策略分类</h1>
      <PolicyCategoryNav
        :items="categories"
        :active-id="activeCategory"
        @select="emit('select-category', $event)"
      />
    </aside>

    <section class="sh-policy-center__main">
      <header class="sh-policy-center__header">
        <div class="sh-policy-center__header-copy">
          <h2 class="sh-policy-center__title">{{ activeItem.label }}</h2>
          <p class="sh-policy-center__description">{{ activeItem.description }}</p>
        </div>
        <span class="sh-tag sh-tag--neutral">{{ activeItem.count }} 项</span>
      </header>

      <div class="sh-policy-center__body">
        <slot :category="activeItem" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import {
  POLICY_CATEGORIES,
  type PolicyCategoryDefinition,
  type PolicyCategoryId,
} from '../../policy/categories'
import PolicyCategoryNav from './PolicyCategoryNav.vue'

interface PolicyCenterCategoryItem extends PolicyCategoryDefinition {
  count: number
}

const props = defineProps<{
  categories: readonly PolicyCenterCategoryItem[]
  activeCategory: PolicyCategoryId
}>()

const emit = defineEmits<{
  'select-category': [category: PolicyCategoryId]
}>()

const activeItem = computed(() => {
  const item = props.categories.find((entry) => entry.id === props.activeCategory)
  if (item) return item
  return {
    ...POLICY_CATEGORIES[0],
    count: 0,
  }
})
</script>

<style scoped>
.sh-policy-center {
  --sh-control-h: 32px;
  --sh-t-body: 12px;
  --sh-t-body-lg: 12px;
  --sh-t-section: 14px;

  display: grid;
  grid-template-columns: minmax(200px, 220px) minmax(0, 1fr);
  gap: var(--sh-s-4);
  align-items: start;
}

.sh-policy-center__nav,
.sh-policy-center__main {
  min-width: 0;
  background: var(--sh-surface-0);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-3);
  box-shadow: var(--sh-shadow-1);
}

.sh-policy-center__nav {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-4);
  padding: var(--sh-s-4);
}

.sh-policy-center__header-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.sh-policy-center__nav-title,
.sh-policy-center__title {
  font-size: 14px;
  font-weight: var(--sh-w-semibold);
  line-height: var(--sh-l-tight);
  letter-spacing: -0.01em;
}

.sh-policy-center__main {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sh-policy-center__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--sh-s-3);
  padding: var(--sh-s-4) var(--sh-s-5);
  border-bottom: 1px solid var(--sh-border);
}

.sh-policy-center__description {
  font-size: var(--sh-t-meta);
  color: var(--sh-fg-2);
  line-height: var(--sh-l-normal);
}

.sh-policy-center__body {
  min-width: 0;
  padding: var(--sh-s-4);
  background: var(--sh-surface-1);
}

@media (max-width: 960px) {
  .sh-policy-center {
    grid-template-columns: minmax(0, 1fr);
  }

  .sh-policy-center__nav,
  .sh-policy-center__header {
    padding: var(--sh-s-3) var(--sh-s-4);
  }

  .sh-policy-center__body {
    padding: var(--sh-s-3);
  }
}
</style>
