<template>
  <div class="user-center">
    <header class="user-header">
      <div class="avatar-ring">
        <div class="avatar-placeholder">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
            <circle cx="12" cy="7" r="4"/>
          </svg>
        </div>
      </div>
      <div class="user-info">
        <h1 class="user-name">{{ t('user.center') }}</h1>
      </div>
    </header>

    <nav class="tab-bar">
      <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="activeTab = $event" />
    </nav>

    <div class="tab-content">
      <MyReviewsTab v-if="activeTab === 'reviews'" />
      <MyVotesTab v-else-if="activeTab === 'votes'" />
      <MyFavoritesTab v-else-if="activeTab === 'favorites'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'
import MyReviewsTab from './MyReviewsTab.vue'
import MyVotesTab from './MyVotesTab.vue'
import MyFavoritesTab from './MyFavoritesTab.vue'

const { t } = useI18n()

const tabItems = computed(() => [
  { value: 'reviews', label: t('user.myReviews') },
  { value: 'votes', label: t('user.myVotes') },
  { value: 'favorites', label: t('user.myFavorites') }
])

const activeTab = ref('reviews')
</script>

<style scoped>
.user-center {
  max-width: 800px;
  margin: 0 auto;
  padding: var(--space-6);
  animation: fadeIn var(--duration-base) var(--ease-out);
}

.user-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

.avatar-ring {
  padding: 3px;
  background: var(--gradient-brand);
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.avatar-placeholder {
  width: 56px;
  height: 56px;
  background: var(--bg-card);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.avatar-placeholder svg {
  width: 28px;
  height: 28px;
}

.user-name {
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0;
}

.tab-bar {
  margin-bottom: var(--space-6);
}

.tab-content {
  animation: fadeIn var(--duration-base) var(--ease-out);
}

@media (max-width: 640px) {
  .user-center {
    padding: var(--space-4);
  }
}
</style>
