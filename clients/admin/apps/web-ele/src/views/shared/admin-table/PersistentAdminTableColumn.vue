<script setup lang="ts">
import { computed, inject } from 'vue';

import { ElTableColumn } from 'element-plus';

import { persistentAdminTableKey } from './context';

defineOptions({ inheritAttrs: false });

const props = defineProps<{
  columnKey: string;
  defaultMinWidth?: number | string;
  defaultWidth?: number | string;
}>();

const table = inject(persistentAdminTableKey, null);
const width = computed(() => {
  return table?.columnWidth(props.columnKey) ?? props.defaultWidth;
});
</script>

<template>
  <ElTableColumn
    v-bind="$attrs"
    :column-key="columnKey"
    :min-width="defaultMinWidth"
    resizable
    :width="width"
  >
    <template v-for="(_, slotName) in $slots" #[slotName]="slotProps">
      <slot :name="slotName" v-bind="slotProps ?? {}"></slot>
    </template>
  </ElTableColumn>
</template>
