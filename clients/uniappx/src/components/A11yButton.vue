<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  ariaDisabled?: boolean | string
  disabled?: boolean
}>(), {
  ariaDisabled: false,
  disabled: false,
})

const emit = defineEmits<{
  tap: []
}>()

const isAriaDisabled = computed(() => (
  props.ariaDisabled === true || props.ariaDisabled === 'true'
))
const isInactive = computed(() => props.disabled || isAriaDisabled.value)

function activate() {
  if (isInactive.value) return
  emit('tap')
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' && event.key !== ' ' && event.key !== 'Spacebar') return
  event.preventDefault()
  activate()
}
</script>

<template>
  <button
    class="a11y-button"
    role="button"
    :tabindex="disabled ? -1 : 0"
    :disabled="disabled"
    :aria-disabled="isInactive"
    @tap="activate"
    @keydown="handleKeydown"
  >
    <slot />
  </button>
</template>
