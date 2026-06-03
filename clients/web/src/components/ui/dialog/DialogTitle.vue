<script setup lang="ts">
import type { HTMLAttributes } from 'vue'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <h2
    :class="['text-lg font-semibold leading-none tracking-normal text-slate-950', props.class]"
  >
    <slot />
  </h2>
</template>
