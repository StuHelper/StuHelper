<template>
  <div class="review-feed">
    <!-- 排序切换 -->
    <div class="feed-header">
      <TabBar
        :tabs="sortTabs"
        :model-value="activeSort"
        @update:model-value="handleSortChange"
      />
      <button class="post-btn" @click="showPostModal = true">
        {{ t('review.hub.postReview') }}
      </button>
    </div>

    <!-- Feed 列表 -->
    <div class="feed-list">
      <ReviewCard
        v-for="review in reviews"
        :key="review.id"
        :review="review"
      />
    </div>

    <!-- 空状态 -->
    <EmptyState
      v-if="!loading && reviews.length === 0"
      :title="t('review.hub.feedEmpty')"
    />

    <!-- 加载更多 -->
    <InfiniteScroll
      :loading="loading"
      :has-more="hasMore"
      @load-more="loadMore"
    />

    <!-- 发布模态框 -->
    <ReviewDialog
      :visible="showPostModal"
      @close="showPostModal = false"
      @posted="handlePosted"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getLatestReviews } from '@/api/review'
import type { Review } from '@/types/review'
import TabBar from '@/components/common/TabBar.vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import ReviewDialog from '@/components/review/ReviewDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'

const { t } = useI18n()

const reviews = ref<Review[]>([])
const loading = ref(false)
const hasMore = ref(true)
const page = ref(1)
const activeSort = ref('time')
const showPostModal = ref(false)

const sortTabs = computed(() => [
  { label: t('review.filters.latest'), value: 'time' },
  { label: t('review.filters.hottest'), value: 'likes' },
  { label: t('review.filters.featured'), value: 'rating' }
])

async function loadReviews(reset = false) {
  if (loading.value) return
  if (reset) {
    page.value = 1
    reviews.value = []
    hasMore.value = true
  }

  loading.value = true
  try {
    const res = await getLatestReviews(page.value, 10, activeSort.value)
    const list = res.data?.list || []
    if (reset) {
      reviews.value = list
    } else {
      reviews.value.push(...list)
    }
    hasMore.value = list.length >= 10
    page.value++
  } catch {
    hasMore.value = false
  } finally {
    loading.value = false
  }
}

function loadMore() {
  loadReviews()
}

function handleSortChange(sort: string) {
  activeSort.value = sort
  loadReviews(true)
}

function handlePosted() {
  loadReviews(true)
}

onMounted(() => {
  loadReviews()
})
</script>
