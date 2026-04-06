<template>
  <div
    class="flex items-center justify-between px-3 py-2 rounded-sm text-sm"
    :class="{ 'opacity-70': saving }"
  >
    <div class="flex items-center">
      <span v-if="saving" class="flex items-center gap-1 text-text-muted">
        <span class="inline-block w-3.5 h-3.5 border-2 border-border border-t-accent rounded-full animate-spin" />
        {{ t('review.draft.saving') }}
      </span>
      <span v-else-if="lastSaved" class="flex items-center gap-1 text-text-muted">
        <Check class="w-3.5 h-3.5 text-rating-4" />
        {{ t('review.draft.saved') }} {{ formatTime(lastSaved) }}
      </span>
    </div>
    <div v-if="hasDraft && !saving" class="flex gap-2">
      <button
        class="px-2 py-1 text-xs rounded-sm cursor-pointer bg-text-primary text-bg-base border-none transition-all duration-fast ease-out hover:bg-accent hover:text-white"
        @click="$emit('restore')"
      >
        {{ t('review.draft.restore') }}
      </button>
      <button
        class="px-2 py-1 text-xs rounded-sm cursor-pointer bg-transparent text-text-muted transition-all duration-fast ease-out hover:border-text-primary hover:text-text-primary"
        @click="$emit('delete')"
      >
        {{ t('common.actions.delete') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Check } from 'lucide-vue-next'

const { t, locale } = useI18n()

defineProps<{
  saving?: boolean
  hasDraft?: boolean
  lastSaved?: Date | null
}>()

defineEmits<{
  restore: []
  delete: []
}>()

const formatTime = (date: Date) => {
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return t('common.time.justNow')
  if (diff < 3600000) return t('common.time.minutesAgo', { n: Math.floor(diff / 60000) })
  return date.toLocaleTimeString(locale.value, { hour: '2-digit', minute: '2-digit' })
}
</script>
