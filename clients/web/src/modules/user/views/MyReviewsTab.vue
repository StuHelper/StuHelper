<template>
  <div class="flex flex-col">
    <div v-if="loading" class="flex flex-col">
      <SkeletonCard v-for="i in 3" :key="i" variant="review" />
    </div>

    <EmptyState
      v-else-if="errorMessage"
      :title="t('common.loadFailed')"
      :description="errorMessage"
    >
      <template #action>
        <button
          class="inline-flex items-center justify-center rounded-sm bg-text-primary px-4 py-2 text-sm font-medium text-bg-base transition-colors duration-fast hover:bg-accent hover:text-white"
          @click="loadInitial"
        >
          {{ t('common.actions.retry') }}
        </button>
      </template>
    </EmptyState>

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
import { getErrorMessage } from '@/api/errors'
import type { Review } from '@stuhelper/shared/review'
import { readReviewPagePayload } from '@/modules/review/reviewListPayload'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const loading = ref(true)
const loadingMore = ref(false)
const reviews = ref<Review[]>([])
const total = ref(0)
const page = ref(1)
const errorMessage = ref('')

async function loadInitial() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await api.user.getMyReviews(1, 10)
    const pageData = readReviewPagePayload(res.data?.data)
    reviews.value = pageData.list
    total.value = pageData.total
    page.value = 1
  } catch (err) {
    errorMessage.value = getErrorMessage(err, t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadInitial)

const loadMore = async () => {
  loadingMore.value = true
  errorMessage.value = ''
  const nextPage = page.value + 1
  try {
    const res = await api.user.getMyReviews(nextPage, 10)
    const pageData = readReviewPagePayload(res.data?.data)
    reviews.value = [...reviews.value, ...pageData.list]
    total.value = pageData.total
    page.value = nextPage
  } catch (err) {
    errorMessage.value = getErrorMessage(err, t('common.loadFailed'))
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
