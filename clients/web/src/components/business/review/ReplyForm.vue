<template>
  <div class="py-3">
    <textarea
      v-model="content"
      class="w-full min-h-[80px] p-3 bg-bg-secondary rounded-lg text-text-primary text-sm font-[inherit] resize-y transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card placeholder:text-text-muted"
      :placeholder="placeholder ?? t('review.reply.placeholder')"
      :aria-label="t('review.reply.inputLabel')"
      :aria-describedby="content.length > 0 && content.trim().length < minLength ? errorId : undefined"
      :maxlength="maxLength"
      @keydown.ctrl.enter="handleSubmit"
      @keydown.meta.enter="handleSubmit"
    />
    <span v-if="content.length > 0 && content.trim().length < minLength" :id="errorId" class="block text-xs text-danger mt-1">
      {{ t('review.validation.replyTooShort', { min: minLength }) }}
    </span>
    <div class="flex items-center justify-between mt-2">
      <span
        class="text-xs text-text-muted"
        :class="{
          'text-color-rating-3': content.length > maxLength * 0.9,
          '!text-danger': content.length > 0 && content.trim().length < minLength
        }"
      >
        {{ t('review.validation.charCount', { current: content.length, max: maxLength }) }}
      </span>
      <div class="flex gap-2">
        <button
          class="px-3 py-1 text-sm rounded-full cursor-pointer transition-all duration-fast bg-transparent text-text-secondary hover:border-text-muted"
          @click="handleCancel"
        >{{ t('common.actions.cancel') }}</button>
        <button
          class="px-3 py-1 text-sm rounded-full cursor-pointer transition-all duration-fast bg-gradient-to-br from-primary to-accent border-none text-white font-medium hover:not-disabled:opacity-90 hover:not-disabled:-translate-y-px disabled:opacity-50 disabled:cursor-not-allowed"
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
import { ref, computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const errorId = `reply-form-error-${useId()}`

const props = withDefaults(defineProps<{
  placeholder?: string
  maxLength?: number
  minLength?: number
  submitting?: boolean
}>(), {
  placeholder: undefined,
  maxLength: 500,
  minLength: 2,
  submitting: false
})

const emit = defineEmits<{
  submit: [content: string]
  cancel: []
}>()

const content = ref('')

const canSubmit = computed(() => {
  const trimmed = content.value.trim()
  return trimmed.length >= props.minLength && trimmed.length <= props.maxLength
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
