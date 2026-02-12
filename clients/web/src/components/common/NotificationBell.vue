<template>
  <div class="notification-bell relative" :class="{ 'has-new': hasUnread }">
    <button
      class="bell-btn relative flex items-center justify-center w-9 h-9 bg-transparent border-none text-text-muted cursor-pointer rounded-sm transition-colors duration-fast hover:text-text-primary"
      :aria-label="t('user.notification.bell')"
      :aria-expanded="showPanel"
      @click="togglePanel"
    >
      <Bell :size="20" />
      <span
        v-if="unreadCount > 0"
        class="absolute top-0.5 right-0.5 min-w-4 h-4 px-1 bg-accent text-white text-[10px] font-semibold tabular-nums rounded-lg flex items-center justify-center"
      >
        {{ unreadCount > 99 ? '99+' : unreadCount }}
      </span>
    </button>

    <transition name="dropdown">
      <div
        v-if="showPanel"
        class="absolute top-[calc(100%+8px)] right-0 w-80 bg-bg-base border border-border rounded-sm overflow-hidden z-[100]"
      >
        <div class="flex items-center justify-between p-3 border-b border-border font-medium text-sm">
          <span>{{ t('user.notification.bell') }}</span>
          <button
            v-if="hasUnread"
            class="text-xs text-text-muted bg-transparent border-none cursor-pointer hover:text-text-primary transition-colors duration-fast"
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
        <router-link
          to="/notifications"
          class="block p-3 text-center text-sm text-text-muted border-t border-border no-underline hover:text-text-primary transition-colors duration-fast"
          @click="showPanel = false"
        >
          {{ t('user.notification.viewAll') }}
        </router-link>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bell } from 'lucide-vue-next'
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

const handleNotificationClick = (id: string) => {
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
.has-new .bell-btn svg {
  animation: bellShake 0.5s ease;
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
