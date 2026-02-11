<template>
  <div class="rating-dist">
    <div
      v-for="star in [5, 4, 3, 2, 1]"
      :key="star"
      class="rating-dist__row"
    >
      <span class="rating-dist__label font-mono">{{ star }}</span>
      <div class="rating-dist__track">
        <div
          class="rating-dist__fill"
          :style="{ width: `${getPercentage(star)}%` }"
          :class="`fill-${star}`"
        />
      </div>
      <span class="rating-dist__count font-mono">{{ getCount(star) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  distribution: Record<number, number>
}>()

const total = computed(() =>
  Object.values(props.distribution).reduce((a, b) => a + b, 0)
)

function getCount(star: number): number {
  return props.distribution[star] || 0
}

function getPercentage(star: number): number {
  if (total.value === 0) return 0
  return (getCount(star) / total.value) * 100
}
</script>

<style scoped>
.rating-dist {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.rating-dist__row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.rating-dist__label {
  font-size: var(--text-xs);
  font-weight: var(--weight-medium);
  color: var(--text-muted);
  width: 16px;
  text-align: center;
}

.rating-dist__track {
  flex: 1;
  height: 8px;
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.rating-dist__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width var(--duration-slower) var(--ease-out);
}

.fill-5 { background: var(--rating-5); }
.fill-4 { background: var(--rating-4); }
.fill-3 { background: var(--rating-3); }
.fill-2 { background: var(--rating-2); }
.fill-1 { background: var(--rating-1); }

.rating-dist__count {
  font-size: var(--text-xs);
  color: var(--text-muted);
  width: 28px;
  text-align: right;
}
</style>
