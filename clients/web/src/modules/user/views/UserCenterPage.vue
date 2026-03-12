<template>
  <div class="max-w-[800px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <header class="flex items-center gap-4 mb-6">
      <div class="p-[3px] bg-gradient-to-br from-primary to-accent rounded-full shrink-0">
        <div class="size-14 bg-bg-card rounded-full flex items-center justify-center text-text-muted">
          <User class="size-7" />
        </div>
      </div>
      <div>
        <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">{{ t('user.center') }}</h1>
      </div>
    </header>

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
import { User } from 'lucide-vue-next'
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
