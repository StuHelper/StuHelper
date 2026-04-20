<template>
  <div class="sh-toolbar sh-audit-filters">
    <div class="sh-audit-filters__stats">
      <span class="sh-toolbar__count">结果 {{ total }}</span>
      <span class="sh-toolbar__count">事件 {{ eventCount }}</span>
      <span class="sh-toolbar__count">举报 {{ reportCount }}</span>
    </div>
    <div class="sh-toolbar__spacer"></div>
    <div class="sh-audit-filters__controls">
      <label class="sh-field sh-audit-filters__field">
        <span class="sh-field__label">对象类型</span>
        <select v-model="kind" class="sh-select">
          <option value="all">全部</option>
          <option value="event">事件</option>
          <option value="report">举报</option>
        </select>
      </label>
      <label class="sh-field sh-audit-filters__field sh-audit-filters__field--wide">
        <span class="sh-field__label">关键词</span>
        <input
          v-model="query"
          class="sh-input"
          placeholder="检索成员、群号、摘要、级别或目标"
        />
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AuditFilterKind } from '../../audit/model'

defineProps<{
  total: number
  eventCount: number
  reportCount: number
}>()

const query = defineModel<string>('query', { required: true })
const kind = defineModel<AuditFilterKind>('kind', { required: true })
</script>

<style scoped>
.sh-audit-filters {
  padding: 0;
  gap: var(--sh-s-3);
}

.sh-audit-filters__stats,
.sh-audit-filters__controls {
  display: flex;
  align-items: flex-end;
  gap: var(--sh-s-3);
  flex-wrap: wrap;
}

.sh-audit-filters__field {
  min-width: 160px;
  flex: 0 1 180px;
}

.sh-audit-filters__field--wide {
  flex-basis: 320px;
}
</style>
