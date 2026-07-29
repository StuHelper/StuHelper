<template>
  <button
    type="button"
    v-ripple
    class="inline-flex items-center gap-1 py-2 px-3 bg-transparent rounded-full text-text-muted cursor-pointer transition-all duration-fast press-spring disabled:opacity-60 disabled:cursor-not-allowed"
    :class="isFavorited
      ? 'border border-accent text-accent bg-accent/[0.08] [&>.heart-icon]:animate-[heartBeat_0.4s_ease]'
      : 'border border-transparent hover:enabled:border-accent hover:enabled:text-accent hover:enabled:bg-accent/[0.06]'"
    :disabled="isLoading"
    :aria-label="isFavorited ? t('review.favorite.favorited') : t('review.favorite.add')"
    :aria-pressed="isFavorited"
    @click="handleClick"
  >
    <Heart
      class="heart-icon w-[18px] h-[18px] transition-transform duration-fast ease-out"
      :size="18"
      :fill="isFavorited ? 'currentColor' : 'none'"
    />
    <span v-if="showText" class="text-sm">
      {{ isFavorited ? t('review.favorite.favorited') : t('review.favorite.add') }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { Heart } from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const toast = useToast()
const router = useRouter()

const props = withDefaults(defineProps<{
  courseID: number
  showText?: boolean
}>(), {
  showText: true
})

const userStore = useUserStore()
const authStore = useAuthStore()
const loading = ref(false)

const favoriteState = computed(() => userStore.isFavorited(props.courseID))
const isFavorited = computed(() => favoriteState.value === true)
const isLoading = computed(() => {
  if (loading.value || authStore.bootstrapPending) return true
  if (authStore.isAuthenticated && !authStore.bootstrapCompleted) return true
  return authStore.isAuthenticated && favoriteState.value === undefined
})

watch(
  () => [authStore.bootstrapCompleted, authStore.isAuthenticated, props.courseID] as const,
  ([bootstrapCompleted, isAuthenticated]) => {
    if (!bootstrapCompleted || !isAuthenticated) return
    void userStore.ensureFavoriteStatus(props.courseID)
  },
  {
    immediate: true,
  },
)

const handleClick = async () => {
  if (isLoading.value) return
  if (!authStore.isAuthenticated) {
    await router.push({
      name: 'login',
      query: { redirect: router.currentRoute.value.fullPath },
    })
    return
  }

  loading.value = true
  try {
    await userStore.toggleFavorite(props.courseID)
  } catch (_error) { void _error;
    toast.error(t('review.favorite.failed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* heartBeat keyframe — 无法用纯 utility 表达 */
@keyframes heartBeat {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.3); }
}
</style>
