<template>
  <div class="error-page">
    <div class="error-content">
      <div class="error-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
      </div>
      <h1 class="error-title">{{ t('errors.loadError.title') }}</h1>
      <p class="error-desc">{{ t('errors.loadError.description') }}</p>
      <button class="btn-primary" @click="reload">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M23 4v6h-6M1 20v-6h6"/>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
        </svg>
        {{ t('errors.loadError.reload') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const reload = () => {
  window.location.reload()
}
</script>

<style scoped>
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  background: var(--bg-base);
}

.error-content {
  text-align: center;
  max-width: 400px;
}

.error-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto var(--space-6);
  color: var(--text-muted);
  opacity: 0.4;
}

.error-icon svg {
  width: 100%;
  height: 100%;
}

.error-title {
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0 0 var(--space-3);
}

.error-desc {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0 0 var(--space-6);
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-5);
  background: var(--gradient-brand);
  color: white;
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  border-radius: var(--radius-full);
  transition: all var(--duration-fast);
}

.btn-primary:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

.btn-primary svg {
  width: 18px;
  height: 18px;
}
</style>
