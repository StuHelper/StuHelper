<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { useAttrs } from 'vue'

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<{
  class?: HTMLAttributes['class']
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const attrs = useAttrs()
</script>

<template>
  <input
    v-bind="attrs"
    :class="[
      'flex h-10 w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-950 shadow-sm transition-colors',
      'placeholder:text-slate-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2',
      'disabled:cursor-not-allowed disabled:opacity-50',
      props.class,
    ]"
    :value="modelValue"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  >
</template>
