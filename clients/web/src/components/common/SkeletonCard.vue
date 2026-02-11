<template>
  <div class="skeleton-card" :class="variant">
    <div v-if="variant === 'review'" class="skeleton-review">
      <div class="skeleton-header">
        <div class="skeleton-line title" />
        <div class="skeleton-line subtitle" />
      </div>
      <div class="skeleton-body">
        <div class="skeleton-line" />
        <div class="skeleton-line" />
        <div class="skeleton-line short" />
      </div>
      <div class="skeleton-footer">
        <div class="skeleton-tags">
          <div class="skeleton-tag" />
          <div class="skeleton-tag" />
          <div class="skeleton-tag" />
        </div>
      </div>
    </div>

    <div v-else-if="variant === 'course'" class="skeleton-course">
      <div class="skeleton-line title" />
      <div class="skeleton-line subtitle" />
      <div class="skeleton-line short" />
    </div>

    <div v-else class="skeleton-default">
      <div class="skeleton-line" />
      <div class="skeleton-line" />
      <div class="skeleton-line short" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  variant?: 'review' | 'course' | 'default'
}>()
</script>

<style scoped>
.skeleton-card {
  background: transparent;
  border-radius: 0;
  padding: var(--space-4) 0;
  border-bottom: 1px solid var(--border);
}

.skeleton-line {
  height: 16px;
  background: linear-gradient(
    90deg,
    var(--bg-elevated) 25%,
    var(--bg-card) 50%,
    var(--bg-elevated) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-2);
}

.skeleton-line.title {
  height: 20px;
  width: 60%;
}

.skeleton-line.subtitle {
  height: 14px;
  width: 40%;
}

.skeleton-line.short {
  width: 30%;
}

.skeleton-header {
  margin-bottom: var(--space-3);
}

.skeleton-body {
  margin-bottom: var(--space-3);
}

.skeleton-tags {
  display: flex;
  gap: var(--space-2);
}

.skeleton-tag {
  width: 60px;
  height: 24px;
  background: linear-gradient(
    90deg,
    var(--bg-elevated) 25%,
    var(--bg-card) 50%,
    var(--bg-elevated) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-sm);
}

@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}
</style>
