<template>
  <header class="sh-workspace-header">
    <div class="sh-workspace-header__copy">
      <h1 class="sh-workspace-header__title">{{ title }}</h1>
      <p v-if="description" class="sh-workspace-header__description">
        {{ description }}
      </p>
      <p v-if="meta" class="sh-workspace-header__meta">{{ meta }}</p>
    </div>
    <div v-if="$slots.actions" class="sh-workspace-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    description?: string
    meta?: string
  }>(),
  {
    description: '',
    meta: '',
  },
)
</script>
