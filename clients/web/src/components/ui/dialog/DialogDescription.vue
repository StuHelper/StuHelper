<script setup lang="ts">
import type { HTMLAttributes } from 'vue'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <p
    :class="['text-sm leading-relaxed text-slate-600', props.class]"
  >
    <slot />
  </p>
</template>
