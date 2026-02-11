<template>
  <div class="reports-page">
    <h1>{{ t('admin.reports.title') }}</h1>

    <div class="filter-bar">
      <button
        v-for="opt in statusOptions"
        :key="opt.value"
        class="filter-btn"
        :class="{ active: status === opt.value }"
        @click="status = opt.value"
      >
        {{ opt.label }}
      </button>
    </div>

    <div v-if="loading" class="loading">{{ t('common.actions.loading') }}</div>

    <div v-else-if="reports.length > 0" class="reports-list">
      <div v-for="report in reports" :key="report.id" class="report-card">
        <div class="report-info">
          <p class="reason">{{ report.reason }}</p>
          <span class="time">{{ formatTime(report.createdAt) }}</span>
        </div>
        <div v-if="report.status === 'pending'" class="actions">
          <button class="btn-reject" @click="handleProcess(report.id, 'reject')">
            {{ t('admin.reports.reject') }}
          </button>
          <button class="btn-hide" @click="handleProcess(report.id, 'hide_review')">
            {{ t('admin.reports.hideReview') }}
          </button>
        </div>
        <span v-else class="status-badge" :class="report.status">
          {{ report.status === 'resolved' ? t('admin.reports.statusResolved') : t('admin.reports.statusRejected') }}
        </span>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.reports.empty')" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getReports, processReport, type Report } from '@/api/admin'
import type { ProcessReportParams } from '@/types/admin'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { formatAbsoluteTime } from '@/utils/date'

const { t, locale } = useI18n()
const toast = useToast()

const statusOptions = computed(() => [
  { value: 'pending', label: t('admin.reports.statusPending') },
  { value: 'resolved', label: t('admin.reports.statusResolved') },
  { value: 'rejected', label: t('admin.reports.statusRejected') }
])

const status = ref('pending')
const loading = ref(true)
const reports = ref<Report[]>([])

const fetchReports = async () => {
  loading.value = true
  try {
    const res = await getReports(status.value)
    reports.value = res.data?.list || []
  } finally {
    loading.value = false
  }
}

const handleProcess = async (id: number, action: ProcessReportParams['action']) => {
  try {
    await processReport(id, { action })
    toast.success(t('admin.reports.processSuccess'))
    fetchReports()
  } catch {
    toast.error(t('admin.reports.processFailed'))
  }
}

const formatTime = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

watch(status, fetchReports)
onMounted(fetchReports)
</script>

<style scoped>
.reports-page h1 {
  margin: 0 0 var(--space-4);
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}

.filter-bar {
  display: flex;
  gap: var(--space-2);
  margin-bottom: var(--space-4);
}

.filter-btn {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  color: var(--text-muted);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.filter-btn:hover {
  color: var(--text-primary);
}

.filter-btn.active {
  border-color: var(--border);
  color: var(--text-primary);
  font-weight: var(--weight-medium);
}

.reports-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.report-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) 0;
  border-bottom: 1px solid var(--border);
}

.reason {
  margin: 0 0 var(--space-1);
  color: var(--text-primary);
}

.time {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.actions {
  display: flex;
  gap: var(--space-2);
}

.btn-reject,
.btn-hide {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-full);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.btn-reject {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.btn-hide {
  background: var(--text-primary);
  border: none;
  color: var(--bg-base);
}

.btn-hide:hover {
  background: var(--brand-primary);
}

.status-badge {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}

.status-badge.resolved {
  background: transparent;
  color: var(--text-muted);
}

.status-badge.rejected {
  background: transparent;
  color: var(--text-muted);
}

.loading {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-muted);
}
</style>
