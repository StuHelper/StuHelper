<script setup lang="ts">
import { ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapListData } from '@/api/result'
import { usePagedList } from '@/composables/usePagedList'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { formatDateTime } from '@/utils/format'
import { DEFAULT_PAGE_SIZE } from '@/config/pagination'

const authStore = useAuthStore()
const t = translate
const lastLoadedAt = ref(0)
const STALE_MS = 30_000
const {
  items: favorites,
  loading,
  loadingMore,
  hasMore,
  refresh: refreshFavorites,
  loadMore,
} = usePagedList<components['schemas']['FavoriteCourse']>({
  pageSize: DEFAULT_PAGE_SIZE,
  async fetchPage(page, pageSize) {
    const result = await api.user.getMyFavorites(page, pageSize)
    return unwrapListData<components['schemas']['FavoriteCourse']>(result)
  },
  onError(error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('user.favorites.loadFailed'),
      icon: 'none',
    })
  },
})

async function loadFavorites() {
  if (!(await authStore.requireAuth(t('user.favorites.requireAuth')))) return
  if (await refreshFavorites()) {
    lastLoadedAt.value = Date.now()
  }
}

function openCourse(id: number) {
  uni.navigateTo({ url: `/pages/course/detail?id=${id}` })
}

onShow(() => {
  setPageTitle('common.pageTitles.myFavorites')
  if (Date.now() - lastLoadedAt.value < STALE_MS) return
  void loadFavorites()
})
</script>

<template>
  <scroll-view class="page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="favorites.length === 0" class="state-card"><text>{{ t('user.favorites.empty') }}</text></view>
    <view v-else class="list-wrap">
      <A11yButton
        v-for="course in favorites"
        :key="course.id"
        class="card"
        :data-testid="`uni-user-favorite-card-${course.id}`"
        @tap="openCourse(course.id)"
      >
        <text class="title">{{ course.name }}</text>
        <text class="meta">
          {{ course.departmentName || t('common.departmentFallback', { id: course.departmentID }) }}
          · {{ t('common.reviewCount', { count: course.reviewCount }) }}
          · {{ t('common.favoritedAt', { value: formatDateTime(course.favoritedAt) }) }}
        </text>
        <text class="content">
          {{ t('common.creditValue', { value: course.credits }) }} ·
          {{ course.category || t('common.unclassified') }}
        </text>
      </A11yButton>
      <A11yButton v-if="hasMore" class="load-more" data-testid="uni-user-favorites-load-more" :disabled="loadingMore" @tap="loadMore">
        {{ loadingMore ? t('common.loading') : t('common.loadMore') }}
      </A11yButton>
    </view>
  </scroll-view>
</template>

<style scoped>
.page { min-height: 100vh; background: #f8fafc; }
.state-card { margin: 24rpx; padding: 40rpx; background: #fff; border-radius: 24rpx; text-align: center; color: #64748b; }
.list-wrap { padding: 24rpx; }
.card { display: block; width: 100%; margin-bottom: 18rpx; padding: 28rpx; border: 0; background: #fff; border-radius: 24rpx; box-shadow: 0 10rpx 30rpx rgba(15,23,42,.05); text-align: left; line-height: inherit; }
.title { display: block; font-size: 30rpx; font-weight: 700; color: #0f172a; }
.meta { display: block; margin-top: 10rpx; font-size: 22rpx; color: #64748b; }
.content { display: block; margin-top: 14rpx; font-size: 26rpx; line-height: 1.7; color: #334155; }
.load-more { display: block; width: 100%; padding: 28rpx; border: 0; background: transparent; text-align: center; color: #4f46e5; font-size: 26rpx; }
</style>
