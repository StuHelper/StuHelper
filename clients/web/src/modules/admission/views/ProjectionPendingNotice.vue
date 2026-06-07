<template>
  <div data-state="projectionPending">
    <h2 class="text-lg font-semibold">身份生效中</h2>
    <p
      v-if="timedOut"
      class="mt-2 text-sm text-slate-600"
      data-projection-timeout
    >
      身份仍在同步，请稍后手动刷新页面。
    </p>
    <p v-else class="mt-2 text-sm text-slate-600">
      认证已通过，正在刷新身份能力。
    </p>
    <button
      v-if="timedOut"
      type="button"
      class="mt-4 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
      data-projection-retry
      @click="$emit('retry')"
    >
      重新检查状态
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  timedOut: boolean
}>()

defineEmits<{
  retry: []
}>()
</script>
