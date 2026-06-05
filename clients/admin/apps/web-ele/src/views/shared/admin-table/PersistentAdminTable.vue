<script setup lang="ts">
import { computed, provide, ref } from 'vue';

import { ElTable } from 'element-plus';

import { persistentAdminTableKey } from './context';

defineOptions({ inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    loading?: boolean;
    tableKey: string;
  }>(),
  { loading: false },
);
const emit = defineEmits<{
  (
    e: 'headerDragend',
    newWidth: number,
    oldWidth: number,
    column: unknown,
    event: MouseEvent,
  ): void;
}>();
const STORAGE_PREFIX = 'stuhelper.admin.tableColumns';
const MIN_COLUMN_WIDTH = 48;

type DragColumn = {
  columnKey?: string;
  id?: string;
  label?: string;
  property?: string;
  rawColumnKey?: string;
};

const storageKey = `${STORAGE_PREFIX}.${props.tableKey}`;
const columnWidths = ref<Record<string, number>>(readStoredWidths(storageKey));

const tableStyle = computed(() => ({ width: '100%' }));

provide(persistentAdminTableKey, {
  columnWidth: (columnKey) => columnWidths.value[columnKey],
});

function handleHeaderDragend(
  newWidth: number,
  oldWidth: number,
  column: DragColumn,
  event: MouseEvent,
) {
  const key = getColumnKey(column);
  if (key) {
    columnWidths.value = {
      ...columnWidths.value,
      [key]: normalizeWidth(newWidth),
    };
    writeStoredWidths(storageKey, columnWidths.value);
  }
  emit('headerDragend', newWidth, oldWidth, column, event);
}

function getColumnKey(column: DragColumn) {
  return String(
    column.columnKey ??
      column.property ??
      column.rawColumnKey ??
      column.label ??
      '',
  ).trim();
}

function normalizeWidth(width: number) {
  return Math.max(MIN_COLUMN_WIDTH, Math.round(width));
}

function readStoredWidths(key: string) {
  const storage = getLocalStorage();
  if (!storage) return {};
  const raw = readStorageItem(storage, key);
  if (!raw) return {};

  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(raw) as Record<string, unknown>;
  } catch (error) {
    void error;
    removeStorageItem(storage, key);
    return {};
  }
  return Object.fromEntries(
    Object.entries(parsed)
      .filter(
        (entry): entry is [string, number] => typeof entry[1] === 'number',
      )
      .map(([columnKey, width]) => [columnKey, normalizeWidth(width)]),
  );
}

function writeStoredWidths(key: string, widths: Record<string, number>) {
  const storage = getLocalStorage();
  if (!storage) return;
  try {
    storage.setItem(key, JSON.stringify(widths));
  } catch (error) {
    void error;
  }
}

function getLocalStorage() {
  if (typeof window === 'undefined') return null;
  try {
    return window.localStorage ?? null;
  } catch (error) {
    void error;
    return null;
  }
}

function readStorageItem(storage: Storage, key: string) {
  try {
    return storage.getItem(key);
  } catch (error) {
    void error;
    return null;
  }
}

function removeStorageItem(storage: Storage, key: string) {
  try {
    storage.removeItem(key);
  } catch (error) {
    void error;
  }
}
</script>

<template>
  <div class="persistent-admin-table">
    <ElTable
      v-bind="$attrs"
      v-loading="loading"
      border
      class="admin-data-table"
      :style="tableStyle"
      @header-dragend="handleHeaderDragend"
    >
      <slot></slot>
    </ElTable>
  </div>
</template>

<style>
.persistent-admin-table {
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.admin-data-table {
  --el-table-row-hover-bg-color: var(--el-fill-color-light);

  max-width: 100%;
  font-size: 13px;
}

.admin-data-table .el-table__inner-wrapper {
  min-width: 0;
}

.admin-data-table .el-scrollbar__wrap {
  overscroll-behavior-x: contain;
}

.admin-data-table .el-table__header th {
  height: 44px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
}

.admin-data-table .el-table__row .el-table__cell {
  padding: 12px 0;
  vertical-align: top;
}

.admin-data-table .el-table__column-resize-proxy {
  border-left-color: var(--el-color-primary);
}

.admin-data-table .el-table-fixed-column--right {
  box-shadow: -8px 0 14px -14px var(--el-text-color-primary);
}

.admin-id-token,
.admin-sub-id {
  display: inline-flex;
  max-width: 100%;
  font-family:
    ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono',
    monospace;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.admin-id-token {
  padding: 3px 7px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border-radius: 6px;
}

.admin-sub-id {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.admin-cell-title {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 600;
  line-height: 1.45;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.admin-cell-body,
.admin-cell-muted {
  line-height: 1.45;
}

.admin-cell-body {
  display: -webkit-box;
  overflow: hidden;
  -webkit-line-clamp: 2;
  color: var(--el-text-color-regular);
  -webkit-box-orient: vertical;
}

.admin-cell-muted {
  color: var(--el-text-color-secondary);
}

.admin-number {
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.admin-action-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
</style>
