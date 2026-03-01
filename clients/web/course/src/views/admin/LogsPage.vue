<template>
  <div class="space-y-4">
    <!-- Header + filter -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.logs.title') }}</h1>
        <TabBar v-model="actionFilter" :tabs="actionTabs" />
      </div>
      <span class="text-sm text-text-muted">{{ t('admin.pagination.total', { total }) }}</span>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div v-for="i in 5" :key="i" class="h-14 border-b border-border animate-pulse" />
    </div>

    <!-- Table -->
    <div v-else-if="filteredLogs.length > 0" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-bg-secondary">
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.logs.operator') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.logs.action') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.logs.resource') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.logs.ipAddress') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.logs.time') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="log in filteredLogs"
              :key="log.id"
              class="border-t border-border transition-colors duration-fast hover:bg-bg-hover"
            >
              <td class="p-3 text-sm text-text-primary font-medium whitespace-nowrap">{{ log.adminUsername }}</td>
              <td class="p-3">
                <span
                  class="inline-flex px-2 py-0.5 text-xs font-medium rounded-md"
                  :class="actionBadgeClass(log.action)"
                >
                  {{ log.action }}
                </span>
              </td>
              <td class="p-3 text-sm text-text-secondary">
                <div class="flex items-center gap-1.5">
                  <component :is="resourceIcon(log.resourceType)" class="size-3.5 text-text-muted shrink-0" />
                  <span>{{ log.resourceType }}</span>
                  <span class="text-text-muted" :title="log.resourceID">#{{ log.resourceID.slice(0, 8) }}</span>
                </div>
              </td>
              <td class="p-3 text-xs text-text-muted hidden md:table-cell font-mono">{{ log.ipAddress || '-' }}</td>
              <td class="p-3 text-xs text-text-muted whitespace-nowrap" :title="formatAbsolute(log.createdAt)">
                {{ formatRelative(log.createdAt) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.logs.empty')" />

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
import { MessageSquare, Flag, FileText } from 'lucide-vue-next'
import { getOperationLogs, type OperationLog } from '@/api/admin'
import TabBar from '@/components/common/TabBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime, formatAbsoluteTime } from '@/utils/date'
import type { Component } from 'vue'

const { t, locale } = useI18n()
const toast = useToast()

const actionFilter = ref('all')
const loading = ref(true)
const logs = ref<OperationLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20

const actionTabs = computed(() => [
  { value: 'all', label: t('admin.logs.filterAll') },
  { value: 'hide', label: t('admin.reviews.hide') },
  { value: 'restore', label: t('admin.reviews.restore') },
  { value: 'delete', label: t('admin.reviews.deleted') },
  { value: 'edit', label: 'Edit' }
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const filteredLogs = computed(() => {
  if (actionFilter.value === 'all') return logs.value
  return logs.value.filter(l => l.action === actionFilter.value)
})

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

function actionBadgeClass(action: string) {
  const map: Record<string, string> = {
    hide: 'bg-warning/10 text-warning',
    restore: 'bg-success/10 text-success',
    delete: 'bg-danger/10 text-danger',
    edit: 'bg-primary/10 text-primary'
  }
  return map[action] || 'bg-bg-secondary text-text-secondary'
}

function resourceIcon(type: string): Component {
  const map: Record<string, Component> = {
    review: MessageSquare,
    report: Flag
  }
  return map[type] || FileText
}

const formatRelative = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)
const formatAbsolute = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await getOperationLogs(page.value, pageSize)
    logs.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch {
    toast.error(t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

watch(page, () => fetchLogs())
onMounted(fetchLogs)
</script>
