<template>
  <div class="my-votes">
    <div v-if="loading" class="loading-list">
      <SkeletonCard v-for="i in 3" :key="i" variant="review" />
    </div>

    <template v-else-if="votes.length > 0">
      <ReviewCard
        v-for="(review, index) in votes"
        :key="review.id"
        :review="review"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-item"
      />

      <div v-if="total > votes.length" class="load-more">
        <button @click="loadMore" :disabled="loadingMore">
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
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import ReviewCard from '@/components/review/ReviewCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const store = useUserStore()
const loading = ref(true)
const loadingMore = ref(false)
const page = ref(1)

const votes = computed(() => store.myVotes)
const total = computed(() => store.myVotesTotal)

onMounted(async () => {
  try {
    await store.fetchMyVotes(1)
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    page.value++
    await store.fetchMyVotes(page.value)
  } finally {
    loadingMore.value = false
  }
}
</script>

<style scoped>
.my-votes {
  display: flex;
  flex-direction: column;
}

.loading-list {
  display: flex;
  flex-direction: column;
}

.animate-item {
  animation: fadeInUp 0.4s var(--ease-out) forwards;
  opacity: 0;
}

.load-more {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.load-more button {
  padding: var(--space-2) var(--space-6);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.load-more button:hover:not(:disabled) {
  border-color: var(--text-primary);
  color: var(--text-primary);
}
</style>
