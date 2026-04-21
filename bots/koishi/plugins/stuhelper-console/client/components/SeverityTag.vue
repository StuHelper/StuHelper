<template>
  <el-tag
    :class="['sh-tag', `sh-tag--${intent}`]"
    :type="tagType"
    effect="light"
    size="small"
    round
  >
    {{ label }}
  </el-tag>
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

const tagType = computed(() => {
  switch (props.intent) {
    case 'success':
      return 'success'
    case 'warning':
      return 'warning'
    case 'danger':
      return 'danger'
    case 'info':
      return 'info'
    default:
      return undefined
  }
})
</script>
