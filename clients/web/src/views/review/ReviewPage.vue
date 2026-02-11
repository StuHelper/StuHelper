<template>
  <div class="review-page">
    <!-- 院系侧栏（桌面端为左侧栏，移动端为顶部水平滚动条） -->
    <div class="review-sidebar">
      <DepartmentSidebar />
    </div>

    <!-- 右侧内容 -->
    <main class="review-main">
      <router-view v-if="hasChildRoute" />
      <ReviewFeed v-else />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import DepartmentSidebar from '@/components/review/DepartmentSidebar.vue'
import ReviewFeed from '@/components/review/ReviewFeed.vue'

const route = useRoute()

const hasChildRoute = computed(() => {
  return route.matched.length > 1 && route.name !== 'review'
})
</script>

<style scoped>
.review-page {
  max-width: var(--max-width);
  margin: 0 auto;
  padding: var(--space-6);
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: var(--space-6);
  min-height: calc(100vh - var(--navbar-height) - var(--gradient-bar-height));
}

.review-sidebar {
  position: sticky;
  top: calc(var(--gradient-bar-height) + var(--navbar-height) + var(--space-6));
  align-self: start;
}

.review-main {
  min-width: 0;
}

@media (max-width: 1023px) {
  .review-page {
    grid-template-columns: 1fr;
    padding: var(--space-4);
  }

  .review-sidebar {
    position: static;
    order: -1;
    overflow-x: auto;
  }
}
</style>
