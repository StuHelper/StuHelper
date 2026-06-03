<script setup lang="ts">
import type { HTMLAttributes } from 'vue'

type ButtonVariant = 'default' | 'outline' | 'secondary' | 'destructive' | 'ghost'
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon'

const props = withDefaults(defineProps<{
  class?: HTMLAttributes['class']
  disabled?: boolean
  size?: ButtonSize
  type?: 'button' | 'submit' | 'reset'
  variant?: ButtonVariant
}>(), {
  size: 'default',
  type: 'button',
  variant: 'default',
})

const baseClasses = [
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-semibold',
  'transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-950 focus-visible:ring-offset-2',
  'disabled:pointer-events-none disabled:opacity-50',
].join(' ')

const variantClasses: Record<ButtonVariant, string> = {
  default: 'bg-slate-950 text-white hover:bg-slate-800',
  destructive: 'bg-red-600 text-white hover:bg-red-700',
  ghost: 'text-slate-700 hover:bg-slate-100 hover:text-slate-950',
  outline: 'border border-slate-200 bg-white text-slate-900 hover:bg-slate-50',
  secondary: 'bg-slate-100 text-slate-900 hover:bg-slate-200',
}

const sizeClasses: Record<ButtonSize, string> = {
  default: 'h-10 px-4 py-2',
  icon: 'h-10 w-10',
  lg: 'h-11 px-5',
  sm: 'h-9 px-3',
}
</script>

<template>
  <button
    :class="[
      baseClasses,
      variantClasses[variant],
      sizeClasses[size],
      props.class,
    ]"
    :disabled="disabled"
    :type="type"
  >
    <slot />
  </button>
</template>
