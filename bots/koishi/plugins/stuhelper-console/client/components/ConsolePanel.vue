<template>
  <section class="sh-section" :class="{ 'sh-section--accent': tone === 'accent' }">
    <header v-if="title || eyebrow || $slots['head-meta']" class="sh-section__head">
      <div class="sh-section__head-copy">
        <span v-if="eyebrow" class="sh-section__eyebrow">{{ eyebrow }}</span>
        <h2 v-if="title" class="sh-section__title">{{ title }}</h2>
        <p v-if="description" class="sh-section__description">{{ description }}</p>
      </div>
      <div v-if="meta || $slots['head-meta']" class="sh-section__meta">
        <slot name="head-meta">
          <span class="sh-tag sh-tag--neutral">{{ meta }}</span>
        </slot>
      </div>
    </header>
    <div class="sh-section__body" :class="{ 'sh-section__body--flush': flush }">
      <slot />
    </div>
  </section>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    eyebrow?: string
    title?: string
    description?: string
    meta?: string
    tone?: 'neutral' | 'accent'
    flush?: boolean
  }>(),
  {
    eyebrow: '',
    title: '',
    description: '',
    meta: '',
    tone: 'neutral',
    flush: false,
  },
)
</script>
