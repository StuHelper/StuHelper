<template>
  <div class="flex flex-col">
    <div v-if="loading" class="flex flex-col">
      <SkeletonCard v-for="i in 3" :key="i" variant="review" />
    </div>

    <template v-else-if="votes.length > 0">
      <ReviewCard
        v-for="(review, index) in votes"
        :key="review.id"
        :review="review"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-fade-in-up opacity-0"
      />

      <div v-if="total > votes.length" class="flex justify-center p-4">
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
      :title="t('user.votes.empty')"
      :description="t('user.votes.emptyDesc')"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import type { Review } from '@/types/review'
import { toReviews } from '@/types/review'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const loading = ref(true)
const loadingMore = ref(false)
const votes = ref<Review[]>([])
const total = ref(0)
const page = ref(1)

onMounted(async () => {
  try {
    const res = await api.user.getMyVotes(1, 10)
    votes.value = toReviews(res.data?.data?.list || [])
    total.value = res.data?.data?.total || 0
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    page.value++
    const res = await api.user.getMyVotes(page.value, 10)
    votes.value = [...votes.value, ...toReviews(res.data?.data?.list || [])]
  } finally {
    loadingMore.value = false
  }
}
</script>
