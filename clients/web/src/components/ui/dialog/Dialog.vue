<script setup lang="ts">
import { computed, provide } from 'vue'

import { dialogContextKey } from './context'

const props = withDefaults(defineProps<{
  open?: boolean
}>(), {
  open: false,
})

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

provide(dialogContextKey, {
  open: computed(() => props.open),
  setOpen(open: boolean) {
    emit('update:open', open)
  },
})
</script>

<template>
  <slot />
</template>
