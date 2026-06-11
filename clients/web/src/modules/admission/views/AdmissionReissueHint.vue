<template>
  <div class="admission-reissue-hint join-chip">
    <p data-admission-reissue-command>
      管理员可在群内使用：
      <code>{{ command }}</code>
    </p>
    <button
      class="secondary-button"
      data-admission-copy-reissue-command
      type="button"
      @click="$emit('copy')"
    >
      复制指令
    </button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  command: string
}>()

defineEmits<{
  copy: []
}>()
</script>

<style scoped>
/*
 * 重发指令提示：玻璃信息卡（.join-chip 提供底/边/圆角），
 * 指令以等宽 code 徽标呈现；复制按钮沿用全局 .secondary-button
 * 玻璃样式（类名同时是测试选择器，必须保留）。
 */
.admission-reissue-hint {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  padding: 12px 14px;
}

.admission-reissue-hint p {
  color: var(--join-ink-soft);
  font-size: 13px;
  line-height: 24px;
  margin: 0;
}

.admission-reissue-hint code {
  background: var(--color-primary-alpha);
  border: 1px solid var(--join-glass-border-soft);
  border-radius: 8px;
  color: var(--join-ink);
  font-family: var(--font-mono);
  font-size: 12.5px;
  overflow-wrap: anywhere;
  padding: 3px 8px;
}
</style>
