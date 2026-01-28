<template>
  <div class="home-page">
    <!-- Hero Section -->
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">{{ $t('home.title') }}</h1>
        <p class="hero-subtitle">{{ $t('home.subtitle') }}</p>
        <div class="hero-actions">
          <router-link to="/review" class="btn-primary">
            {{ $t('home.explore') }}
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 12h14M12 5l7 7-7 7"/>
            </svg>
          </router-link>
          <router-link to="/review/post" class="btn-secondary">
            {{ $t('home.postReview') }}
          </router-link>
        </div>
      </div>
    </section>

    <!-- Features Section -->
    <section class="features">
      <div class="feature-card active" @click="goTo('/review')">
        <div class="feature-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
        </div>
        <div class="feature-info">
          <h3>{{ $t('home.features.review.title') }}</h3>
          <p>{{ $t('home.features.review.desc') }}</p>
        </div>
        <span class="feature-badge available">{{ $t('common.status.available') }}</span>
      </div>

      <div class="feature-card" @click="goTo('/resource')">
        <div class="feature-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
          </svg>
        </div>
        <div class="feature-info">
          <h3>{{ $t('home.features.resource.title') }}</h3>
          <p>{{ $t('home.features.resource.desc') }}</p>
        </div>
        <span class="feature-badge coming">{{ $t('common.status.coming') }}</span>
      </div>

      <div class="feature-card" @click="goTo('/spoc')">
        <div class="feature-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="2" y="3" width="20" height="14" rx="2"/>
            <path d="M8 21h8M12 17v4"/>
          </svg>
        </div>
        <div class="feature-info">
          <h3>{{ $t('home.features.spoc.title') }}</h3>
          <p>{{ $t('home.features.spoc.desc') }}</p>
        </div>
        <span class="feature-badge coming">{{ $t('common.status.coming') }}</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goTo = (path: string) => {
  router.push(path)
}
</script>

<style scoped>
.home-page {
  min-height: 100vh;
  padding: var(--space-6);
}

.hero {
  max-width: 600px;
  margin: 0 auto;
  padding: var(--space-16) 0;
  text-align: center;
}

.hero-title {
  font-size: var(--text-4xl);
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: var(--space-3);
  letter-spacing: -0.02em;
}

.hero-subtitle {
  font-size: var(--text-lg);
  color: var(--text-secondary);
  margin-bottom: var(--space-8);
}

.hero-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: center;
}

.btn-primary,
.btn-secondary {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5);
  font-size: var(--text-sm);
  font-weight: 600;
  border-radius: var(--radius-md);
  text-decoration: none;
  transition: all var(--duration-fast);
}

.btn-primary {
  background: var(--brand-primary);
  color: white;
}

.btn-primary:hover {
  background: var(--brand-primary-dark);
  transform: translateY(-1px);
}

.btn-primary svg {
  width: 16px;
  height: 16px;
}

.btn-secondary {
  background: var(--bg-secondary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--bg-hover);
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}

/* Features */
.features {
  max-width: 900px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-4);
}

.feature-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--space-3);
  padding: var(--space-6);
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--duration-base);
}

.feature-card:hover {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-4px);
}

.feature-card.active {
  border-color: var(--brand-primary);
}

.feature-icon {
  width: 56px;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  color: var(--brand-primary);
}

.feature-icon svg {
  width: 28px;
  height: 28px;
}

.feature-info h3 {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--space-1);
}

.feature-info p {
  font-size: var(--text-sm);
  color: var(--text-secondary);
  line-height: 1.4;
}

.feature-badge {
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-xs);
  font-weight: 500;
  border-radius: var(--radius-full);
}

.feature-badge.available {
  background: #dcfce7;
  color: #16a34a;
}

.feature-badge.coming {
  background: #fef3c7;
  color: #d97706;
}

@media (prefers-color-scheme: dark) {
  .feature-badge.available {
    background: rgba(34, 197, 94, 0.2);
    color: #4ade80;
  }
  .feature-badge.coming {
    background: rgba(251, 191, 36, 0.2);
    color: #fbbf24;
  }
}

@media (max-width: 640px) {
  .hero {
    padding: var(--space-10) 0;
  }
  .hero-title {
    font-size: var(--text-3xl);
  }
  .hero-actions {
    flex-direction: column;
  }
  .btn-primary,
  .btn-secondary {
    width: 100%;
    justify-content: center;
  }
  .features {
    grid-template-columns: 1fr;
  }
}
</style>
