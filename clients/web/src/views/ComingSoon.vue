<template>
  <div class="coming-soon">
    <div class="content">
      <div class="glow-dot"></div>
      <h1 class="gradient-text">{{ t('teaching.comingSoon') }}</h1>
      <p>{{ moduleName }}</p>
      <button class="back-btn" @click="goBack">{{ t('nav.home') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const moduleName = computed(() => {
  if (route.meta.titleKey) {
    return t(route.meta.titleKey as string)
  }
  return ''
})

const goBack = () => {
  router.push('/')
}
</script>

<style scoped>
.coming-soon {
  min-height: 80vh;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn var(--duration-base) var(--ease-out);
}

.content {
  text-align: center;
  max-width: 360px;
}

.glow-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--brand-primary);
  margin: 0 auto var(--space-4);
  animation: glowPulse 2s ease-in-out infinite;
}

h1 {
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  margin: 0 0 var(--space-3) 0;
}

p {
  color: var(--text-muted);
  margin: 0 0 var(--space-8) 0;
  font-size: var(--text-sm);
}

.back-btn {
  padding: var(--space-2) var(--space-6);
  background: var(--gradient-brand);
  color: white;
  border: none;
  border-radius: var(--radius-full);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.back-btn:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}
</style>
