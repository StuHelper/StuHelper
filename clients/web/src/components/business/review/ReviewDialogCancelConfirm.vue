<template>
  <Transition name="overlay">
    <div v-if="show" class="fixed inset-0 bg-black/30 z-[var(--z-modal)] flex items-center justify-center p-4" @click.self="$emit('cancel')">
      <div class="bg-bg-card rounded-xl shadow-2xl w-full max-w-[340px] p-6 flex flex-col items-center gap-4 animate-modal-in">
        <div class="w-11 h-11 rounded-full bg-warning/10 flex items-center justify-center">
          <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-warning"><path d="M14 3v4a1 1 0 0 0 1 1h4"/><path d="M17 21H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h7l5 5v11a2 2 0 0 1-2 2z"/><path d="M12 10v4"/><path d="M12 17h.01"/></svg>
        </div>
        <p class="text-sm font-medium text-text-primary text-center">{{ t('review.draft.confirmSave') }}</p>
        <div class="flex gap-3 w-full">
          <button
            class="flex-1 py-2 px-3 text-sm text-text-secondary rounded-lg cursor-pointer transition-colors duration-fast hover:bg-bg-secondary"
            @click="$emit('discard')"
          >
            {{ t('review.draft.discard') }}
          </button>
          <button
            class="flex-1 py-2 px-3 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-lg cursor-pointer transition-opacity duration-fast disabled:opacity-50"
            :disabled="saving"
            @click="$emit('saveDraft')"
          >
            {{ saving ? t('review.draft.saving') : t('review.draft.saveDraft') }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  show: boolean
  saving: boolean
}>()

defineEmits<{
  cancel: []
  discard: []
  saveDraft: []
}>()

const { t } = useI18n()
</script>
