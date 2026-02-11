<template>
  <button
    class="favorite-btn"
    :class="{ active: isFavorited }"
    :disabled="loading"
    @click="handleClick"
  >
    <svg
      class="heart-icon"
      viewBox="0 0 24 24"
      :fill="isFavorited ? 'currentColor' : 'none'"
      stroke="currentColor"
      stroke-width="2"
    >
      <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
    </svg>
    <span v-if="showText" class="text">
      {{ isFavorited ? t('review.favorite.favorited') : t('review.favorite.add') }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
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
.favorite-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-2) var(--space-3);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.favorite-btn:hover:not(:disabled) {
  border-color: var(--brand-accent);
  color: var(--brand-accent);
}

.favorite-btn.active {
  border-color: var(--brand-accent);
  color: var(--brand-accent);
}

.favorite-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.heart-icon {
  width: 18px;
  height: 18px;
  transition: transform var(--duration-fast) ease;
}

.favorite-btn.active .heart-icon {
  animation: heartBeat 0.4s ease;
}

.text {
  font-size: var(--text-sm);
}

@keyframes heartBeat {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.3); }
}
</style>
