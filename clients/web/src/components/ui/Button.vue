<script setup lang="ts">
interface Props {
  variant?: 'primary' | 'secondary' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  type?: 'button' | 'submit' | 'reset'
}

withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  type: 'button'
})
</script>

<template>
  <button
    :type="type"
    :class="[
      'btn',
      `btn-${variant}`,
      `btn-${size}`,
      'transition-all duration-300'
    ]"
  >
    <slot />
  </button>
</template>

<style scoped>
.btn {
  border-radius: 0.5rem;
  font-weight: 500;
  cursor: pointer;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-secondary {
  background: var(--color-secondary);
  color: white;
}

.btn-ghost {
  background: transparent;
  border: 1px solid currentColor;
}

.btn-sm { padding: 0.5rem 1rem; font-size: 0.875rem; }
.btn-md { padding: 0.75rem 1.5rem; font-size: 1rem; }
.btn-lg { padding: 1rem 2rem; font-size: 1.125rem; }
</style>
