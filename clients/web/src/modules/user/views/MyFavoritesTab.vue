<template>
  <div class="grid grid-cols-[repeat(auto-fill,minmax(280px,1fr))] gap-4">
    <div v-if="loading" class="contents">
      <SkeletonCard v-for="i in 3" :key="i" variant="course" />
    </div>

    <EmptyState
      v-else-if="errorMessage"
      class="col-span-full mx-auto w-full max-w-[420px]"
      :title="t('common.loadFailed')"
      :description="errorMessage"
    >
      <template #action>
        <button
          type="button"
          class="inline-flex items-center justify-center rounded-sm bg-text-primary px-4 py-2 text-sm font-medium text-bg-base transition-colors duration-fast hover:bg-accent hover:text-white"
          @click="loadInitial"
        >
          {{ t('common.actions.retry') }}
        </button>
      </template>
    </EmptyState>

    <template v-else-if="favorites.length > 0">
      <CourseCard
        v-for="(course, index) in favorites"
        :key="course.id"
        :course="course"
        :style="{ animationDelay: `${index * 50}ms` }"
        class="animate-fade-in-up opacity-0"
      />

      <div v-if="total > favorites.length || loadMoreError" class="col-span-full flex flex-col items-center gap-3 p-4">
        <p v-if="loadMoreError" role="alert" class="m-0 text-sm text-danger">
          {{ loadMoreError }}
        </p>
        <button
          type="button"
          v-if="total > favorites.length"
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
      class="col-span-full mx-auto w-full max-w-[420px]"
      :title="t('user.favorites.empty')"
      :description="t('user.favorites.emptyDesc')"
    >
      <template #action>
        <router-link
          to="/courses"
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
import { getErrorMessage } from '@/api/errors'
import CourseCard from '@/components/business/review/CourseCard.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const store = useUserStore()
const loading = ref(true)
const loadingMore = ref(false)
const page = ref(1)
const errorMessage = ref('')
const loadMoreError = ref('')

const favorites = computed(() => store.myFavorites)
const total = computed(() => store.myFavoritesTotal)

async function loadInitial() {
  loading.value = true
  errorMessage.value = ''
  loadMoreError.value = ''
  try {
    await store.fetchMyFavorites(1)
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
  loadMoreError.value = ''
  const nextPage = page.value + 1
  try {
    await store.fetchMyFavorites(nextPage)
    page.value = nextPage
  } catch (err) {
    loadMoreError.value = getErrorMessage(err, t('common.loadFailed'))
  } finally {
    loadingMore.value = false
  }
}
</script>
