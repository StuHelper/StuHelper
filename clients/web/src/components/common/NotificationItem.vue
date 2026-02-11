<template>
  <div
    class="notification-item"
    :class="{ unread: !notification.isRead }"
    @click="$emit('click')"
  >
    <div class="icon-wrapper" :class="notification.type">
      <svg v-if="notification.type === 'reply'" viewBox="0 0 24 24" fill="currentColor">
        <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
      </svg>
      <svg v-else-if="notification.type === 'vote'" viewBox="0 0 24 24" fill="currentColor">
        <path d="M14 9V5a3 3 0 0 0-3-3l-4 9v11h11.28a2 2 0 0 0 2-1.7l1.38-9a2 2 0 0 0-2-2.3zM7 22H4a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2h3"/>
      </svg>
      <svg v-else viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10zm0-14v4m0 4h.01"/>
      </svg>
    </div>
    <div class="content">
      <p class="title">{{ notification.title }}</p>
      <p v-if="notification.content" class="desc">{{ notification.content }}</p>
      <span class="time">{{ formatTime(notification.createdAt) }}</span>
    </div>
    <span v-if="!notification.isRead" class="unread-dot" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Notification } from '@/types/notification'
import { formatRelativeTime } from '@/utils/date'

const { t, locale } = useI18n()

defineProps<{
  notification: Notification
}>()

defineEmits<{
  click: []
}>()

const formatTime = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)
</script>

<style scoped>
.notification-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3);
  cursor: pointer;
  transition: background var(--duration-fast) ease;
  position: relative;
}

.notification-item:hover {
  background: var(--bg-secondary);
}

.notification-item.unread {
  background: transparent;
}

.icon-wrapper {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--text-muted);
}

.icon-wrapper svg {
  width: 14px;
  height: 14px;
}

.icon-wrapper.reply,
.icon-wrapper.vote,
.icon-wrapper.system {
  color: var(--text-muted);
}

.content {
  flex: 1;
  min-width: 0;
}

.title {
  font-size: var(--text-sm);
  color: var(--text-primary);
  margin: 0 0 var(--space-1);
  line-height: 1.4;
}

.desc {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin: 0 0 var(--space-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.time {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.unread-dot {
  width: 8px;
  height: 8px;
  background: var(--accent);
  border-radius: 50%;
  flex-shrink: 0;
}
</style>
