<template>
  <div class="draft-indicator" :class="{ saving }">
    <div class="status">
      <span v-if="saving" class="saving-text">
        <span class="spinner" />
        {{ t('review.draft.saving') }}
      </span>
      <span v-else-if="lastSaved" class="saved-text">
        <svg class="check-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path d="M20 6L9 17l-5-5" stroke-width="2" />
        </svg>
        {{ t('review.draft.saved') }} {{ formatTime(lastSaved) }}
      </span>
    </div>
    <div v-if="hasDraft && !saving" class="actions">
      <button class="restore-btn" @click="$emit('restore')">
        {{ t('review.draft.restore') }}
      </button>
      <button class="delete-btn" @click="$emit('delete')">
        {{ t('common.actions.delete') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

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

<style scoped>
.draft-indicator {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
}

.status {
  display: flex;
  align-items: center;
}

.saving-text,
.saved-text {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  color: var(--text-muted);
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.check-icon {
  width: 14px;
  height: 14px;
  color: #22c55e;
}

.actions {
  display: flex;
  gap: var(--space-2);
}

.restore-btn,
.delete-btn {
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--duration-fast) ease;
}

.restore-btn {
  background: var(--text-primary);
  border: none;
  color: var(--bg-base);
}

.restore-btn:hover {
  background: var(--accent);
  color: white;
}

.delete-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
}

.delete-btn:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
