<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4"
      @click.self="$emit('keep')"
    >
      <section
        class="w-full max-w-[420px] rounded-xl bg-bg-card shadow-xl p-5"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
      >
        <h2 :id="titleId" class="m-0 text-lg font-bold text-text-primary">
          {{ title }}
        </h2>
        <p class="mt-3 mb-0 text-sm leading-relaxed text-text-secondary">
          {{ description }}
        </p>
        <div class="mt-5 flex flex-wrap justify-end gap-2">
          <button
            v-if="cancelText"
            type="button"
            class="px-4 py-2 rounded-lg border border-border bg-transparent text-text-secondary cursor-pointer hover:bg-bg-hover"
            @click="$emit('keep')"
          >
            {{ cancelText }}
          </button>
          <button
            v-if="dangerText"
            type="button"
            class="px-4 py-2 rounded-lg border border-danger/30 bg-danger/10 text-danger cursor-pointer hover:bg-danger/15"
            @click="$emit('discard')"
          >
            {{ dangerText }}
          </button>
          <button
            type="button"
            class="px-4 py-2 rounded-lg border border-primary bg-primary text-white cursor-pointer hover:bg-primary-dark"
            @click="$emit('confirm')"
          >
            {{ confirmText }}
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{
  cancelText?: string
  confirmText: string
  dangerText?: string
  description: string
  title: string
  titleId: string
  visible: boolean
}>()

defineEmits<{
  confirm: []
  discard: []
  keep: []
}>()
</script>
