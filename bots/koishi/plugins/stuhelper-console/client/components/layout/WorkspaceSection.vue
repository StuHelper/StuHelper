<template>
  <section class="sh-section" :class="{ 'sh-section--accent': tone === 'accent' }">
    <header
      v-if="title || description || meta || $slots.meta || $slots.actions"
      class="sh-section__head"
    >
      <div class="sh-section__head-copy">
        <h2 v-if="title" class="sh-section__title">{{ title }}</h2>
        <p v-if="description" class="sh-section__description">{{ description }}</p>
      </div>
      <div
        v-if="meta || $slots.meta || $slots.actions"
        class="sh-section__meta"
      >
        <slot name="meta">
          <SeverityTag v-if="meta" :label="meta" intent="neutral" />
        </slot>
        <slot name="actions" />
      </div>
    </header>
    <div class="sh-section__body" :class="{ 'sh-section__body--flush': flush }">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
import SeverityTag from '../SeverityTag.vue'

withDefaults(
  defineProps<{
    title?: string
    description?: string
    meta?: string
    tone?: 'neutral' | 'accent'
    flush?: boolean
  }>(),
  {
    title: '',
    description: '',
    meta: '',
    tone: 'neutral',
    flush: false,
  },
)
</script>
