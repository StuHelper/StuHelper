<template>
  <QueueTable
    :columns="columns"
    :rows="tableRows"
    :selected-id="selectedId"
    empty-title="没有匹配记录"
    empty-body="调整筛选条件后重试。"
    @select="emit('select', $event)"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { AuditRow } from '../../audit/model'
import { describeLevel, formatTimestamp } from '../../use-console-page'
import QueueTable from '../queue/QueueTable.vue'

const columns = [
  { key: 'createdAt', label: '时间' },
  { key: 'kind', label: '类型' },
  { key: 'summary', label: '摘要' },
  { key: 'member', label: '成员' },
  { key: 'target', label: '目标' },
  { key: 'level', label: '级别' },
] as const

const props = withDefaults(
  defineProps<{
    rows: readonly AuditRow[]
    selectedId?: string
  }>(),
  {
    selectedId: '',
  },
)

const emit = defineEmits<{
  select: [rowId: string]
}>()

const tableRows = computed(() =>
  props.rows.map((row) => ({
    id: row.id,
    cells: {
      createdAt: {
        text: formatTimestamp(row.createdAt),
        mono: true,
      },
      kind: {
        text: row.kind === 'event' ? '事件' : '举报',
        tone: row.kind === 'event' ? 'info' : 'warning',
      },
      summary: {
        text: row.summary,
        secondary: row.detail,
      },
      member: {
        text: row.memberId,
        mono: row.memberId !== '—',
      },
      target: {
        text: row.target,
        mono: row.target !== '—',
      },
      level: {
        text: row.level,
        tone: describeLevel(row.level),
      },
    },
  })),
)
</script>
