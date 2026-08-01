<template>
  <section
    :aria-atomic="true"
    :aria-busy="state === 'loading' ? 'true' : undefined"
    :aria-live="liveMode"
    class="admission-state-panel"
    :class="toneClass"
    :data-state="state"
    :role="liveRole"
  >
    <div class="admission-state-panel__heading">
      <span class="admission-state-panel__icon" aria-hidden="true">
        <component :is="icon" class="admission-state-panel__icon-svg" />
      </span>
      <div class="admission-state-panel__copy">
        <h2>{{ title }}</h2>
        <p
          v-for="line in descriptionLines"
          :key="line"
        >
          {{ line }}
        </p>
      </div>
    </div>

    <div v-if="$slots.default" class="admission-state-panel__body">
      <slot />
    </div>

    <div v-if="$slots.actions" class="admission-state-panel__actions">
      <slot name="actions" />
    </div>

    <div v-if="$slots.help" class="admission-state-panel__help">
      <slot name="help" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'

const props = withDefaults(
  defineProps<{
    description?: string | string[]
    icon: Component
    state: string
    title: string
    tone?: 'danger' | 'info' | 'neutral' | 'success' | 'warning'
  }>(),
  {
    description: '',
    tone: 'neutral',
  },
)

const descriptionLines = computed(() => {
  if (Array.isArray(props.description)) return props.description.filter(Boolean)
  return props.description ? [props.description] : []
})

const liveMode = computed(() => (props.tone === 'danger' ? 'assertive' : 'polite'))
const liveRole = computed(() => (props.tone === 'danger' ? 'alert' : 'status'))
const toneClass = computed(() => `admission-state-panel--${props.tone}`)
</script>

<style scoped>
.admission-state-panel {
  background: #ffffff;
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  display: grid;
  gap: 16px;
}

.admission-state-panel__heading {
  display: grid;
  gap: 12px;
  grid-template-columns: auto minmax(0, 1fr);
}

.admission-state-panel__icon {
  align-items: center;
  background: #f1f5f9;
  border-radius: 8px;
  color: #334155;
  display: inline-flex;
  height: 44px;
  justify-content: center;
  width: 44px;
}

.admission-state-panel__icon-svg {
  height: 22px;
  width: 22px;
}

.admission-state-panel__copy {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.admission-state-panel__copy h2,
.admission-state-panel__copy p {
  margin: 0;
}

.admission-state-panel__copy h2 {
  color: #0f172a;
  font-size: 20px;
  font-weight: 700;
  line-height: 28px;
}

.admission-state-panel__copy p,
.admission-state-panel__body,
.admission-state-panel__help {
  color: #475569;
  font-size: 14px;
  line-height: 22px;
}

.admission-state-panel__body,
.admission-state-panel__help {
  display: grid;
  gap: 12px;
}

.admission-state-panel__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.admission-state-panel--info .admission-state-panel__icon {
  background: #eff6ff;
  color: #1d4ed8;
}

.admission-state-panel--success .admission-state-panel__icon {
  background: #ecfdf5;
  color: #047857;
}

.admission-state-panel--warning .admission-state-panel__icon {
  background: #fffbeb;
  color: #b45309;
}

.admission-state-panel--danger .admission-state-panel__icon {
  background: #fef2f2;
  color: #b91c1c;
}

@media (max-width: 520px) {
  .admission-state-panel__heading {
    grid-template-columns: 1fr;
  }
}
</style>
