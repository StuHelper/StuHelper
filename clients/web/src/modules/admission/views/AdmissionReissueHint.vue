<template>
  <div class="admission-reissue-hint">
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
.admission-reissue-hint {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.admission-reissue-hint p {
  color: #64748b;
  font-size: 13px;
  line-height: 20px;
  margin: 0;
}

.admission-reissue-hint code {
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  color: #0f172a;
  font-size: 13px;
  padding: 2px 6px;
}

.secondary-button {
  align-items: center;
  background: #e2e8f0;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #0f172a;
  cursor: pointer;
  display: inline-flex;
  font-size: 14px;
  font-weight: 600;
  justify-content: center;
  line-height: 20px;
  min-height: 44px;
  padding: 10px 16px;
  transition: background-color 160ms ease, border-color 160ms ease;
}

.secondary-button:hover {
  background: #cbd5e1;
}

.secondary-button:focus-visible {
  outline: 3px solid rgb(15 118 110 / 0.28);
  outline-offset: 2px;
}
</style>
