<script setup lang="ts">
import { ref } from 'vue'
import type { Component } from 'vue'

interface Props {
  title: string
  description: string
  icon: Component
  color: string
  stats?: string
}

defineProps<Props>()

const cardRef = ref<HTMLElement>()
const cardStyle = ref<Record<string, string>>({
  transform: 'perspective(1000px) rotateX(0deg) rotateY(0deg) scale(1)'
})

const onHover = (e: MouseEvent) => {
  if (!cardRef.value) return
  const rect = cardRef.value.getBoundingClientRect()
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
  <div
    ref="cardRef"
    class="feature-card"
    :style="{ '--gradient-color': color, ...cardStyle }"
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
  </div>
</template>

<style scoped>
.feature-card {
  position: relative;
  padding: 2rem;
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(12px);
  border-radius: 1rem;
  border: 1px solid transparent;
  background-clip: padding-box;
  cursor: pointer;
  transform-style: preserve-3d;
  transition: all 0.3s ease;
}

.feature-card::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 1rem;
  padding: 1px;
  background: linear-gradient(135deg, var(--gradient-color), transparent);
  -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask-composite: exclude;
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
  color: #1a1a1a;
  margin-bottom: 0.5rem;
}

.feature-card__description {
  font-size: 0.875rem;
  color: #666;
  line-height: 1.6;
}

.feature-card__stats {
  margin-top: 1rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--gradient-color);
}

/* 暗色模式 */
@media (prefers-color-scheme: dark) {
  .feature-card {
    background: rgba(30, 30, 30, 0.7);
  }

  .feature-card__title {
    color: #f0f0f0;
  }

  .feature-card__description {
    color: #a0a0a0;
  }
}
</style>
