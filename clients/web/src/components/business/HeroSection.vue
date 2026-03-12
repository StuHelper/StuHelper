<script setup lang="ts">
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
        <Button variant="primary" size="lg" @click="emit('explore')">
          开始探索
        </Button>
        <Button variant="secondary" size="lg" @click="emit('learnMore')">
          了解更多
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
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #334155 100%);
  overflow: hidden;
}

.dark .hero-section {
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
  color: white;
  margin-bottom: 1.5rem;
  line-height: 1.2;
}

.hero-subtitle {
  font-size: 1.25rem;
  color: rgba(255, 255, 255, 0.9);
  margin-bottom: 2.5rem;
  animation: fadeIn 0.8s ease-out 0.3s both;
}

.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
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
</style>
