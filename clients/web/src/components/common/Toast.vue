<template>
  <Teleport to="body">
    <TransitionGroup
      name="toast"
      tag="div"
      class="fixed top-[calc(var(--navbar-height)+1rem)] right-4 z-[60] flex flex-col gap-2 pointer-events-none"
    >
      <div
        v-for="item in toasts"
        :key="item.id"
        class="flex items-center gap-2 py-3 px-4 bg-bg-card border border-border rounded-lg shadow-lg text-sm text-text-primary pointer-events-auto max-w-[360px]"
      >
        <Check
          v-if="item.type === 'success'"
          :size="16"
          class="shrink-0 text-[var(--color-rating-5)]"
        />
        <XCircle
          v-else-if="item.type === 'error'"
          :size="16"
          class="shrink-0 text-[var(--color-rating-1)]"
        />
        <AlertTriangle
          v-else-if="item.type === 'warning'"
          :size="16"
          class="shrink-0 text-[var(--color-rating-3)]"
        />
        <Info
          v-else
          :size="16"
          class="shrink-0 text-primary"
        />
        <span class="flex-1 leading-snug">{{ item.message }}</span>
        <button
          class="text-text-muted text-lg leading-none px-1 cursor-pointer hover:text-text-primary"
          @click="remove(item.id)"
        >
          &times;
        </button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup lang="ts">
import { Check, XCircle, AlertTriangle, Info } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { toasts, remove } = useToast()
</script>

<style scoped>
.toast-enter-active {
  animation: fadeInDown var(--duration-slow) var(--ease-out);
}

.toast-leave-active {
  animation: fadeIn var(--duration-fast) var(--ease-out) reverse;
}

.toast-move {
  transition: transform var(--duration-slow) var(--ease-out);
}
</style>
