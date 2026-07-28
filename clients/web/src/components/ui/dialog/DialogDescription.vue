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
  <p
    :id="dialog?.descriptionId"
    :class="['text-sm leading-relaxed text-slate-600', props.class]"
  >
    <slot />
  </p>
</template>
