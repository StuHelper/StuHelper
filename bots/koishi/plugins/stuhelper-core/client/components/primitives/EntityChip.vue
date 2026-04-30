<template>
  <button
    type="button"
    class="sh-entity"
    :class="{ 'sh-entity--inline': inline }"
    :data-kind="kind"
    :title="`${kind === 'user' ? '用户' : '群组'} ${id}${name ? ` · ${name}` : ''}`"
    @click="handleClick"
  >
    <span v-if="name" class="sh-entity__name">{{ name }}</span>
    <span class="sh-entity__id">{{ displayId }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import { useAppShell, type EntityKind } from '../../composables/use-app-shell'

const props = withDefaults(
  defineProps<{
    kind: EntityKind
    id: string
    name?: string
    /** Optional guild context for user chips (used when navigating from user → review/warns). */
    guildId?: string
    /** Inline mode renders without background, suitable for table cells. */
    inline?: boolean
  }>(),
  { inline: false },
)

const shell = useAppShell()

const displayId = computed(() => {
  if (!props.id) return '—'
  if (props.id.length > 16) return `${props.id.slice(0, 14)}…`
  return props.id
})

function handleClick() {
  shell.openEntity({
    kind: props.kind,
    id: props.id,
    name: props.name,
    guildId: props.guildId,
  })
}
</script>
