<template>
  <EmptyState v-if="rows.length === 0" :title="emptyTitle" :body="emptyBody" />
  <div v-else class="sh-table-shell">
    <table class="sh-table">
      <thead>
        <tr>
          <th
            v-for="column in columns"
            :key="column.key"
            :style="column.width ? { width: column.width } : undefined"
            :class="headClass(column)"
          >
            {{ column.label }}
          </th>
          <th v-if="hasActions" class="sh-queue-table__actions-head">{{ actionsLabel }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in rows"
          :key="row.id"
          data-clickable="true"
          :aria-selected="selectedId === row.id"
          @click="emit('select', row.id)"
        >
          <td
            v-for="column in columns"
            :key="`${row.id}-${column.key}`"
            :class="cellClass(column, row)"
          >
            <slot
              :name="`cell-${column.key}`"
              :row="row"
              :column="column"
              :value="row.cells[column.key]"
            >
              <SeverityTag
                v-if="isTagCell(row.cells[column.key])"
                :label="normalizeCell(row.cells[column.key]).text"
                :intent="normalizeCell(row.cells[column.key]).tone"
              />
              <div v-else-if="normalizeCell(row.cells[column.key]).secondary" class="sh-queue-table__stack">
                <div :class="{ 'sh-table__mono': normalizeCell(row.cells[column.key]).mono }">
                  {{ normalizeCell(row.cells[column.key]).text }}
                </div>
                <div
                  class="sh-table__id"
                  :class="{ 'sh-table__mono': normalizeCell(row.cells[column.key]).mono }"
                >
                  {{ normalizeCell(row.cells[column.key]).secondary }}
                </div>
              </div>
              <span v-else :class="{ 'sh-table__mono': normalizeCell(row.cells[column.key]).mono }">
                {{ normalizeCell(row.cells[column.key]).text }}
              </span>
            </slot>
          </td>
          <td v-if="hasActions" class="sh-table__actions">
            <button
              v-for="action in row.actions ?? []"
              :key="action.key"
              type="button"
              class="sh-btn sh-btn--sm"
              :class="actionClass(action.tone)"
              :disabled="action.disabled"
              @click.stop="emit('action', { rowId: row.id, action: action.key })"
            >
              {{ action.label }}
            </button>
          </td>
        </tr>
      </tbody>
    </table>
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

function normalizeCell(cell: QueueTableCell | undefined): QueueTableCellObject {
  if (typeof cell === 'string' || typeof cell === 'number') {
    return { text: String(cell) }
  }

  return cell ?? { text: '—' }
}

function isTagCell(cell: QueueTableCell | undefined) {
  const value = normalizeCell(cell)
  return Boolean(value.tone) && !value.secondary
}

function headClass(column: QueueTableColumn) {
  return {
    'sh-queue-table__right': column.align === 'right',
  }
}

function cellClass(column: QueueTableColumn, row: QueueTableRow) {
  return {
    'sh-queue-table__right': column.align === 'right',
    'sh-table__mono': normalizeCell(row.cells[column.key]).mono && !normalizeCell(row.cells[column.key]).secondary,
  }
}

function actionClass(tone: QueueTableAction['tone']) {
  return {
    'sh-btn--ghost': !tone || tone === 'ghost',
    'sh-btn--primary': tone === 'primary',
    'sh-btn--danger': tone === 'danger',
  }
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
