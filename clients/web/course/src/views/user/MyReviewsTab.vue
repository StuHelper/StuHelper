<template>
  <div class="flex flex-col">
    <div v-if="loading" class="flex flex-col">
      <SkeletonCard v-for="i in 3" :key="i" variant="review" />
    </div>

    <template v-else-if="reviews.length > 0">
      <ReviewCard
        v-for="(review, index) in reviews"
        :key="review.id"
        :review="review"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-fade-in-up opacity-0"
      />

      <div v-if="total > reviews.length" class="flex justify-center p-4">
        <button
          class="px-6 py-2 bg-transparent border border-border rounded-sm text-text-secondary text-sm cursor-pointer transition-all duration-fast hover:not-disabled:border-text-primary hover:not-disabled:text-text-primary"
          @click="loadMore"
          :disabled="loadingMore"
        >
          {{ loadingMore ? t('common.actions.loading') : t('common.actions.loadMore') }}
        </button>
      </div>
    </template>

    <EmptyState
      v-else
      :title="t('user.reviews.empty')"
      :description="t('user.reviews.emptyDesc')"
    >
      <template #action>
        <router-link
          to="/"
          class="inline-block px-4 py-2 bg-text-primary text-bg-base rounded-sm no-underline text-sm font-medium transition-all duration-fast hover:bg-accent hover:text-white"
        >
          {{ t('user.reviews.browseCourses') }}
        </router-link>
      </template>
    </EmptyState>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMyReviews } from '@/api/user'
import type { Review } from '@/types/review'
import ReviewCard from '@/components/review/ReviewCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const loading = ref(true)
const loadingMore = ref(false)
const reviews = ref<Review[]>([])
const total = ref(0)
const page = ref(1)

onMounted(async () => {
  try {
    const res = await getMyReviews(1, 10)
    reviews.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    page.value++
    const res = await getMyReviews(page.value, 10)
    reviews.value = [...reviews.value, ...(res.data?.list || [])]
  } finally {
    loadingMore.value = false
  }
}
</script>
