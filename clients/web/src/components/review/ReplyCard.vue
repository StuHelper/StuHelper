<template>
  <div
    class="p-3 mb-2 last:mb-0 bg-bg-secondary rounded-lg animate-fade-in-up"
    :class="reply.isOwner && 'bg-primary/[0.06] border-l-3 border-l-primary'"
  >
    <div class="text-sm text-text-primary leading-relaxed break-words">
      {{ reply.content }}
    </div>
    <div class="flex items-center justify-between mt-2">
      <span class="text-xs text-text-muted">{{ formatTime(reply.createdAt) }}</span>
      <button
        v-if="reply.isOwner"
        class="text-xs text-text-muted bg-transparent border-none cursor-pointer px-2 py-1 rounded-sm transition-all duration-fast hover:text-accent disabled:opacity-50 disabled:cursor-not-allowed"
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
  delete: [id: string]
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
