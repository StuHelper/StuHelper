<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.dashboard.title') }}</h1>
      <button
        class="flex items-center gap-1.5 px-3 py-1.5 text-sm text-text-secondary border border-border rounded-lg transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
        :disabled="loading"
        :aria-label="t('common.actions.refresh')"
        @click="fetchStats"
      >
        <RefreshCw class="size-3.5" :class="loading && 'animate-spin'" />
        {{ t('common.actions.refresh') }}
      </button>
    </div>

    <!-- Stats cards skeleton -->
    <div v-if="loading && !hasData" class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div v-for="i in 8" :key="i" class="h-[88px] bg-bg-card border border-border rounded-xl animate-pulse" />
    </div>

    <!-- Stats cards -->
    <div v-else class="grid grid-cols-2 md:grid-cols-4 gap-4">
      <div
        v-for="(card, index) in statCards"
        :key="card.key"
        class="flex items-center gap-4 bg-bg-card border border-border rounded-xl p-5 shadow-card"
        :class="isVisible && 'animate-fade-in-up'"
        :style="isVisible ? getStyle(index) : { opacity: 0 }"
      >
        <div
          class="size-10 rounded-full flex items-center justify-center shrink-0"
          :class="card.bgClass"
        >
          <component :is="card.icon" class="size-5" :class="card.iconClass" />
        </div>
        <div class="min-w-0">
          <div class="text-2xl font-semibold tabular-nums text-text-primary" :class="card.highlight && 'text-warning'">
            <CountTo :end-val="card.value" :duration="1200" separator="," />
          </div>
          <div class="text-xs text-text-muted truncate">{{ card.label }}</div>
        </div>
      </div>
    </div>

    <!-- Quick actions -->
    <div v-if="quickActions.length > 0" class="space-y-3">
      <h2 class="text-sm font-semibold text-text-secondary uppercase tracking-wider">{{ t('admin.dashboard.quickActions') }}</h2>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <component
          :is="action.to ? 'router-link' : 'button'"
          v-for="action in quickActions"
          :key="action.key"
          v-bind="action.to ? { to: action.to } : { type: 'button' }"
          class="relative flex items-center gap-3 p-4 bg-bg-card border border-border rounded-xl shadow-card text-left text-text-primary transition-all duration-fast hover:shadow-md hover:border-primary/30"
          :class="action.to ? 'no-underline' : ''"
          @click="action.onClick?.()"
        >
          <component :is="action.icon" class="size-5" :class="action.iconClass" />
          <span class="text-sm font-medium">{{ action.label }}</span>
          <span
            v-if="action.badge && action.badge > 0"
            class="absolute top-3 right-3 min-w-[20px] h-5 px-1.5 flex items-center justify-center bg-danger text-text-inverse text-xs font-medium rounded-full"
          >
            {{ action.badge }}
          </span>
        </component>
      </div>
    </div>

    <!-- Recent logs -->
    <div v-if="canViewLogs" class="space-y-3">
      <div class="flex items-center justify-between">
        <h2 class="text-sm font-semibold text-text-secondary uppercase tracking-wider">{{ t('admin.dashboard.recentLogs') }}</h2>
        <router-link
          :to="{ name: 'admin-logs' }"
          class="text-xs text-primary no-underline hover:text-primary-light transition-colors duration-fast"
        >
          {{ t('admin.dashboard.viewAll') }}
        </router-link>
      </div>
      <div v-if="recentLogs.length > 0" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
        <div class="divide-y divide-border">
          <div
            v-for="log in recentLogs"
            :key="log.id"
            class="flex items-center gap-3 px-4 py-3"
          >
            <div class="size-2 rounded-full shrink-0" :class="actionDotClass(log.action)" />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 text-sm">
                <span class="font-medium text-text-primary">{{ log.adminUsername }}</span>
                <span
                  class="px-1.5 py-0.5 text-xs rounded-md font-medium"
                  :class="actionBadgeClass(log.action)"
                >
                  {{ log.action }}
                </span>
                <span class="text-text-muted truncate">{{ log.resourceType }} #{{ log.resourceID.slice(0, 8) }}</span>
              </div>
            </div>
            <span class="text-xs text-text-muted whitespace-nowrap" :title="formatAbsolute(log.createdAt)">
              {{ formatRelative(log.createdAt) }}
            </span>
          </div>
        </div>
      </div>
      <div v-else class="text-sm text-text-muted text-center py-6">{{ t('admin.logs.empty') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  MessageSquare, CheckCircle, EyeOff, Trash2,
  Flag, Shield, TrendingUp, Calendar,
  RefreshCw, Download
} from 'lucide-vue-next'
import { CountTo } from '@cimom/vben-effects-common-ui'
import { api } from '@/api'
import { useStaggerAnimation } from '@/composables/useStaggerAnimation'
import { useToast } from '@/composables/useToast'
import { useAsyncData } from '@/composables/useAsyncData'
import { formatRelativeTime, formatAbsoluteTime } from '@/utils/date'
import { useAuthStore } from '@/stores/auth'
import {
  ADMIN_LOGS_VIEW,
  ADMIN_REPORTS_MANAGE,
  ADMIN_REVIEWS_MANAGE,
  hasCapability,
} from '@stuhelper/shared/constants'

const { t, locale } = useI18n()
const toast = useToast()
const authStore = useAuthStore()
const { isVisible, getStyle } = useStaggerAnimation(50)

const canManageReviews = computed(() =>
  hasCapability(authStore.user?.capabilities ?? [], ADMIN_REVIEWS_MANAGE),
)
const canManageReports = computed(() =>
  hasCapability(authStore.user?.capabilities ?? [], ADMIN_REPORTS_MANAGE),
)
const canViewLogs = computed(() =>
  hasCapability(authStore.user?.capabilities ?? [], ADMIN_LOGS_VIEW),
)

const { data: dashboardData, loading, execute: fetchStats } = useAsyncData(async () => {
  const [statsRes, logsRes] = await Promise.all([
    api.admin.getStats(),
    canViewLogs.value
      ? api.admin.getLogs({ page: 1, pageSize: 5 })
      : Promise.resolve({ data: { data: { list: [] } } })
  ])
  return {
    stats: statsRes.data?.data || {
      totalReviews: 0,
      publishedReviews: 0,
      hiddenReviews: 0,
      deletedReviews: 0,
      pendingReports: 0,
      totalReports: 0,
      todayReviews: 0,
      weekReviews: 0
    },
    recentLogs: logsRes.data?.data?.list || []
  }
})

const stats = computed(() => dashboardData.value?.stats || {
  totalReviews: 0,
  publishedReviews: 0,
  hiddenReviews: 0,
  deletedReviews: 0,
  pendingReports: 0,
  totalReports: 0,
  todayReviews: 0,
  weekReviews: 0
})
const recentLogs = computed(() => dashboardData.value?.recentLogs || [])
const hasData = computed(() => dashboardData.value !== null)
const quickActions = computed(() => {
  const actions: Array<{
    key: string
    label: string
    icon: typeof Download
    iconClass: string
    badge?: number
    to?: { name: string }
    onClick?: () => void
  }> = []

  if (canManageReviews.value) {
    actions.push({
      key: 'reviews',
      label: t('admin.dashboard.manageReviews'),
      icon: MessageSquare,
      iconClass: 'text-primary',
      to: { name: 'admin-reviews' },
    })
    actions.push({
      key: 'export',
      label: t('admin.dashboard.exportData'),
      icon: Download,
      iconClass: 'text-success',
      onClick: handleExport,
    })
  }

  if (canManageReports.value) {
    actions.push({
      key: 'reports',
      label: t('admin.dashboard.handleReports'),
      icon: Flag,
      iconClass: 'text-accent',
      badge: stats.value.pendingReports,
      to: { name: 'admin-reports' },
    })
  }

  return actions
})

const statCards = computed(() => [
  {
    key: 'total',
    icon: MessageSquare,
    bgClass: 'bg-primary/10',
    iconClass: 'text-primary',
    label: t('admin.dashboard.totalReviews'),
    value: stats.value.totalReviews,
    highlight: false
  },
  {
    key: 'published',
    icon: CheckCircle,
    bgClass: 'bg-success/10',
    iconClass: 'text-success',
    label: t('admin.dashboard.publishedReviews'),
    value: stats.value.publishedReviews,
    highlight: false
  },
  {
    key: 'hidden',
    icon: EyeOff,
    bgClass: 'bg-warning/10',
    iconClass: 'text-warning',
    label: t('admin.dashboard.hiddenReviews'),
    value: stats.value.hiddenReviews,
    highlight: false
  },
  {
    key: 'deleted',
    icon: Trash2,
    bgClass: 'bg-danger/10',
    iconClass: 'text-danger',
    label: t('admin.dashboard.deletedReviews'),
    value: stats.value.deletedReviews,
    highlight: false
  },
  {
    key: 'pending',
    icon: Flag,
    bgClass: 'bg-accent/10',
    iconClass: 'text-accent',
    label: t('admin.dashboard.pendingReports'),
    value: stats.value.pendingReports,
    highlight: stats.value.pendingReports > 0
  },
  {
    key: 'reports',
    icon: Shield,
    bgClass: 'bg-bg-secondary',
    iconClass: 'text-text-secondary',
    label: t('admin.dashboard.totalReports'),
    value: stats.value.totalReports,
    highlight: false
  },
  {
    key: 'today',
    icon: TrendingUp,
    bgClass: 'bg-primary/10',
    iconClass: 'text-primary',
    label: t('admin.dashboard.todayReviews'),
    value: stats.value.todayReviews,
    highlight: false
  },
  {
    key: 'week',
    icon: Calendar,
    bgClass: 'bg-primary-light/10',
    iconClass: 'text-primary-light',
    label: t('admin.dashboard.weekReviews'),
    value: stats.value.weekReviews,
    highlight: false
  }
])

function actionBadgeClass(action: string) {
  const map: Record<string, string> = {
    hide: 'bg-warning/10 text-warning',
    restore: 'bg-success/10 text-success',
    delete: 'bg-danger/10 text-danger',
    edit: 'bg-primary/10 text-primary'
  }
  return map[action] || 'bg-bg-secondary text-text-secondary'
}

function actionDotClass(action: string) {
  const map: Record<string, string> = {
    hide: 'bg-warning',
    restore: 'bg-success',
    delete: 'bg-danger',
    edit: 'bg-primary'
  }
  return map[action] || 'bg-text-muted'
}

const formatRelative = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)
const formatAbsolute = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

async function handleExport() {
  try {
    const res = await api.admin.exportReviews()
    if (!res.data) return
    const url = URL.createObjectURL(new Blob([res.data]))
    const a = document.createElement('a')
    a.href = url
    a.download = `reviews-export-${new Date().toISOString().slice(0, 10)}.ndjson`
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.error(t('common.error.loadFailed'))
  }
}

defineExpose({ refresh: fetchStats })
</script>
