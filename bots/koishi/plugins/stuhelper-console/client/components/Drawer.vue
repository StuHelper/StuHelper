<template>
  <el-drawer
    class="sh-drawer"
    :model-value="open"
    :with-header="false"
    :append-to-body="true"
    :destroy-on-close="true"
    size="440px"
    @close="emit('close')"
    @closed="restoreFocus"
  >
    <div class="sh-drawer__content" :aria-labelledby="title ? headingId : undefined">
      <header v-if="title || subtitle" class="sh-drawer__head">
        <div>
          <h3 :id="headingId" class="sh-drawer__title">{{ title }}</h3>
          <p v-if="subtitle" class="sh-drawer__subtitle">{{ subtitle }}</p>
        </div>
        <el-button class="sh-button sh-button--ghost sh-button--sm" @click="emit('close')">
          关闭
        </el-button>
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
    </div>
    <template v-if="$slots.footer" #footer>
      <div class="sh-drawer__foot">
        <slot name="footer" />
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

import type { DrawerSection } from '../ui-state'

const props = withDefaults(
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

const headingId = `sh-drawer-title-${Math.random().toString(36).slice(2, 8)}`
const lastActiveElement = ref<HTMLElement | null>(null)

watch(
  () => props.open,
  (open, previousOpen) => {
    if (!open || previousOpen) return
    if (document.activeElement instanceof HTMLElement) {
      lastActiveElement.value = document.activeElement
    }
  },
)

function restoreFocus() {
  lastActiveElement.value?.focus()
  lastActiveElement.value = null
}
</script>

<style scoped>
.sh-drawer__content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

:deep(.sh-drawer .el-drawer) {
  max-width: min(440px, 88vw);
  background: var(--sh-surface-0);
}

:deep(.sh-drawer .el-drawer__body) {
  padding: 0;
}

:deep(.sh-drawer .el-drawer__footer) {
  padding: 0;
}
</style>
