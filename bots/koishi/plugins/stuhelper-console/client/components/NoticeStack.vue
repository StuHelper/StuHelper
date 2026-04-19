<template>
  <transition-group
    tag="div"
    name="sh-notice"
    class="sh-notice-stack"
    aria-live="polite"
  >
    <article
      v-for="item in items"
      :key="item.id"
      class="sh-notice"
      :class="`sh-notice--${item.kind}`"
      role="status"
    >
      <span class="sh-notice__icon">{{ item.kind === 'success' ? '✓' : '!' }}</span>
      <span>{{ item.message }}</span>
      <button
        type="button"
        class="sh-notice__close"
        aria-label="关闭通知"
        @click="emit('dismiss', item.id)"
      >
        ✕
      </button>
    </article>
  </transition-group>
</template>

<script setup lang="ts">
export interface NoticeItem {
  id: string
  kind: 'success' | 'error'
  message: string
}

defineProps<{ items: NoticeItem[] }>()

const emit = defineEmits<{ dismiss: [id: string] }>()
</script>

<style scoped>
.sh-notice-enter-active,
.sh-notice-leave-active {
  transition:
    opacity var(--sh-dur-base) var(--sh-ease),
    transform var(--sh-dur-base) var(--sh-ease);
}

.sh-notice-enter-from { opacity: 0; transform: translateY(8px); }
.sh-notice-leave-to { opacity: 0; transform: translateX(16px); }
</style>
