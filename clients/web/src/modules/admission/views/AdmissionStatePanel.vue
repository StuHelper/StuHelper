<template>
  <section
    class="admission-state-panel"
    :class="toneClass"
    :data-state="state"
  >
    <div class="admission-state-panel__heading">
      <span
        class="admission-state-panel__icon"
        :class="toneIconClass"
        aria-hidden="true"
      >
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

const toneClass = computed(() => `admission-state-panel--${props.tone}`)
const toneIconClass = computed(() => `join-tone-${props.tone}`)
</script>

<style scoped>
/*
 * 品牌玻璃风状态面板：本组件渲染在 AdmissionShell 的 .join-glass 面板内，
 * 因此自身不再叠加玻璃底，只负责层级（色调气泡图标 → 标题 → 辅文 → 槽位）。
 * data-state 与 h2 标题是测试契约，不可改动。
 */
.admission-state-panel {
  display: grid;
  gap: 18px;
}

.admission-state-panel__heading {
  align-items: start;
  display: grid;
  gap: 14px;
  grid-template-columns: auto minmax(0, 1fr);
}

/* 色调背景/文字色来自全局 .join-tone-*（join-theme.css） */
.admission-state-panel__icon {
  align-items: center;
  border-radius: 14px;
  box-shadow: inset 0 1px 0 var(--join-glass-highlight);
  display: inline-flex;
  height: 46px;
  justify-content: center;
  width: 46px;
}

.admission-state-panel__icon-svg {
  height: 22px;
  width: 22px;
}

/* loading 态的虚线圆环缓速旋转（仅 transform，遵循 reduced-motion 全局开关） */
.admission-state-panel[data-state="loading"] .admission-state-panel__icon-svg {
  animation: admission-state-panel-spin 2.4s linear infinite;
}

@keyframes admission-state-panel-spin {
  to {
    transform: rotate(360deg);
  }
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
  color: var(--join-ink);
  font-size: 21px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 28px;
}

.admission-state-panel__copy p,
.admission-state-panel__body,
.admission-state-panel__help {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
}

.admission-state-panel__copy p {
  max-width: 60ch;
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

@media (max-width: 420px) {
  .admission-state-panel__heading {
    grid-template-columns: 1fr;
  }
}
</style>
