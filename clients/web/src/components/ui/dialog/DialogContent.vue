<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { computed, inject, useAttrs } from 'vue'
import { X } from 'lucide-vue-next'

import { dialogContextKey } from './context'

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
const dialog = inject(dialogContextKey)
const attrs = useAttrs()
const open = computed(() => dialog?.open.value ?? false)

function closeDialog(): void {
  dialog?.setOpen(false)
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50" @keydown.esc="closeDialog">
      <div class="fixed inset-0 bg-black/45" @click="closeDialog" />
      <div
        v-bind="attrs"
        :class="[
          'fixed left-1/2 top-1/2 z-50 grid w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 gap-4',
          'rounded-lg border border-slate-200 bg-white p-6 text-slate-950 shadow-xl',
          props.class,
        ]"
        role="dialog"
        aria-modal="true"
        tabindex="-1"
        @click.stop
      >
        <slot />
        <button
          class="absolute right-4 top-4 grid h-8 w-8 place-items-center rounded-md text-slate-500 opacity-80 transition-colors hover:bg-slate-100 hover:text-slate-950 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950"
          type="button"
          @click="closeDialog"
        >
          <X class="h-4 w-4" aria-hidden="true" />
          <span class="sr-only">关闭</span>
        </button>
      </div>
    </div>
  </Teleport>
</template>
