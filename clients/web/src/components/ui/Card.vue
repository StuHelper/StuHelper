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
.glass-card {
  border-radius: 1rem;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.15);
}

.glass-navbar {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px) saturate(180%);
}

.glass-modal {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(30px) saturate(200%);
}
</style>
