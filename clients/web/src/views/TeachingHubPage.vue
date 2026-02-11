<template>
  <div class="teaching-hub">
    <header class="hub-header">
      <h1 class="hub-title">{{ t('teaching.title') }}</h1>
      <p class="hub-subtitle">{{ t('teaching.subtitle') }}</p>
    </header>

    <div class="module-grid">
      <router-link to="/review" class="module-card">
        <div class="module-icon review-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
          </svg>
        </div>
        <h2 class="module-name">{{ t('teaching.modules.review.name') }}</h2>
        <p class="module-desc">{{ t('teaching.modules.review.description') }}</p>
      </router-link>

      <router-link to="/teacher" class="module-card">
        <div class="module-icon teacher-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
            <circle cx="12" cy="7" r="4" />
          </svg>
        </div>
        <h2 class="module-name">{{ t('teaching.modules.teacher.name') }}</h2>
        <p class="module-desc">{{ t('teaching.modules.teacher.description') }}</p>
      </router-link>

      <div class="module-card disabled">
        <div class="module-icon spoc-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
          </svg>
        </div>
        <h2 class="module-name">{{ t('teaching.modules.spoc.name') }}</h2>
        <p class="module-desc">{{ t('teaching.modules.spoc.description') }}</p>
        <span class="coming-pill">{{ t('teaching.comingSoon') }}</span>
      </div>

      <div class="module-card disabled">
        <div class="module-icon resource-icon">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
            <polyline points="14 2 14 8 20 8" />
          </svg>
        </div>
        <h2 class="module-name">{{ t('teaching.modules.resource.name') }}</h2>
        <p class="module-desc">{{ t('teaching.modules.resource.description') }}</p>
        <span class="coming-pill">{{ t('teaching.comingSoon') }}</span>
      </div>
    </div>

    <footer v-if="stats" class="hub-stats">
      <span class="hub-stats__item font-mono">
        {{ t('teaching.stats.courses', { count: stats.courseCount }) }}
      </span>
      <span class="hub-stats__dot"></span>
      <span class="hub-stats__item font-mono">
        {{ t('teaching.stats.reviews', { count: stats.reviewCount }) }}
      </span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getStats, type Stats } from '@/api/course'

const { t } = useI18n()
const stats = ref<Stats | null>(null)

onMounted(async () => {
  try {
    const res = await getStats()
    stats.value = res.data
  } catch {
    stats.value = null
  }
})
</script>

<style scoped>
.teaching-hub {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--space-8) var(--space-6);
  animation: fadeIn var(--duration-base) var(--ease-out);
}

.hub-header {
  text-align: center;
  margin-bottom: var(--space-8);
}

.hub-title {
  font-family: var(--font-sans);
  font-size: var(--text-2xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
  margin: 0 0 var(--space-2) 0;
}

.hub-subtitle {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0;
}

.module-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.module-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  padding: var(--space-5);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-decoration: none;
  color: var(--text-primary);
  transition: border-color var(--duration-fast),
    box-shadow var(--duration-fast),
    transform var(--duration-fast);
  position: relative;
  cursor: pointer;
}

.module-card:hover:not(.disabled) {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
  transform: translateY(-2px);
}

.module-card.disabled {
  opacity: 0.6;
  cursor: default;
}

.module-icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
}

.review-icon {
  color: var(--brand-primary);
  background: color-mix(in srgb, var(--brand-primary) 10%, transparent);
}

.teacher-icon {
  color: var(--brand-accent);
  background: color-mix(in srgb, var(--brand-accent) 10%, transparent);
}

.spoc-icon {
  color: var(--text-muted);
  background: var(--bg-secondary);
}

.resource-icon {
  color: var(--text-muted);
  background: var(--bg-secondary);
}

.module-name {
  font-size: var(--text-base);
  font-weight: var(--weight-semibold);
  margin: 0 0 var(--space-1) 0;
}

.module-desc {
  font-size: var(--text-xs);
  color: var(--text-muted);
  margin: 0;
  line-height: var(--leading-relaxed);
}

.coming-pill {
  display: inline-block;
  margin-top: var(--space-3);
  padding: 2px 10px;
  font-size: var(--text-xs);
  color: var(--text-muted);
  background: var(--bg-secondary);
  border-radius: var(--radius-full);
}

.hub-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  margin-top: var(--space-8);
  color: var(--text-muted);
  font-size: var(--text-xs);
}

.hub-stats__dot {
  width: 3px;
  height: 3px;
  border-radius: var(--radius-full);
  background: var(--text-muted);
}

@media (max-width: 639px) {
  .teaching-hub {
    padding: var(--space-6) var(--space-4);
  }

  .module-grid {
    grid-template-columns: 1fr;
  }
}
</style>
