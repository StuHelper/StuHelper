<template>
  <div class="sh-toolbar sh-queue-toolbar">
    <div class="sh-queue-toolbar__stats">
      <span v-for="item in stats" :key="item.label" class="sh-toolbar__count">
        {{ item.label }} {{ item.value }}
      </span>
    </div>
    <div class="sh-toolbar__spacer"></div>
    <div class="sh-queue-toolbar__controls">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    stats?: ReadonlyArray<{
      label: string
      value: string | number
    }>
  }>(),
  {
    stats: () => [],
  },
)
</script>

<style scoped>
.sh-queue-toolbar {
  gap: var(--sh-s-3);
  padding: 0;
}

.sh-queue-toolbar__stats {
  display: flex;
  align-items: center;
  gap: var(--sh-s-2);
  flex-wrap: wrap;
}

.sh-queue-toolbar__controls {
  display: flex;
  align-items: flex-end;
  gap: var(--sh-s-3);
  flex-wrap: wrap;
}
</style>
