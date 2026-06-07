<template>
  <div
    class="my-3 rounded-lg border border-primary/15 bg-primary/[0.06] px-4 py-3"
    data-testid="review-reply-login-prompt"
  >
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <p class="m-0 text-sm text-text-secondary">
        {{ t('review.reply.loginPrompt') }}
      </p>
      <button
        class="shrink-0 rounded-full bg-primary px-4 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90"
        type="button"
        @click="$emit('login')"
      >
        {{ t('review.reply.loginAction') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineEmits<{
  login: []
}>()

const { t } = useI18n()
</script>
