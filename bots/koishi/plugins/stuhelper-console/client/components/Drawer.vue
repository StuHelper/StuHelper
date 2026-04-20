<template>
  <teleport to="body">
    <transition name="sh-drawer-fade">
      <div
        v-if="open"
        class="sh-drawer-mask"
        :data-open="String(open)"
        @click="emit('close')"
      />
    </transition>
    <aside class="sh-drawer" :data-open="String(open)" role="dialog" aria-modal="true">
      <header v-if="title || subtitle" class="sh-drawer__head">
        <div>
          <h3 class="sh-drawer__title">{{ title }}</h3>
          <p v-if="subtitle" class="sh-drawer__subtitle">{{ subtitle }}</p>
        </div>
        <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="emit('close')">关闭</button>
      </header>
      <div class="sh-drawer__body">
        <template v-if="sections.length > 0">
          <section
            v-for="section in sections"
            :key="section.key"
            class="sh-drawer__section"
          >
            <h4 class="sh-drawer__section-title">{{ section.title }}</h4>
            <dl class="sh-keylist">
              <template v-for="item in section.items" :key="`${section.key}-${item.label}`">
                <dt>{{ item.label }}</dt>
                <dd :class="{ 'sh-mono': item.mono }">{{ item.value }}</dd>
              </template>
            </dl>
          </section>
        </template>
        <slot v-else />
      </div>
      <footer v-if="$slots.footer" class="sh-drawer__foot">
        <slot name="footer" />
      </footer>
    </aside>
  </teleport>
</template>

<script setup lang="ts">
import type { DrawerSection } from '../ui-state'

withDefaults(
  defineProps<{
    open: boolean
    title?: string
    subtitle?: string
    sections?: readonly DrawerSection[]
  }>(),
  {
    title: '',
    subtitle: '',
    sections: () => [],
  },
)

const emit = defineEmits<{ close: [] }>()
</script>

<style scoped>
.sh-drawer-fade-enter-active,
.sh-drawer-fade-leave-active { transition: opacity var(--sh-dur-slow) var(--sh-ease); }
.sh-drawer-fade-enter-from,
.sh-drawer-fade-leave-to { opacity: 0; }
</style>
