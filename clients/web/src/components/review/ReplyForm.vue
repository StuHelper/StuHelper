<template>
  <div class="reply-form">
    <textarea
      v-model="content"
      class="reply-input"
      :placeholder="placeholder ?? t('review.reply.placeholder')"
      :maxlength="maxLength"
      @keydown.ctrl.enter="handleSubmit"
      @keydown.meta.enter="handleSubmit"
    />
    <div class="reply-form-footer">
      <span class="char-count" :class="{ warning: content.length > maxLength * 0.9 }">
        {{ content.length }}/{{ maxLength }}
      </span>
      <div class="actions">
        <button class="cancel-btn" @click="handleCancel">{{ t('common.actions.cancel') }}</button>
        <button
          class="submit-btn"
          :disabled="!canSubmit || submitting"
          @click="handleSubmit"
        >
          {{ submitting ? t('common.actions.sending') : t('common.actions.send') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  placeholder?: string
  maxLength?: number
  submitting?: boolean
}>(), {
  placeholder: undefined,
  maxLength: 500,
  submitting: false
})

const emit = defineEmits<{
  submit: [content: string]
  cancel: []
}>()

const content = ref('')

const canSubmit = computed(() => {
  const trimmed = content.value.trim()
  return trimmed.length > 0 && trimmed.length <= props.maxLength
})

const handleSubmit = () => {
  if (!canSubmit.value || props.submitting) return
  emit('submit', content.value.trim())
}

const handleCancel = () => {
  content.value = ''
  emit('cancel')
}

// 暴露清空方法
defineExpose({
  clear: () => { content.value = '' }
})
</script>

<style scoped>
.reply-form {
  padding: var(--space-3) 0;
}

.reply-input {
  width: 100%;
  min-height: 80px;
  padding: var(--space-2);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: var(--text-sm);
  font-family: inherit;
  resize: vertical;
  transition: border-color var(--duration-fast);
}

.reply-input:focus {
  outline: none;
  border-color: var(--brand-primary);
}

.reply-input::placeholder {
  color: var(--text-muted);
}

.reply-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: var(--space-2);
}

.char-count {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.char-count.warning {
  color: #f59e0b;
}

.actions {
  display: flex;
  gap: var(--space-2);
}

.cancel-btn,
.submit-btn {
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-sm);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.cancel-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.cancel-btn:hover {
  border-color: var(--text-muted);
}

.submit-btn {
  background: var(--text-primary);
  border: none;
  color: var(--bg-base);
  font-weight: var(--weight-medium);
}

.submit-btn:hover:not(:disabled) {
  background: var(--brand-primary);
  color: white;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
