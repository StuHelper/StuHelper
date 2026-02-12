<template>
  <div>
    <h1 class="mb-6 font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.dashboard.title') }}</h1>

    <div v-if="loading" class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">
      <div v-for="i in 4" :key="i" class="h-[100px] bg-bg-card border border-border rounded-lg animate-pulse" />
    </div>

    <div v-else class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">
      <div class="flex flex-col gap-2 p-4 bg-bg-card border border-border rounded-lg">
        <span class="text-sm text-text-muted">{{ t('admin.dashboard.totalReviews') }}</span>
        <span class="text-2xl font-semibold tabular-nums text-text-primary">{{ stats.totalReviews }}</span>
      </div>
      <div class="flex flex-col gap-2 p-4 bg-bg-card border border-border rounded-lg">
        <span class="text-sm text-text-muted">{{ t('admin.dashboard.pendingReports') }}</span>
        <span class="text-2xl font-semibold tabular-nums text-accent">{{ stats.pendingReports }}</span>
      </div>
      <div class="flex flex-col gap-2 p-4 bg-bg-card border border-border rounded-lg">
        <span class="text-sm text-text-muted">{{ t('admin.dashboard.todayReviews') }}</span>
        <span class="text-2xl font-semibold tabular-nums text-text-primary">{{ stats.todayReviews }}</span>
      </div>
      <div class="flex flex-col gap-2 p-4 bg-bg-card border border-border rounded-lg">
        <span class="text-sm text-text-muted">{{ t('admin.dashboard.weekReviews') }}</span>
        <span class="text-2xl font-semibold tabular-nums text-text-primary">{{ stats.weekReviews }}</span>
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
