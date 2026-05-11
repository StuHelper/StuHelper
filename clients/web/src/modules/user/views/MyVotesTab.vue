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
import { getErrorMessage } from '@/api/errors'
import type { Review } from '@stuhelper/shared/review'
import { normalizeReviews } from '@stuhelper/shared/review'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const loading = ref(true)
const loadingMore = ref(false)
const votes = ref<Review[]>([])
const total = ref(0)
const page = ref(1)
const errorMessage = ref('')

async function loadInitial() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await api.user.getMyVotes(1, 10)
    votes.value = normalizeReviews(res.data?.data?.list)
    total.value = res.data?.data?.total || 0
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
    const res = await api.user.getMyVotes(nextPage, 10)
    votes.value = [...votes.value, ...normalizeReviews(res.data?.data?.list)]
    page.value = nextPage
  } catch (err) {
    errorMessage.value = getErrorMessage(err, t('common.loadFailed'))
  } finally {
    loadingMore.value = false
  }
}
</script>
