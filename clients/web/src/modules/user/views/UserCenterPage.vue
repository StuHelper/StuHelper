<template>
  <div class="max-w-[800px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <nav class="mb-6">
      <TabBar :tabs="tabItems" :model-value="activeTab" @update:model-value="handleTabChange" />
    </nav>

    <div class="min-h-[248px]">
      <MyReviewsTab v-show="activeTab === 'reviews'" />
      <MyVotesTab v-show="activeTab === 'votes'" />
      <MyFavoritesTab v-show="activeTab === 'favorites'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import TabBar from '@/components/common/TabBar.vue'
import MyReviewsTab from './MyReviewsTab.vue'
import MyVotesTab from './MyVotesTab.vue'
import MyFavoritesTab from './MyFavoritesTab.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const tabItems = computed(() => [
  { value: 'reviews', label: t('user.myReviews') },
  { value: 'votes', label: t('user.myVotes') },
  { value: 'favorites', label: t('user.myFavorites') },
])

const activeTab = computed(() => {
  if (route.name === 'user-votes') return 'votes'
  if (route.name === 'user-favorites') return 'favorites'
  return 'reviews'
})

function handleTabChange(tab: string) {
  if (tab === activeTab.value) return

  const routeName = tab === 'votes'
    ? 'user-votes'
    : tab === 'favorites'
      ? 'user-favorites'
      : 'user-reviews'

  void router.replace({ name: routeName })
}
</script>
