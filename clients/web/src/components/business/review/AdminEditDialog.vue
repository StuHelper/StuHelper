<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      @click.self="emit('close')"
    >
      <div class="bg-bg-card rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 animate-fade-in-up">
        <h3 class="text-lg font-bold text-text-primary mb-4">{{ t('review.admin.editTitle') }}</h3>

        <label class="block text-sm text-text-secondary mb-1.5">{{ t('review.review.titleLabel') }}</label>
        <input
          v-model="title"
          type="text"
          maxlength="200"
          class="w-full rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40 mb-4"
        />

        <label class="block text-sm text-text-secondary mb-1.5">{{ t('review.review.contentLabel') }}</label>
        <textarea
          v-model="content"
          maxlength="5000"
          rows="6"
          class="w-full rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40 resize-none mb-4"
        />

        <label class="block text-sm text-text-secondary mb-1.5">{{ t('review.admin.editReasonLabel') }}</label>
        <input
          v-model="reason"
          type="text"
          maxlength="500"
          :placeholder="t('review.admin.editReasonPlaceholder')"
          class="w-full rounded-lg border border-border bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40"
        />

        <div class="flex justify-end gap-3 mt-5">
          <button
            class="px-4 py-2 text-sm rounded-lg text-text-secondary hover:bg-bg-hover cursor-pointer transition-colors"
            @click="emit('close')"
          >
            {{ t('common.actions.cancel') }}
          </button>
          <button
            class="px-4 py-2 text-sm rounded-lg bg-primary text-white font-medium cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
            :disabled="!content.trim() || submitting"
            @click="handleConfirm"
          >
            {{ t('review.admin.editConfirm') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Review } from '@/types/review'

const props = defineProps<{
  visible: boolean
  review: Review
}>()

const emit = defineEmits<{
  confirm: [payload: { title: string; content: string; reason: string }]
  close: []
}>()

const { t } = useI18n()
const title = ref('')
const content = ref('')
const reason = ref('')
const submitting = ref(false)

// 弹窗打开时预填当前内容
watch(() => props.visible, (val) => {
  if (val) {
    title.value = props.review.title || ''
    content.value = props.review.content || ''
    reason.value = ''
  }
})

function handleConfirm() {
  emit('confirm', { title: title.value, content: content.value, reason: reason.value })
}
</script>
