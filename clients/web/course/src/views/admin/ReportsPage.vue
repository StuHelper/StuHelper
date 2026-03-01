<template>
  <div>
    <h1 class="mb-4 font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.reports.title') }}</h1>

    <div class="flex gap-2 mb-4">
      <button
        v-for="opt in statusOptions"
        :key="opt.value"
        class="px-3 py-1 bg-transparent border border-transparent rounded-full text-text-muted text-sm cursor-pointer transition-all duration-fast hover:text-text-primary"
        :class="status === opt.value && '!border-border !text-text-primary !font-medium'"
        @click="status = opt.value"
      >
        {{ opt.label }}
      </button>
    </div>

    <div v-if="loading" class="text-center p-8 text-text-muted">{{ t('common.actions.loading') }}</div>

    <div v-else-if="reports.length > 0" class="flex flex-col gap-3">
      <div
        v-for="report in reports"
        :key="report.id"
        class="flex items-center justify-between py-4 border-b border-border"
      >
        <div>
          <p class="m-0 mb-1 text-text-primary">{{ report.reason }}</p>
          <span class="text-xs text-text-muted">{{ formatTime(report.createdAt) }}</span>
        </div>
        <div v-if="report.status === 'pending'" class="flex gap-2">
          <button
            class="px-3 py-1 bg-transparent border border-border rounded-full text-text-secondary text-sm cursor-pointer transition-all duration-fast"
            @click="handleProcess(report.id, 'reject')"
          >
            {{ t('admin.reports.reject') }}
          </button>
          <button
            class="px-3 py-1 bg-text-primary border-none rounded-full text-bg-base text-sm cursor-pointer transition-all duration-fast hover:bg-primary"
            @click="handleProcess(report.id, 'hide_review')"
          >
            {{ t('admin.reports.hideReview') }}
          </button>
        </div>
        <span
          v-else
          class="px-2 py-1 rounded-sm text-xs text-text-muted"
        >
          {{ report.status === 'resolved' ? t('admin.reports.statusResolved') : t('admin.reports.statusRejected') }}
        </span>
      </div>
    </div>

    <button
      v-if="!loading && hasMore"
      class="mt-4 mx-auto block px-4 py-2 bg-bg-secondary border border-border rounded-lg text-sm text-text-secondary cursor-pointer transition-colors duration-fast hover:bg-bg-hover"
      @click="loadMore"
    >
      {{ t('common.actions.loadMore') }}
    </button>

    <EmptyState v-else-if="!loading && reports.length === 0" :title="t('admin.reports.empty')" />
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
const page = ref(1)
const pageSize = 20
const total = ref(0)
const hasMore = computed(() => reports.value.length < total.value)

const fetchReports = async () => {
  loading.value = true
  try {
    const res = await getReports(status.value, page.value, pageSize)
    reports.value = page.value === 1
      ? (res.data?.list || [])
      : [...reports.value, ...(res.data?.list || [])]
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const loadMore = () => {
  if (loading.value || !hasMore.value) return
  page.value++
  fetchReports()
}

const handleProcess = async (id: string, action: ProcessReportParams['action']) => {
  try {
    await processReport(id, { action })
    toast.success(t('admin.reports.processSuccess'))
    fetchReports()
  } catch {
    toast.error(t('admin.reports.processFailed'))
  }
}

const formatTime = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

watch(status, () => {
  page.value = 1
  fetchReports()
})
onMounted(fetchReports)
</script>
