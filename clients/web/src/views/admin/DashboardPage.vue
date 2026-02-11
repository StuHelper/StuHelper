<template>
  <div class="dashboard">
    <h1>{{ t('admin.dashboard.title') }}</h1>

    <div v-if="loading" class="stats-grid">
      <div v-for="i in 4" :key="i" class="stat-card skeleton" />
    </div>

    <div v-else class="stats-grid">
      <div class="stat-card">
        <span class="stat-label">{{ t('admin.dashboard.totalReviews') }}</span>
        <span class="stat-value">{{ stats.totalReviews }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('admin.dashboard.pendingReports') }}</span>
        <span class="stat-value warning">{{ stats.pendingReports }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('admin.dashboard.todayReviews') }}</span>
        <span class="stat-value">{{ stats.todayReviews }}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">{{ t('admin.dashboard.weekReviews') }}</span>
        <span class="stat-value">{{ stats.weekReviews }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAdminStats, type AdminStats } from '@/api/admin'

const { t } = useI18n()

const loading = ref(true)
const stats = ref<AdminStats>({
  totalReviews: 0,
  publishedReviews: 0,
  hiddenReviews: 0,
  deletedReviews: 0,
  pendingReports: 0,
  totalReports: 0,
  todayReviews: 0,
  weekReviews: 0
})

onMounted(async () => {
  try {
    const res = await getAdminStats()
    if (res.data) stats.value = res.data
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.dashboard h1 {
  margin: 0 0 var(--space-6);
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-4);
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.stat-card.skeleton {
  height: 100px;
  animation: pulse 1.5s infinite;
}

.stat-label {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

.stat-value {
  font-size: var(--text-2xl);
  font-weight: var(--weight-semibold);
  font-variant-numeric: tabular-nums;
  color: var(--text-primary);
}

.stat-value.warning {
  color: var(--brand-accent);
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
