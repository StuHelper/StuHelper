<template>
  <div class="max-h-[300px] overflow-y-auto">
    <div v-if="loading" class="flex items-center justify-center p-6 text-text-muted text-sm">
      <span class="w-5 h-5 border-2 border-border border-t-text-primary rounded-full animate-spin" />
    </div>
    <template v-else-if="notifications.length > 0">
      <NotificationItem
        v-for="n in notifications"
        :key="n.id"
        :notification="n"
        @click="$emit('click', n.id)"
      />
    </template>
    <div v-else class="flex items-center justify-center p-6 text-text-muted text-sm">
      {{ t('user.notification.empty') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Notification } from '@stuhelper/shared/notification'
import NotificationItem from './NotificationItem.vue'

const { t } = useI18n()

defineProps<{
  notifications: Notification[]
  loading?: boolean
}>()

defineEmits<{
  click: [id: string]
}>()
</script>
