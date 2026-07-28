<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      role="presentation"
      @click.self="requestClose"
    >
      <div
        ref="dialogRef"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="dialogTitleId"
        class="bg-bg-card rounded-xl shadow-xl w-full max-w-lg mx-4 p-6 animate-fade-in-up"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <h3 :id="dialogTitleId" class="text-lg font-bold text-text-primary mb-4">
          {{ t('review.admin.editTitle') }}
        </h3>

        <label :for="titleInputId" class="block text-sm text-text-secondary mb-1.5">
          {{ t('review.review.titleLabel') }}
        </label>
        <input
          :id="titleInputId"
          v-model="title"
          type="text"
          maxlength="200"
          autofocus
          :disabled="submitting"
          class="w-full rounded-lg bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40 mb-4"
        />

        <label :for="contentInputId" class="block text-sm text-text-secondary mb-1.5">
          {{ t('review.review.contentLabel') }}
        </label>
        <textarea
          :id="contentInputId"
          v-model="content"
          maxlength="5000"
          rows="6"
          :disabled="submitting"
          class="w-full rounded-lg bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40 resize-none mb-4"
        />

        <label :for="reasonInputId" class="block text-sm text-text-secondary mb-1.5">
          {{ t('review.admin.editReasonLabel') }}
        </label>
        <input
          :id="reasonInputId"
          v-model="reason"
          type="text"
          maxlength="500"
          :placeholder="t('review.admin.editReasonPlaceholder')"
          :disabled="submitting"
          class="w-full rounded-lg bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40"
        />

        <div class="flex justify-end gap-3 mt-5">
          <button
            type="button"
            class="px-4 py-2 text-sm rounded-lg text-text-secondary hover:bg-bg-hover cursor-pointer transition-colors"
            :disabled="submitting"
            @click="requestClose"
          >
            {{ t('common.actions.cancel') }}
          </button>
          <button
            type="button"
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
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Review } from '@stuhelper/shared/review'
import { useBodyScrollLock } from '@/composables/useBodyScrollLock'
import { useDialogFocus } from '@/composables/useDialogFocus'

const props = defineProps<{
  visible: boolean
  review: Review
  submitting?: boolean
}>()

const emit = defineEmits<{
  confirm: [payload: { title: string; content: string; reason: string }]
  close: []
}>()

const { t } = useI18n()
const title = ref('')
const content = ref('')
const reason = ref('')
const visible = computed(() => props.visible)
const submitting = computed(() => props.submitting ?? false)
const dialogRef = ref<HTMLElement | null>(null)
const instanceId = useId()
const dialogTitleId = `admin-edit-dialog-title-${instanceId}`
const titleInputId = `admin-edit-dialog-title-input-${instanceId}`
const contentInputId = `admin-edit-dialog-content-${instanceId}`
const reasonInputId = `admin-edit-dialog-reason-${instanceId}`

useBodyScrollLock(visible)
const { handleKeydown } = useDialogFocus({
  close: requestClose,
  dialogRef,
  open: visible,
})

// 弹窗打开时预填当前内容
watch(() => props.visible, (val) => {
  if (val) {
    title.value = props.review.title || ''
    content.value = props.review.content || ''
    reason.value = ''
  }
})

function requestClose() {
  if (submitting.value) return
  emit('close')
}

function handleConfirm() {
  if (submitting.value) return
  emit('confirm', { title: title.value, content: content.value, reason: reason.value })
}
</script>
