<template>
  <div class="sh-table-shell">
    <el-table :data="rows" class="sh-dashboard-table" row-key="id">
      <template #empty>
        <div class="sh-dashboard-table__empty">当前没有待处理事项。</div>
      </template>

      <el-table-column label="类型">
        <template #default="{ row }">
          {{ kindLabel(row.kind) }}
        </template>
      </el-table-column>
      <el-table-column prop="title" label="事项" />
      <el-table-column prop="meta" label="信息" />
      <el-table-column label="状态">
        <template #default="{ row }">
          <SeverityTag :label="row.status" :intent="statusIntent(row.kind)" />
        </template>
      </el-table-column>
      <el-table-column label="操作" align="right">
        <template #default="{ row }">
          <el-button
            class="sh-button sh-button--ghost sh-button--sm"
            @click="emit('open', row.target)"
          >
            进入
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import type { DashboardTodoRow } from '../../dashboard/model'
import SeverityTag, { type TagIntent } from '../SeverityTag.vue'

defineProps<{
  rows: readonly DashboardTodoRow[]
}>()

const emit = defineEmits<{
  open: [target: DashboardTodoRow['target']]
}>()

function kindLabel(kind: DashboardTodoRow['kind']) {
  return kind === 'review' ? '复核' : '认证'
}

function statusIntent(kind: DashboardTodoRow['kind']): TagIntent {
  return kind === 'review' ? 'warning' : 'primary'
}
</script>
