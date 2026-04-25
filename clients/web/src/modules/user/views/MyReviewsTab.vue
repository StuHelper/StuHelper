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
        :is-own-review="true"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-fade-in-up opacity-0"
        @deleted="handleDeleted"
        @updated="handleUpdated"
      />

      <div v-if="total > reviews.length" class="flex justify-center p-4">
        <button
          class="px-6 py-2 bg-transparent rounded-sm text-text-secondary text-sm cursor-pointer transition-all duration-fast hover:not-disabled:border-text-primary hover:not-disabled:text-text-primary"
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
import { api } from '@/api'
import type { Review } from '@stuhelper/shared/review'
import { normalizeReviews } from '@stuhelper/shared/review'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
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
    const res = await api.user.getMyReviews(1, 10)
    reviews.value = normalizeReviews(res.data?.data?.list)
    total.value = res.data?.data?.total || 0
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    page.value++
    const res = await api.user.getMyReviews(page.value, 10)
    reviews.value = [...reviews.value, ...normalizeReviews(res.data?.data?.list)]
  } finally {
    loadingMore.value = false
  }
}

const handleDeleted = (id: string) => {
  reviews.value = reviews.value.filter(r => r.id !== id)
  total.value = Math.max(0, total.value - 1)
}

const handleUpdated = (id: string, content: string) => {
  reviews.value = reviews.value.map(r =>
    r.id === id ? { ...r, content } : r,
  )
}
</script>
