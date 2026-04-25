<template>
  <div class="sh-table-shell">
    <el-table
      :data="rows"
      row-key="id"
      class="sh-grid-table"
      @row-click="handleRowClick"
      :row-class-name="rowClassName"
    >
      <template #empty>
        <EmptyState :title="emptyTitle" :body="emptyBody" />
      </template>

      <el-table-column
        v-for="column in columns"
        :key="column.key"
        :label="column.label"
        :prop="column.key"
        :align="column.align ?? 'left'"
        :width="column.width"
      >
        <template #default="{ row }">
          <slot
            :name="`cell-${column.key}`"
            :row="row"
            :column="column"
            :value="row.cells[column.key]"
          >
            <SeverityTag
              v-if="isTagCell(row.cells[column.key])"
              :label="resolveCell(row, column.key).text"
              :intent="resolveCell(row, column.key).tone"
            />
            <div v-else-if="resolveCell(row, column.key).secondary" class="sh-queue-table__stack">
              <div :class="{ 'sh-table__mono': resolveCell(row, column.key).mono }">
                {{ resolveCell(row, column.key).text }}
              </div>
              <div
                class="sh-table__id"
                :class="{ 'sh-table__mono': resolveCell(row, column.key).mono }"
              >
                {{ resolveCell(row, column.key).secondary }}
              </div>
            </div>
            <span v-else :class="{ 'sh-table__mono': resolveCell(row, column.key).mono }">
              {{ resolveCell(row, column.key).text }}
            </span>
          </slot>
        </template>
      </el-table-column>

      <el-table-column
        v-if="hasActions"
        :label="actionsLabel"
        align="right"
        class-name="sh-queue-table__actions-column"
      >
        <template #default="{ row }">
          <div class="sh-table__actions">
            <el-button
              v-for="action in row.actions ?? []"
              :key="action.key"
              size="small"
              :type="buttonType(action.tone)"
              class="sh-button sh-button--sm"
              :class="buttonClass(action.tone)"
              :disabled="action.disabled"
              @click.stop="emit('action', { rowId: row.id, action: action.key })"
            >
              {{ action.label }}
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import EmptyState from '../EmptyState.vue'
import SeverityTag, { type TagIntent } from '../SeverityTag.vue'

interface QueueTableCellObject {
  text: string
  secondary?: string
  mono?: boolean
  tone?: TagIntent
}

type QueueTableCell = QueueTableCellObject | string | number

interface QueueTableColumn {
  key: string
  label: string
  width?: string
  align?: 'left' | 'right'
}

interface QueueTableAction {
  key: string
  label: string
  tone?: 'ghost' | 'primary' | 'danger'
  disabled?: boolean
}

interface QueueTableRow {
  id: string
  cells: Record<string, QueueTableCell>
  actions?: readonly QueueTableAction[]
}

const props = withDefaults(
  defineProps<{
    columns: readonly QueueTableColumn[]
    rows: readonly QueueTableRow[]
    selectedId?: string
    emptyTitle: string
    emptyBody: string
    actionsLabel?: string
  }>(),
  {
    selectedId: '',
    actionsLabel: '操作',
  },
)

const emit = defineEmits<{
  select: [rowId: string]
  action: [payload: { rowId: string; action: string }]
}>()

const hasActions = computed(() => props.rows.some((row) => (row.actions?.length ?? 0) > 0))

function handleRowClick(row: QueueTableRow) {
  emit('select', row.id)
}

function rowClassName({ row }: { row: QueueTableRow }) {
  return row.id === props.selectedId ? 'is-selected' : ''
}

function normalizeCell(cell: QueueTableCell | undefined): QueueTableCellObject {
  if (typeof cell === 'string' || typeof cell === 'number') {
    return { text: String(cell) }
  }

  return cell ?? { text: '—' }
}

function resolveCell(row: QueueTableRow, columnKey: string) {
  return normalizeCell(row.cells[columnKey])
}

function isTagCell(cell: QueueTableCell | undefined) {
  const value = normalizeCell(cell)
  return Boolean(value.tone) && !value.secondary
}

function buttonClass(tone: QueueTableAction['tone']) {
  return {
    'sh-button--ghost': !tone || tone === 'ghost',
    'sh-button--primary': tone === 'primary',
    'sh-button--danger': tone === 'danger',
  }
}

function buttonType(tone: QueueTableAction['tone']) {
  if (tone === 'primary') return 'primary'
  if (tone === 'danger') return 'danger'
  return undefined
}
</script>

<style scoped>
.sh-queue-table__actions-head,
.sh-queue-table__right {
  text-align: right;
}

.sh-queue-table__stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
</style>
