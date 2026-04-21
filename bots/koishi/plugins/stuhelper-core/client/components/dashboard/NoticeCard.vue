<template>
  <div class="card notice-card" v-if="content">
    <div class="card-header">
      <k-icon name="stuhelperGroupCenter:octicons.megaphone" />
      <h3>最新公告</h3>
    </div>
    <div class="notice-content">
      <k-markdown :source="content" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  content: string
}>()
</script>

<style scoped>
/* GitHub Dimmed 风格公告卡片 - 使用 Koishi 变量 */
.card {
  background: var(--k-card-bg);
  border-radius: 6px;
  padding: 1rem 1.25rem;
  border: 1px solid var(--k-color-border);
  transition: border-color 0.15s ease;
  height: 100%;
  min-height: 320px;
  overflow-y: auto;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

.card:hover {
  border-color: var(--fg3);
}

/* 左侧强调边框 - 直角 */
.notice-card {
  border-left: 3px solid var(--k-color-primary);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.875rem;
  color: var(--fg1);
  font-weight: 500;
}

.card-header h3 {
  margin: 0;
  font-size: 0.875rem;
}

.notice-content {
  color: var(--fg2);
  line-height: 1.6;
  font-size: 0.85rem;
  flex: 1;
  overflow-y: auto;
}

/* Markdown 内容样式覆盖 */
.notice-content :deep(p) {
  margin: 0.5rem 0;
}

.notice-content :deep(a) {
  color: var(--k-color-primary);
  text-decoration: none;
}

.notice-content :deep(a:hover) {
  text-decoration: underline;
}

.notice-content :deep(code) {
  font-family: var(--gh-font-mono, 'JetBrains Mono', 'SF Mono', Consolas, monospace);
  font-size: 0.8rem;
  background: var(--bg3);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.notice-content :deep(pre) {
  background: var(--bg1);
  padding: 0.75rem;
  border-radius: 4px;
  overflow-x: auto;
  font-size: 0.8rem;
}
</style>