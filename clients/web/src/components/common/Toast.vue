<template>
  <Teleport to="body">
    <TransitionGroup name="toast" tag="div" class="toast-container">
      <div
        v-for="item in toasts"
        :key="item.id"
        class="toast-item"
        :class="[`toast-${item.type}`]"
      >
        <svg v-if="item.type === 'success'" class="toast-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="20,6 9,17 4,12" />
        </svg>
        <svg v-else-if="item.type === 'error'" class="toast-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" y1="9" x2="9" y2="15" />
          <line x1="9" y1="9" x2="15" y2="15" />
        </svg>
        <svg v-else class="toast-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="16" x2="12" y2="12" />
          <line x1="12" y1="8" x2="12.01" y2="8" />
        </svg>
        <span class="toast-msg">{{ item.message }}</span>
        <button class="toast-close" @click="remove(item.id)">&times;</button>
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { toasts, remove } = useToast()
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: calc(var(--gradient-bar-height) + var(--navbar-height) + var(--space-4));
  right: var(--space-4);
  z-index: var(--z-tooltip);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  pointer-events: none;
}

.toast-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  font-size: var(--text-sm);
  color: var(--text-primary);
  pointer-events: auto;
  max-width: 360px;
}

.toast-icon {
  flex-shrink: 0;
}

.toast-success .toast-icon { color: var(--rating-5); }
.toast-error .toast-icon { color: var(--rating-1); }
.toast-warning .toast-icon { color: var(--rating-3); }
.toast-info .toast-icon { color: var(--brand-primary); }

.toast-msg {
  flex: 1;
  line-height: var(--leading-snug);
}

.toast-close {
  color: var(--text-muted);
  font-size: var(--text-lg);
  line-height: 1;
  padding: 0 var(--space-1);
  cursor: pointer;
}

.toast-close:hover {
  color: var(--text-primary);
}

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
