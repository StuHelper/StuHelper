<template>
  <div class="reply-card" :class="{ 'is-owner': reply.isOwner }">
    <div class="reply-content">
      {{ reply.content }}
    </div>
    <div class="reply-footer">
      <span class="reply-time">{{ formatTime(reply.createdAt) }}</span>
      <button
        v-if="reply.isOwner"
        class="delete-btn"
        @click="handleDelete"
        :disabled="deleting"
      >
        {{ deleting ? t('common.actions.deleting') : t('common.actions.delete') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Reply } from '@/types/reply'
import { formatRelativeTime } from '@/utils/date'

const { t, locale } = useI18n()

const props = defineProps<{
  reply: Reply
}>()

const emit = defineEmits<{
  delete: [id: number]
}>()

const deleting = ref(false)

const formatTime = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)

const handleDelete = () => {
  if (!window.confirm(t('review.reply.deleteConfirm'))) {
    return
  }
  deleting.value = true
  emit('delete', props.reply.id)
  // 安全兜底：删除成功时组件会被卸载，失败时 3 秒后恢复按钮
  setTimeout(() => { deleting.value = false }, 3000)
}
</script>

<style scoped>
.reply-card {
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--border);
  animation: fadeInUp var(--duration-base) var(--ease-out);
}

.reply-card:last-child {
  border-bottom: none;
}

.reply-card.is-owner {
  padding-left: var(--space-3);
  border-left: 2px solid var(--brand-primary);
}

.reply-content {
  font-size: var(--text-sm);
  color: var(--text-primary);
  line-height: 1.6;
  word-break: break-word;
}

.reply-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: var(--space-2);
}

.reply-time {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.delete-btn {
  font-size: var(--text-xs);
  color: var(--text-muted);
  background: none;
  border: none;
  cursor: pointer;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  transition: all var(--duration-fast) ease;
}

.delete-btn:hover:not(:disabled) {
  color: var(--brand-accent);
}

.delete-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
