<template>
  <button
    class="inline-flex items-center gap-1 py-2 px-3 bg-transparent border border-border rounded-full text-text-muted cursor-pointer transition-all duration-fast disabled:opacity-60 disabled:cursor-not-allowed"
    :class="isFavorited
      ? 'border-accent text-accent bg-accent/[0.08] [&>.heart-icon]:animate-[heartBeat_0.4s_ease]'
      : 'hover:enabled:border-accent hover:enabled:text-accent hover:enabled:bg-accent/[0.06]'"
    :disabled="loading"
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
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Heart } from 'lucide-vue-next'
import { useUserStore } from '@/stores/user'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  courseID: number
  showText?: boolean
}>(), {
  showText: true
})

const userStore = useUserStore()
const loading = ref(false)
const error = ref('')

const isFavorited = computed(() => userStore.isFavorited(props.courseID))

const handleClick = async () => {
  loading.value = true
  error.value = ''
  try {
    await userStore.toggleFavorite(props.courseID)
  } catch {
    error.value = t('review.favorite.failed')
    // 3秒后清除错误
    setTimeout(() => { error.value = '' }, 3000)
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
