<template>
  <div class="notification-bell" :class="{ 'has-new': hasUnread }">
    <button class="bell-btn" @click="togglePanel">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" />
        <path d="M13.73 21a2 2 0 0 1-3.46 0" />
      </svg>
      <span v-if="unreadCount > 0" class="badge">
        {{ unreadCount > 99 ? '99+' : unreadCount }}
      </span>
    </button>

    <transition name="dropdown">
      <div v-if="showPanel" class="panel">
        <div class="panel-header">
          <span>{{ t('user.notification.bell') }}</span>
          <button
            v-if="hasUnread"
            class="mark-all-btn"
            @click="handleMarkAllRead"
          >
            {{ t('user.notification.bellMarkAllRead') }}
          </button>
        </div>
        <NotificationList
          :notifications="notifications"
          :loading="loading"
          @click="handleNotificationClick"
        />
        <router-link to="/notifications" class="view-all" @click="showPanel = false">
          {{ t('user.notification.viewAll') }}
        </router-link>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotificationStore } from '@/stores/notification'
import NotificationList from './NotificationList.vue'

const { t } = useI18n()
const store = useNotificationStore()
const showPanel = ref(false)

const notifications = computed(() => store.notifications.slice(0, 5))
const unreadCount = computed(() => store.unreadCount)
const hasUnread = computed(() => store.hasUnread)
const loading = computed(() => store.loading)

const togglePanel = () => {
  showPanel.value = !showPanel.value
  if (showPanel.value && notifications.value.length === 0) {
    store.fetchNotifications(1, 5)
  }
}

const handleMarkAllRead = () => {
  store.markAllAsRead()
}

const handleNotificationClick = (id: number) => {
  store.markAsRead(id)
}

// 点击外部关闭
const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as HTMLElement
  if (!target.closest('.notification-bell')) {
    showPanel.value = false
  }
}

onMounted(() => {
  store.startPolling()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  store.stopPolling()
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.notification-bell {
  position: relative;
}

.bell-btn {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: color var(--duration-fast);
}

.bell-btn:hover {
  color: var(--text-primary);
}

.bell-btn svg {
  width: 20px;
  height: 20px;
}

.has-new .bell-btn svg {
  animation: bellShake 0.5s ease;
}

.badge {
  position: absolute;
  top: 2px;
  right: 2px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  background: var(--accent);
  color: white;
  font-size: 10px;
  font-weight: var(--weight-semibold);
  font-variant-numeric: tabular-nums;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.panel {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  width: 320px;
  background: var(--bg-base);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  overflow: hidden;
  z-index: 100;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3);
  border-bottom: 1px solid var(--border);
  font-weight: var(--weight-medium);
  font-size: var(--text-sm);
}

.mark-all-btn {
  font-size: var(--text-xs);
  color: var(--text-muted);
  background: none;
  border: none;
  cursor: pointer;
  transition: color var(--duration-fast);
}

.mark-all-btn:hover {
  color: var(--text-primary);
}

.view-all {
  display: block;
  padding: var(--space-3);
  text-align: center;
  font-size: var(--text-sm);
  color: var(--text-muted);
  border-top: 1px solid var(--border);
  text-decoration: none;
  transition: color var(--duration-fast);
}

.view-all:hover {
  color: var(--text-primary);
}

@keyframes bellShake {
  0%, 100% { transform: rotate(0); }
  25% { transform: rotate(15deg); }
  75% { transform: rotate(-15deg); }
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-base) ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
