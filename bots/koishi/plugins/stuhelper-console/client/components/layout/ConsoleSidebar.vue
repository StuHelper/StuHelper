<template>
  <aside class="sh-sidebar">
    <nav class="sh-sidebar__nav" aria-label="群管中心分区">
      <button
        v-for="item in props.items"
        :key="item.id"
        type="button"
        class="sh-sidebar__item"
        :data-active="item.id === props.active"
        :aria-current="item.id === props.active ? 'page' : undefined"
        @click="emit('select', item.id)"
      >
        <span class="sh-sidebar__label">{{ item.label }}</span>
        <span v-if="item.count !== null" class="sh-sidebar__count sh-num">
          {{ item.count }}
        </span>
      </button>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import type { SidebarItem } from '../../sidebar-items'
import type { ConsoleSectionId } from '../../sections'

const props = defineProps<{
  items: SidebarItem[]
  active: ConsoleSectionId
}>()

const emit = defineEmits<{
  select: [value: ConsoleSectionId]
}>()
</script>
