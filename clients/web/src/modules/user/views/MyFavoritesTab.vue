<template>
  <div class="grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4">
    <div v-if="loading" class="contents">
      <SkeletonCard v-for="i in 3" :key="i" variant="course" />
    </div>

    <template v-else-if="favorites.length > 0">
      <CourseCard
        v-for="(course, index) in favorites"
        :key="course.id"
        :course="course"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-fade-in-up opacity-0"
      />

      <div v-if="total > favorites.length" class="col-span-full flex justify-center p-4">
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
      :title="t('user.favorites.empty')"
      :description="t('user.favorites.emptyDesc')"
    >
      <template #action>
        <router-link
          to="/"
          class="inline-block px-4 py-2 bg-text-primary text-bg-base rounded-sm no-underline text-sm font-medium transition-all duration-fast hover:bg-accent hover:text-white"
        >
          {{ t('user.favorites.browseCourses') }}
        </router-link>
      </template>
    </EmptyState>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import CourseCard from '@/components/business/review/CourseCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const store = useUserStore()
const loading = ref(true)
const loadingMore = ref(false)
const page = ref(1)

const favorites = computed(() => store.myFavorites)
const total = computed(() => store.myFavoritesTotal)

onMounted(async () => {
  try {
    await store.fetchMyFavorites(1)
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    page.value++
    await store.fetchMyFavorites(page.value)
  } finally {
    loadingMore.value = false
  }
}
</script>
