<script setup lang="ts">
import { ref } from 'vue'
import type { Component } from 'vue'
import { RouterLink } from 'vue-router'

interface Props {
  title: string
  description: string
  icon: Component
  color: string
  stats?: string
  to?: string
}

defineProps<Props>()

const prefersReducedMotion = typeof window !== 'undefined'
  && window.matchMedia('(prefers-reduced-motion: reduce)').matches
const hasHover = typeof window !== 'undefined'
  && window.matchMedia('(hover: hover)').matches
const tiltDisabled = prefersReducedMotion || !hasHover

const cardRef = ref<HTMLElement>()
const cardStyle = ref<Record<string, string>>({
  transform: 'perspective(1000px) rotateX(0deg) rotateY(0deg) scale(1)'
})

const onHover = (e: MouseEvent) => {
  const target = e.currentTarget
  if (tiltDisabled || !(target instanceof HTMLElement)) return
  const rect = target.getBoundingClientRect()
  const x = e.clientX - rect.left
  const y = e.clientY - rect.top
  const centerX = rect.width / 2
  const centerY = rect.height / 2
  const rotateX = (y - centerY) / 10
  const rotateY = (centerX - x) / 10

  cardStyle.value = {
    transform: `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) scale(1.05)`
  }
}

const onLeave = () => {
  cardStyle.value = {
    transform: 'perspective(1000px) rotateX(0deg) rotateY(0deg) scale(1)'
  }
}
</script>

<template>
  <component
    :is="to ? RouterLink : 'div'"
    ref="cardRef"
    class="feature-card"
    :class="{ 'feature-card--link': to }"
    :style="{ '--gradient-color': color, ...cardStyle }"
    :to="to"
    @mouseenter="onHover"
    @mousemove="onHover"
    @mouseleave="onLeave"
  >
    <div class="feature-card__icon">
      <component :is="icon" class="w-8 h-8" />
    </div>
    <h3 class="feature-card__title">{{ title }}</h3>
    <p class="feature-card__description">{{ description }}</p>
    <div v-if="stats" class="feature-card__stats">{{ stats }}</div>
  </component>
</template>

<style scoped>
.feature-card {
  position: relative;
  padding: 2rem;
  background: var(--color-bg-glass);
  backdrop-filter: blur(12px) saturate(180%);
  -webkit-backdrop-filter: blur(12px) saturate(180%);
  border-radius: 1rem;
  border: 1px solid transparent;
  background-clip: padding-box;
  transform-style: preserve-3d;
  transition: all 0.3s ease;
}

.feature-card--link {
  display: block;
  color: inherit;
  cursor: pointer;
  text-decoration: none;
}

.feature-card--link:focus-visible {
  outline: 3px solid var(--color-primary);
  outline-offset: 4px;
}

.feature-card::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 1rem;
  padding: 1px;
  background: linear-gradient(135deg, var(--gradient-color), transparent);
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  mask-composite: exclude;
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  pointer-events: none;
}

.feature-card__icon {
  display: inline-flex;
  padding: 0.75rem;
  background: linear-gradient(135deg, var(--gradient-color), rgba(255, 255, 255, 0.5));
  border-radius: 0.75rem;
  color: white;
  margin-bottom: 1rem;
}

.feature-card__title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 0.5rem;
}

.feature-card__description {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  line-height: 1.6;
}

.feature-card__stats {
  margin-top: 1rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--gradient-color);
}
</style>
