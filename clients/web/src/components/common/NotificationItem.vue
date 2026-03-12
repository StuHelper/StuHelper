<template>
  <div
    class="flex items-start gap-3 p-3 cursor-pointer transition-colors duration-fast hover:bg-bg-secondary relative"
    @click="$emit('click')"
  >
    <div class="w-7 h-7 rounded-sm flex items-center justify-center shrink-0 text-text-muted">
      <MessageSquare v-if="notification.type === 'reply'" class="w-3.5 h-3.5" />
      <ThumbsUp v-else-if="notification.type === 'vote'" class="w-3.5 h-3.5" />
      <Info v-else class="w-3.5 h-3.5" />
    </div>
    <div class="flex-1 min-w-0">
      <p class="text-sm text-text-primary m-0 mb-1 leading-snug">{{ notification.title }}</p>
      <p v-if="notification.content" class="text-xs text-text-muted m-0 mb-1 overflow-hidden text-ellipsis whitespace-nowrap">{{ notification.content }}</p>
      <span class="text-xs text-text-muted">{{ formattedTime }}</span>
    </div>
    <span v-if="!notification.isRead" class="w-2 h-2 bg-accent rounded-full shrink-0" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessageSquare, ThumbsUp, Info } from 'lucide-vue-next'
import type { Notification } from '@/types/notification'
import { formatRelativeTime } from '@/utils/date'

const { t, locale } = useI18n()

const props = defineProps<{
  notification: Notification
}>()

defineEmits<{
  click: []
}>()

const formattedTime = computed(() => formatRelativeTime(props.notification.createdAt, locale.value, t))
</script>
