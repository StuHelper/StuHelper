<template>
  <span class="sh-tag" :class="intentClass">{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type TagIntent =
  | 'neutral'
  | 'primary'
  | 'success'
  | 'warning'
  | 'danger'
  | 'info'
  | 'muted'

const props = withDefaults(
  defineProps<{
    label: string
    intent?: TagIntent
  }>(),
  { intent: 'neutral' },
)

const intentClass = computed(() => `sh-tag--${props.intent}`)
</script>
