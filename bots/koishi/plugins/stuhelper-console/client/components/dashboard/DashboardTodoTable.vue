<template>
  <div class="sh-table-shell">
    <table class="sh-table sh-dashboard-table">
      <thead>
        <tr>
          <th>类型</th>
          <th>事项</th>
          <th>信息</th>
          <th>状态</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody v-if="rows.length > 0">
        <tr v-for="row in rows" :key="row.id">
          <td>{{ kindLabel(row.kind) }}</td>
          <td>{{ row.title }}</td>
          <td>{{ row.meta }}</td>
          <td>
            <span class="sh-tag" :class="statusClass(row.kind)">
              {{ row.status }}
            </span>
          </td>
          <td>
            <button
              type="button"
              class="sh-btn sh-btn--ghost"
              @click="emit('open', row.target.section)"
            >
              进入
            </button>
          </td>
        </tr>
      </tbody>
      <tbody v-else>
        <tr>
          <td colspan="5" class="sh-dashboard-table__empty">当前没有待处理事项。</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import type { DashboardTodoRow } from '../../dashboard/model'
import type { ConsoleSectionId } from '../../sections'

defineProps<{
  rows: readonly DashboardTodoRow[]
}>()

const emit = defineEmits<{
  open: [section: ConsoleSectionId]
}>()

function kindLabel(kind: DashboardTodoRow['kind']) {
  return kind === 'review' ? '复核' : '认证'
}

function statusClass(kind: DashboardTodoRow['kind']) {
  return kind === 'review' ? 'sh-tag--warning' : 'sh-tag--primary'
}
</script>
