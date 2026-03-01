<template>
  <div class="space-y-4">
    <!-- Header + filter -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.reports.title') }}</h1>
        <TabBar v-model="status" :tabs="statusTabs" />
      </div>
      <span class="text-sm text-text-muted">{{ t('admin.pagination.total', { total }) }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-4">
      <div v-for="i in 3" :key="i" class="h-36 bg-bg-card border border-border rounded-xl animate-pulse" />
    </div>

    <!-- Report cards -->
    <div v-else-if="reports.length > 0" class="space-y-4">
      <div
        v-for="(report, index) in reports"
        :key="report.id"
        class="bg-bg-card border border-border rounded-xl p-4 shadow-card"
        :class="isVisible && 'animate-fade-in-up'"
        :style="isVisible ? getStyle(index) : { opacity: 0 }"
      >
        <!-- Top: reason + time + status -->
        <div class="flex items-center justify-between gap-2 mb-3">
          <div class="flex items-center gap-2">
            <span class="px-2 py-0.5 text-xs font-medium rounded-md bg-danger/10 text-danger">
              {{ report.reason }}
            </span>
            <span class="text-xs text-text-muted" :title="formatAbsolute(report.createdAt)">
              {{ formatRelative(report.createdAt) }}
            </span>
          </div>
          <span
            class="px-2 py-0.5 text-xs font-medium rounded-md"
            :class="statusBadgeClass(report.status)"
          >
            {{ statusLabel(report.status) }}
          </span>
        </div>

        <!-- Middle: review preview -->
        <div v-if="report.review" class="bg-bg-secondary rounded-lg p-3 mb-3">
          <div class="flex items-center gap-2 mb-1.5">
            <span class="text-xs font-medium text-text-secondary">{{ report.review.courseName }}</span>
            <span v-if="report.review.teacherName" class="text-xs text-text-muted">· {{ report.review.teacherName }}</span>
          </div>
          <p class="text-sm text-text-primary m-0 line-clamp-3">{{ report.review.content }}</p>
        </div>

        <!-- Description -->
        <p v-if="report.description" class="text-xs text-text-muted m-0 mb-3">{{ report.description }}</p>

        <!-- Bottom: actions or result -->
        <div class="flex items-center justify-between">
          <div v-if="report.status === 'pending'" class="flex items-center gap-2">
            <button
              class="px-3 py-1.5 text-xs font-medium text-text-secondary border border-border rounded-lg transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
              @click="handleProcess(report.id, 'reject')"
            >
              {{ t('admin.reports.reject') }}
            </button>
            <button
              class="px-3 py-1.5 text-xs font-medium text-text-inverse bg-gradient-to-br from-primary to-accent rounded-lg transition-all duration-fast hover:shadow-glow-sm"
              @click="handleProcess(report.id, 'hide_review')"
            >
              {{ t('admin.reports.hideReview') }}
            </button>
          </div>
          <div v-else class="flex items-center gap-2 text-xs text-text-muted">
            <span v-if="report.resolvedBy">{{ t('admin.reports.processedBy') }}: {{ report.resolvedBy }}</span>
            <span v-if="report.resolvedAt" :title="formatAbsolute(report.resolvedAt)">· {{ formatRelative(report.resolvedAt) }}</span>
          </div>
          <span v-if="report.resolutionNote" class="text-xs text-text-muted italic">{{ report.resolutionNote }}</span>
        </div>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.reports.empty')" />

    <!-- Pagination -->
    <div v-if="total > 0" class="flex flex-col sm:flex-row items-center justify-between gap-3 text-sm">
      <span class="text-text-muted">{{ t('admin.pagination.total', { total }) }}</span>
      <div class="flex items-center gap-1">
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page <= 1"
          @click="page--"
        >{{ t('admin.pagination.prev') }}</button>
        <template v-for="p in visiblePages" :key="p">
          <span v-if="p === '...'" class="px-2 text-text-muted">...</span>
          <button
            v-else
            class="min-w-[36px] h-9 px-2 rounded-lg text-sm font-medium transition-colors duration-fast"
            :class="p === page ? 'bg-primary text-text-inverse' : 'text-text-secondary hover:bg-bg-hover'"
            @click="page = p as number"
          >{{ p }}</button>
        </template>
        <button
          class="px-3 py-1.5 border border-border rounded-lg text-text-secondary transition-colors duration-fast hover:bg-bg-hover disabled:opacity-40 disabled:cursor-not-allowed"
          :disabled="page >= totalPages"
          @click="page++"
        >{{ t('admin.pagination.next') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getReports, processReport } from '@/api/admin'
import type { Report, ProcessReportParams } from '@/types/admin'
import TabBar from '@/components/common/TabBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { useStaggerAnimation } from '@/composables/useStaggerAnimation'
import { formatRelativeTime, formatAbsoluteTime } from '@/utils/date'

const { t, locale } = useI18n()
const toast = useToast()
const { isVisible, getStyle } = useStaggerAnimation(50)

const statusTabs = computed(() => [
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

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const visiblePages = computed(() => {
  const tp = totalPages.value
  if (tp <= 5) return Array.from({ length: tp }, (_, i) => i + 1)
  const pages: (number | string)[] = []
  if (page.value <= 3) {
    pages.push(1, 2, 3, 4, '...', tp)
  } else if (page.value >= tp - 2) {
    pages.push(1, '...', tp - 3, tp - 2, tp - 1, tp)
  } else {
    pages.push(1, '...', page.value - 1, page.value, page.value + 1, '...', tp)
  }
  return pages
})

function statusBadgeClass(s: string) {
  if (s === 'pending') return 'bg-warning/10 text-warning'
  if (s === 'resolved') return 'bg-success/10 text-success'
  return 'bg-bg-secondary text-text-muted'
}

function statusLabel(s: string) {
  if (s === 'pending') return t('admin.reports.statusPending')
  if (s === 'resolved') return t('admin.reports.statusResolved')
  return t('admin.reports.statusRejected')
}

const formatRelative = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)
const formatAbsolute = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

const fetchReports = async () => {
  loading.value = true
  try {
    const res = await getReports(status.value, page.value, pageSize)
    reports.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
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

watch(status, () => {
  page.value = 1
  fetchReports()
})
watch(page, () => fetchReports())
onMounted(fetchReports)
</script>

<style scoped>
.line-clamp-3 {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
