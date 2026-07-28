<script setup lang="ts">
import { computed, provide, useId } from 'vue'

import { dialogContextKey } from './context'

const props = withDefaults(defineProps<{
  open?: boolean
}>(), {
  open: false,
})

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

const id = useId()

provide(dialogContextKey, {
  descriptionId: `dialog-description-${id}`,
  open: computed(() => props.open),
  setOpen(open: boolean) {
    emit('update:open', open)
  },
  titleId: `dialog-title-${id}`,
})
</script>

<template>
  <slot />
</template>
