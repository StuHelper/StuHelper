<template>
  <div class="flex flex-col gap-4">
    <!-- 排序切换 -->
    <div class="flex items-center justify-between gap-3 pb-2">
      <TabBar
        :tabs="sortTabs"
        :model-value="activeSort"
        @update:model-value="handleSortChange"
      />
    </div>

    <!-- 首次加载骨架屏 -->
    <div v-if="isFirstLoad && loading" class="flex flex-col gap-4" aria-busy="true" role="status" :aria-label="t('review.hub.loading')">
      <SkeletonCard v-for="i in 3" :key="i" variant="review" />
    </div>

    <!-- 错误状态（首次加载失败，无已有数据） -->
    <div v-else-if="error && reviews.length === 0" class="text-center py-8 px-4 text-text-muted text-sm" role="alert" aria-live="polite">
      <p class="mb-3">{{ error }}</p>
      <button
        class="text-primary font-medium cursor-pointer underline underline-offset-2 hover:text-primary-dark"
        @click="handleRetry"
      >
        {{ t('review.hub.retry') }}
      </button>
    </div>

    <!-- 空状态 -->
    <EmptyState
      v-else-if="!loading && reviews.length === 0"
      :title="t('review.hub.feedEmpty')"
      role="status"
    />

    <!-- Feed 列表 + 底部状态 -->
    <template v-else>
      <div class="flex flex-col gap-4">
        <ReviewCard
          v-for="review in reviews"
          :key="review.id"
          :review="review"
          @moderated="() => loadReviews(true)"
        />
      </div>

      <!-- 翻页错误（已有数据时） -->
      <div v-if="error" class="text-center py-8 px-4 text-text-muted text-sm" role="alert" aria-live="polite">
        <p class="mb-3">{{ error }}</p>
        <button
          class="text-primary font-medium cursor-pointer underline underline-offset-2 hover:text-primary-dark"
          @click="handleRetry"
        >
          {{ t('review.hub.retry') }}
        </button>
      </div>

      <!-- 加载更多 -->
      <InfiniteScroll
        v-else
        :loading="loading"
        :has-more="hasMore"
        @load-more="loadMore"
      />
    </template>

  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import type { Review } from '@/types/review'
import { toReviews } from '@/types/review'
import TabBar from '@/components/common/TabBar.vue'
import ReviewCard from '@/components/business/review/ReviewCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import InfiniteScroll from '@/components/common/InfiniteScroll.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'

const { t } = useI18n()

const reviews = ref<Review[]>([])
const loading = ref(false)
const hasMore = ref(true)
const error = ref<string | null>(null)
const page = ref(1)
const pageSize = 10
const activeSort = ref('time')
const isFirstLoad = ref(true)

// H-46: 用于取消进行中的请求，防止组件卸载后状态更新
let abortController: AbortController | null = null
// M-109: 请求版本号，防止排序切换后旧请求覆盖新结果
let requestVersion = 0

const sortTabs = computed(() => [
  { label: t('review.filters.latest'), value: 'time' },
  { label: t('review.filters.hottest'), value: 'likes' },
  { label: t('review.filters.featured'), value: 'rating' }
])

async function loadReviews(reset = false) {
  if (loading.value && !reset) return

  // H-46: 取消上一次进行中的请求
  if (abortController) {
    abortController.abort()
  }
  abortController = new AbortController()
  const signal = abortController.signal
  // M-109: 版本号递增，防止排序切换后旧请求覆盖新结果
  const currentVersion = ++requestVersion

  if (reset) {
    page.value = 1
    reviews.value = []
    hasMore.value = true
  }

  loading.value = true
  error.value = null
  try {
    const res = await api.review.getLatestReviews({ page: page.value, pageSize, sort: activeSort.value as 'time' | 'likes' | 'rating' })
    // 请求被取消或版本过期则忽略结果
    if (signal.aborted || currentVersion !== requestVersion) return
    const list = toReviews(res.data?.data?.list || [])
    if (reset) {
      reviews.value = list
    } else {
      // M-61: 去重：分页间隙可能导致重复条目
      const existingIDs = new Set(reviews.value.map((r: Review) => r.id))
      reviews.value.push(...list.filter((r: Review) => !existingIDs.has(r.id)))
    }
    hasMore.value = list.length >= pageSize
    page.value++
  } catch (err) {
    if (signal.aborted || currentVersion !== requestVersion) return
    error.value = t('review.hub.loadFailed')
  } finally {
    if (!signal.aborted && currentVersion === requestVersion) {
      loading.value = false
      isFirstLoad.value = false
    }
  }
}

function loadMore() {
  loadReviews()
}

function handleSortChange(sort: string) {
  activeSort.value = sort
  loadReviews(true)
}

// M-11 & L-06: 重试时清除错误状态，不重置 hasMore，直接重试当前页
function handleRetry() {
  error.value = null
  loadReviews()
}

onMounted(() => {
  loadReviews()
})

onUnmounted(() => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
})
</script>
