<template>
  <el-empty
    :class="['sh-empty', { 'sh-empty--error': tone === 'error' }]"
    :image-size="72"
  >
    <template #description>
      <div class="sh-empty__copy">
        <p class="sh-empty__title">{{ title }}</p>
        <p v-if="body" class="sh-empty__body">{{ body }}</p>
      </div>
    </template>
    <slot name="action" />
  </el-empty>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    body?: string
    tone?: 'default' | 'error'
  }>(),
  {
    body: '',
    tone: 'default',
  },
)
</script>
