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
        :aria-labelledby="titleId"
        class="bg-bg-card rounded-xl shadow-xl w-full max-w-md mx-4 p-6 animate-fade-in-up"
        tabindex="-1"
        @keydown="handleKeydown"
      >
        <h3 :id="titleId" class="text-lg font-bold text-text-primary mb-4">
          {{ t('review.admin.moderateTitle') }}
        </h3>

        <label :for="reasonId" class="block text-sm text-text-secondary mb-1.5">
          {{ t('review.admin.moderateReasonLabel') }}
        </label>
        <textarea
          :id="reasonId"
          v-model="reason"
          :aria-label="t('review.admin.moderateReasonLabel')"
          :placeholder="t('review.admin.moderateReasonPlaceholder')"
          maxlength="500"
          rows="3"
          data-dialog-initial-focus
          :disabled="submitting"
          class="w-full rounded-lg bg-bg-input px-3 py-2 text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/40 resize-none"
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
            class="px-4 py-2 text-sm rounded-lg bg-warning text-white font-medium cursor-pointer hover:opacity-90 transition-opacity disabled:opacity-50"
            :disabled="submitting"
            @click="handleConfirm"
          >
            {{ t('review.admin.moderateConfirm') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useBodyScrollLock } from '@/composables/useBodyScrollLock'
import { useDialogFocus } from '@/composables/useDialogFocus'

const props = defineProps<{
  visible: boolean
  submitting?: boolean
}>()

const emit = defineEmits<{
  confirm: [reason: string]
  close: []
}>()

const { t } = useI18n()
const reason = ref('')
const visible = computed(() => props.visible)
const submitting = computed(() => props.submitting ?? false)
const dialogRef = ref<HTMLElement | null>(null)
const instanceId = useId()
const titleId = `moderation-dialog-title-${instanceId}`
const reasonId = `moderation-dialog-reason-${instanceId}`

useBodyScrollLock(visible)
const { handleKeydown } = useDialogFocus({
  close: requestClose,
  dialogRef,
  open: visible,
})

watch(
  () => props.visible,
  (isVisible, wasVisible) => {
    if (isVisible && !wasVisible) reason.value = ''
  },
)

function requestClose() {
  if (submitting.value) return
  emit('close')
}

function handleConfirm() {
  if (submitting.value) return
  emit('confirm', reason.value)
}
</script>
