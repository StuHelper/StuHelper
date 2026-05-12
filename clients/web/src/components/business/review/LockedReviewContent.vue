<template>
  <div class="rounded-lg border border-border-light bg-bg-elevated/60 px-4 py-4">
    <div
      v-if="content"
      class="relative overflow-hidden rounded-md"
    >
      <p
        aria-hidden="true"
        class="m-0 whitespace-pre-line text-sm leading-relaxed text-text-secondary break-words blur-[3px] select-none"
        v-text="content"
      />
      <p
        aria-hidden="true"
        class="absolute inset-x-0 top-0 m-0 line-clamp-1 bg-bg-elevated text-sm leading-relaxed text-text-secondary break-words"
        v-text="visibleLine"
      />
      <div class="pointer-events-none absolute inset-0 bg-gradient-to-b from-transparent via-bg-elevated/25 to-bg-elevated/90" />
      <div class="pointer-events-none absolute inset-x-0 bottom-0 flex justify-center pb-2">
        <div class="inline-flex items-center gap-1.5 rounded-full border border-border-light bg-bg-card/90 px-3 py-1.5 shadow-xs backdrop-blur">
          <Lock :size="14" class="text-text-muted" />
          <span class="text-xs font-medium text-text-secondary">{{ message }}</span>
        </div>
      </div>
    </div>
    <div class="mt-3 flex flex-col items-center gap-1.5 text-center">
      <button
        class="text-xs font-medium text-primary hover:underline"
        type="button"
        @click="$emit('action')"
      >
        {{ actionLabel }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Lock } from 'lucide-vue-next'

const props = defineProps<{
  content?: string
  message: string
  actionLabel: string
}>()

defineEmits<{
  action: []
}>()

const content = computed(() => (props.content ?? '').trim())
const visibleLine = computed(() => {
  const [firstLine = ''] = content.value.split(/\r?\n/, 1)
  return firstLine.trim()
})
</script>
