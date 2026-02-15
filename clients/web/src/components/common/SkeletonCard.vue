<template>
  <div role="status" aria-label="Loading" class="bg-bg-card border border-border rounded-xl p-5 shadow-card">
    <div v-if="variant === 'review'" class="flex flex-col gap-3">
      <!-- 头部 -->
      <div class="flex flex-col gap-1">
        <div class="skeleton-line h-5 w-3/5" />
        <div class="skeleton-line h-3.5 w-2/5" />
      </div>
      <!-- 内容 -->
      <div>
        <div class="skeleton-line" />
        <div class="skeleton-line" />
        <div class="skeleton-line w-[30%]" />
      </div>
      <!-- 标签 -->
      <div class="flex gap-2">
        <div class="skeleton-tag" />
        <div class="skeleton-tag" />
        <div class="skeleton-tag" />
      </div>
    </div>

    <div v-else-if="variant === 'course'">
      <div class="skeleton-line h-5 w-3/5" />
      <div class="skeleton-line h-3.5 w-2/5" />
      <div class="skeleton-line w-[30%]" />
    </div>

    <div v-else>
      <div class="skeleton-line" />
      <div class="skeleton-line" />
      <div class="skeleton-line w-[30%]" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  variant?: 'review' | 'course' | 'default'
}>()
</script>

<style scoped>
.skeleton-line {
  height: 16px;
  background: linear-gradient(
    90deg,
    var(--color-bg-secondary) 25%,
    var(--color-bg-hover) 50%,
    var(--color-bg-secondary) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 9999px;
  margin-bottom: 0.5rem;
}

.skeleton-tag {
  width: 60px;
  height: 24px;
  background: linear-gradient(
    90deg,
    var(--color-bg-secondary) 25%,
    var(--color-bg-hover) 50%,
    var(--color-bg-secondary) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 9999px;
}
</style>
