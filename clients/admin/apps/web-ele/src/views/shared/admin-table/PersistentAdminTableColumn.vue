<script setup lang="ts">
import { computed, inject, useAttrs } from 'vue';

import { ElTableColumn } from 'element-plus';

import { persistentAdminTableKey } from './context';

type TableColumnSlotProps = {
  $index?: number;
  [key: string]: unknown;
  column?: unknown;
  row?: any;
};

defineOptions({ inheritAttrs: false });

const props = defineProps<{
  columnKey: string;
  defaultMinWidth?: number | string;
  defaultWidth?: number | string;
}>();

defineSlots<{
  [name: string]: ((props: TableColumnSlotProps) => unknown) | undefined;
  default?: (props: TableColumnSlotProps) => unknown;
  header?: (props: TableColumnSlotProps) => unknown;
}>();

const table = inject(persistentAdminTableKey, null);
const attrs = useAttrs();
const width = computed(() => {
  return table?.columnWidth(props.columnKey) ?? props.defaultWidth;
});
const minWidth = computed(() => {
  if (props.defaultMinWidth !== undefined) {
    return props.defaultMinWidth;
  }
  const inheritedMinWidth = attrs['min-width'];
  return typeof inheritedMinWidth === 'number' ||
    typeof inheritedMinWidth === 'string'
    ? inheritedMinWidth
    : undefined;
});

function normalizeSlotProps(slotProps: unknown): TableColumnSlotProps {
  return typeof slotProps === 'object' && slotProps !== null
    ? (slotProps as TableColumnSlotProps)
    : {};
}

function shouldRenderSlot(slotName: string, slotProps: unknown) {
  if (slotName !== 'default') {
    return true;
  }
  if (
    typeof slotProps !== 'object' ||
    slotProps === null ||
    !('row' in slotProps)
  ) {
    return false;
  }

  const row = (slotProps as { row?: unknown }).row;
  if (typeof row === 'object' && row !== null) {
    return Object.keys(row).length > 0;
  }
  return row !== undefined && row !== null;
}
</script>

<template>
  <ElTableColumn
    v-bind="$attrs"
    :column-key="columnKey"
    :min-width="minWidth"
    resizable
    :width="width"
  >
    <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
      <slot
        v-if="shouldRenderSlot(String(slotName), slotProps)"
        :name="slotName"
        v-bind="normalizeSlotProps(slotProps)"
      ></slot>
    </template>
  </ElTableColumn>
</template>
