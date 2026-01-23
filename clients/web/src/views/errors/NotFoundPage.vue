<template>
  <div class="error-page">
    <div class="error-content">
      <div class="error-icon">
        <svg viewBox="0 0 120 120" fill="none">
          <circle cx="60" cy="60" r="56" stroke="currentColor" stroke-width="2" opacity="0.2"/>
          <circle cx="60" cy="60" r="40" stroke="currentColor" stroke-width="2" opacity="0.3"/>
          <text x="60" y="72" text-anchor="middle" fill="currentColor" font-size="32" font-weight="700">404</text>
        </svg>
      </div>
      <h1 class="error-title">页面不存在</h1>
      <p class="error-desc">你访问的页面可能已被移除或地址有误</p>
      <div class="error-actions">
        <router-link to="/" class="btn-primary">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
            <polyline points="9,22 9,12 15,12 15,22"/>
          </svg>
          返回首页
        </router-link>
        <button class="btn-secondary" @click="goBack">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M19 12H5M12 19l-7-7 7-7"/>
          </svg>
          返回上页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

const goBack = () => {
  if (window.history.length > 1) {
    router.back()
  } else {
    router.push('/')
  }
}
</script>

<style scoped>
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  background: var(--bg-primary);
}

.error-content {
  text-align: center;
  max-width: 400px;
}

.error-icon {
  width: 120px;
  height: 120px;
  margin: 0 auto var(--space-6);
  color: var(--accent);
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-10px); }
}

.error-title {
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 var(--space-3);
}

.error-desc {
  font-size: var(--text-base);
  color: var(--text-muted);
  margin: 0 0 var(--space-8);
}

.error-actions {
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
  background: var(--accent);
  color: var(--bg-primary);
}

.btn-primary:hover {
  background: var(--accent-hover);
  transform: translateY(-2px);
}

.btn-secondary {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.btn-secondary:hover {
  border-color: var(--border-light);
  color: var(--text-primary);
}

.btn-primary svg,
.btn-secondary svg {
  width: 18px;
  height: 18px;
}

@media (max-width: 480px) {
  .error-actions {
    flex-direction: column;
  }
  .btn-primary,
  .btn-secondary {
    width: 100%;
    justify-content: center;
  }
}
</style>
