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
  </div>
</template>

<script setup lang="ts">
defineProps<{
  timedOut: boolean
}>()
</script>
