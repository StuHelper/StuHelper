<template>
  <div
    class="p-3 mb-2 last:mb-0 bg-bg-secondary rounded-lg animate-fade-in-up"
    :class="reply.isOwner && 'bg-primary/[0.06] border-l-3 border-l-primary'"
    :data-testid="`reply-card-${reply.id}`"
  >
    <div class="text-sm text-text-primary leading-relaxed break-words">
      {{ reply.content }}
    </div>
    <div class="flex items-center justify-between mt-2">
      <span class="text-xs text-text-muted">{{ formattedTime }}</span>
      <button
        v-if="reply.isOwner"
        class="text-xs text-text-muted bg-transparent border-none cursor-pointer px-2 py-1 rounded-sm transition-all duration-fast hover:text-accent"
        type="button"
        :data-testid="`reply-delete-${reply.id}`"
        @click="requestDelete"
      >
        {{ t('common.actions.delete') }}
      </button>
    </div>
    <div
      v-if="confirmingDelete"
      class="mt-3 rounded-md border border-accent/20 bg-accent/5 p-3"
      :data-testid="`reply-delete-confirm-${reply.id}`"
      role="group"
      :aria-label="t('review.reply.deleteConfirm')"
    >
      <p class="m-0 text-xs text-text-primary">{{ t('review.reply.deleteConfirm') }}</p>
      <div class="mt-3 flex justify-end gap-2">
        <button
          class="rounded-sm border border-border bg-bg-primary px-3 py-1 text-xs text-text-secondary transition-colors hover:text-text-primary"
          type="button"
          @click="cancelDelete"
        >
          {{ t('common.actions.cancel') }}
        </button>
        <button
          class="rounded-sm border border-accent bg-accent px-3 py-1 text-xs text-white transition-colors hover:bg-accent/90"
          type="button"
          @click="confirmDelete"
        >
          {{ t('common.actions.confirm') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Reply } from '@stuhelper/shared/reply'
import { formatRelativeTime } from '@/utils/date'

const { t, locale } = useI18n()

const props = defineProps<{
  reply: Reply
}>()

const emit = defineEmits<{
  delete: [id: string]
}>()

const confirmingDelete = ref(false)

const formattedTime = computed(() => formatRelativeTime(props.reply.createdAt, locale.value, t))

const requestDelete = () => {
  confirmingDelete.value = true
}

const cancelDelete = () => {
  confirmingDelete.value = false
}

const confirmDelete = () => {
  confirmingDelete.value = false
  emit('delete', props.reply.id)
}
</script>
