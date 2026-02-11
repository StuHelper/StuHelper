<template>
  <div class="notification-list">
    <div v-if="loading" class="loading">
      <span class="spinner" />
    </div>
    <template v-else-if="notifications.length > 0">
      <NotificationItem
        v-for="n in notifications"
        :key="n.id"
        :notification="n"
        @click="$emit('click', n.id)"
      />
    </template>
    <div v-else class="empty">
      暂无通知
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Notification } from '@/types/notification'
import NotificationItem from './NotificationItem.vue'

defineProps<{
  notifications: Notification[]
  loading?: boolean
}>()

defineEmits<{
  click: [id: number]
}>()
</script>

<style scoped>
.notification-list {
  max-height: 300px;
  overflow-y: auto;
}

.loading,
.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border);
  border-top-color: var(--text-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
