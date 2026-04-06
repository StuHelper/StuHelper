<script setup lang="ts">
interface Props {
  variant?: 'card' | 'navbar' | 'modal'
}

withDefaults(defineProps<Props>(), {
  variant: 'card'
})
</script>

<template>
  <div :class="['glass-card', `glass-${variant}`]">
    <slot />
  </div>
</template>

<style scoped>
.glass-navbar {
  background: var(--color-bg-glass);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
}

.glass-modal {
  background: var(--color-bg-glass-heavy);
  backdrop-filter: blur(30px) saturate(200%);
  -webkit-backdrop-filter: blur(30px) saturate(200%);
}
</style>
