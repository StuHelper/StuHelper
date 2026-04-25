<template>
  <header class="sh-workspace-head">
    <div class="sh-workspace-head__copy">
      <h1 class="sh-workspace-head__title">{{ title }}</h1>
      <p v-if="description" class="sh-workspace-head__description">
        {{ description }}
      </p>
      <div
        v-if="metaChips.length > 0 || $slots.meta"
        class="sh-workspace-head__meta"
      >
        <span
          v-for="chip in metaChips"
          :key="chip.text"
          class="sh-meta-chip"
          :class="{ 'sh-mono': chip.mono, 'sh-num': chip.numeric }"
        >
          {{ chip.text }}
        </span>
        <slot name="meta" />
      </div>
    </div>
    <div v-if="$slots.actions" class="sh-workspace-head__actions sh-btn-row">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface WorkspaceHeadChip {
  text: string
  mono?: boolean
  numeric?: boolean
}

const props = withDefaults(
  defineProps<{
    title: string
    description?: string
    chips?: ReadonlyArray<WorkspaceHeadChip | string>
  }>(),
  {
    description: '',
    chips: () => [],
  },
)

const metaChips = computed<WorkspaceHeadChip[]>(() =>
  props.chips.map((chip) =>
    typeof chip === 'string' ? { text: chip } : chip,
  ),
)
</script>
