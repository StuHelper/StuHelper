<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import Button from '../ui/Button.vue'
import TypeWriter from '../animated/TypeWriter.vue'
import ParticleBackground from '../animated/ParticleBackground.vue'

interface Props {
  title: string
  subtitle: string
}

defineProps<Props>()

const emit = defineEmits<{
  explore: []
  learnMore: []
}>()

const { t } = useI18n()
</script>

<template>
  <section class="hero-section">
    <ParticleBackground />

    <div class="hero-content">
      <TypeWriter :text="title" class="hero-title" />

      <p class="hero-subtitle">
        {{ subtitle }}
      </p>

      <div class="hero-actions">
        <Button variant="primary" size="lg" class="hero-cta-1 press-spring" @click="emit('explore')">
          {{ t('home.explore') }}
        </Button>
        <Button variant="secondary" size="lg" class="hero-cta-2 press-spring" @click="emit('learnMore')">
          {{ t('common.actions.learnMore') }}
        </Button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero-section {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--color-bg-base) 0%, var(--color-bg-card) 50%, var(--color-bg-elevated) 100%);
  overflow: hidden;
}

[data-theme="dark"] .hero-section,
:root:where(:not([data-theme="light"]):not([data-theme="dark"])) .hero-section {
  background: linear-gradient(135deg, #020617 0%, #0f172a 50%, #1e293b 100%);
}

.hero-content {
  position: relative;
  z-index: 10;
  text-align: center;
  padding: 2rem;
  max-width: 800px;
}

.hero-title {
  font-size: 3.5rem;
  font-weight: 800;
  background: linear-gradient(135deg, var(--color-primary), var(--color-accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 1.5rem;
  line-height: 1.2;
}

.hero-subtitle {
  font-size: 1.25rem;
  color: var(--color-text-secondary);
  margin-bottom: 2.5rem;
  opacity: 0;
  animation: heroFadeUp 0.6s var(--ease-out) 0.4s forwards;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

.hero-cta-1 {
  opacity: 0;
  animation: heroFadeUp 0.5s var(--ease-spring) 0.7s forwards;
}

.hero-cta-2 {
  opacity: 0;
  animation: heroFadeUp 0.5s var(--ease-spring) 0.9s forwards;
}

@keyframes heroFadeUp {
  from {
    opacity: 0;
    transform: translateY(24px) scale(0.96);
    filter: blur(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
    filter: blur(0);
  }
}

@media (max-width: 768px) {
  .hero-title {
    font-size: 2.5rem;
  }

  .hero-subtitle {
    font-size: 1rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-subtitle,
  .hero-cta-1,
  .hero-cta-2 {
    animation: none;
    opacity: 1;
    transform: none;
  }
}
</style>
