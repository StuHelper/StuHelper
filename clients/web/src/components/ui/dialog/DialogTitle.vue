<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { inject } from 'vue'

import { dialogContextKey } from './context'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()

const dialog = inject(dialogContextKey)
</script>

<template>
  <h2
    :id="dialog?.titleId"
    :class="['text-lg font-semibold leading-none tracking-normal text-slate-950', props.class]"
  >
    <slot />
  </h2>
</template>
