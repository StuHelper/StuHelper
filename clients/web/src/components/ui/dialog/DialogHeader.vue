<script setup lang="ts">
import type { HTMLAttributes } from 'vue'

const props = defineProps<{
  class?: HTMLAttributes['class']
}>()
</script>

<template>
  <div :class="['flex flex-col gap-2 text-left', props.class]">
    <slot />
  </div>
</template>
