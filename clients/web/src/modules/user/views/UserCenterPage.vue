<template>
  <div class="max-w-[800px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <!-- Profile + verification status -->
    <ProfileSection />

    <nav class="mb-6">
      <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="handleTabChange" />
    </nav>

    <div class="animate-fade-in">
      <MyReviewsTab v-if="activeTab === 'reviews'" />
      <MyVotesTab v-else-if="activeTab === 'votes'" />
      <MyFavoritesTab v-else-if="activeTab === 'favorites'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'
import ProfileSection from './ProfileSection.vue'
import MyReviewsTab from './MyReviewsTab.vue'
import MyVotesTab from './MyVotesTab.vue'
import MyFavoritesTab from './MyFavoritesTab.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const tabItems = computed(() => [
  { value: 'reviews', label: t('user.myReviews') },
  { value: 'votes', label: t('user.myVotes') },
  { value: 'favorites', label: t('user.myFavorites') }
])

const activeTab = computed(() => {
  if (route.name === 'user-votes') return 'votes'
  if (route.name === 'user-favorites') return 'favorites'
  return 'reviews'
})

function handleTabChange(tab: string) {
  const routeName = tab === 'votes'
    ? 'user-votes'
    : tab === 'favorites'
      ? 'user-favorites'
      : 'user-reviews'

  router.push({ name: routeName })
}
</script>
