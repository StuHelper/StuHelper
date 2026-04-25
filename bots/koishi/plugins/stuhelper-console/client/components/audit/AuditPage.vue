<template>
  <div class="sh-view">
    <WorkspaceSection
      title="筛选条件"
      description="统一检索事件与举报，定位最近发生了什么。"
    >
      <AuditFilters
        v-model:query="query"
        v-model:kind="kind"
        :total="rows.length"
        :event-count="eventCount"
        :report-count="reportCount"
      />
    </WorkspaceSection>

    <WorkspaceSection
      title="检索结果"
      description="按时间倒序展示，点击行查看详情。"
      :meta="`${rows.length} 条`"
      flush
    >
      <AuditResultsTable
        :rows="rows"
        :selected-id="selectedId"
        @select="handleSelect"
      />
    </WorkspaceSection>
  </div>
</template>

<script setup lang="ts">
import type { AuditFilterKind, AuditRow } from '../../audit/model'
import AuditFilters from './AuditFilters.vue'
import AuditResultsTable from './AuditResultsTable.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'

const props = withDefaults(
  defineProps<{
    rows: readonly AuditRow[]
    eventCount: number
    reportCount: number
    selectedId?: string
  }>(),
  {
    selectedId: '',
  },
)

const emit = defineEmits<{
  inspect: [row: AuditRow]
}>()

const query = defineModel<string>('query', { required: true })
const kind = defineModel<AuditFilterKind>('kind', { required: true })

function handleSelect(rowId: string) {
  const row = props.rows.find((item) => item.id === rowId)
  if (!row) return
  emit('inspect', row)
}
</script>
