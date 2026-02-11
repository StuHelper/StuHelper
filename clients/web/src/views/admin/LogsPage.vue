<template>
  <div class="logs-page">
    <h1>{{ t('admin.logs.title') }}</h1>

    <div v-if="loading" class="loading">{{ t('common.actions.loading') }}</div>

    <table v-else-if="logs.length > 0" class="logs-table">
      <thead>
        <tr>
          <th>{{ t('admin.logs.operator') }}</th>
          <th>{{ t('admin.logs.action') }}</th>
          <th>{{ t('admin.logs.resource') }}</th>
          <th>{{ t('admin.logs.time') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="log in logs" :key="log.id">
          <td>{{ log.adminUsername }}</td>
          <td>{{ log.action }}</td>
          <td>{{ log.resourceType }} #{{ log.resourceID }}</td>
          <td>{{ formatTime(log.createdAt) }}</td>
        </tr>
      </tbody>
    </table>

    <EmptyState v-else :title="t('admin.logs.empty')" />

    <div v-if="hasMore" class="load-more">
      <button @click="loadMore" :disabled="loadingMore">
        {{ loadingMore ? t('common.actions.loading') : t('admin.logs.loadMore') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getOperationLogs, type OperationLog } from '@/api/admin'
import EmptyState from '@/components/common/EmptyState.vue'
import { formatAbsoluteTime } from '@/utils/date'

const { t, locale } = useI18n()

const loading = ref(true)
const loadingMore = ref(false)
const logs = ref<OperationLog[]>([])
const total = ref(0)
const page = ref(1)

const hasMore = computed(() => logs.value.length < total.value)

const fetchLogs = async (p: number) => {
  const res = await getOperationLogs(p)
  return res.data
}

onMounted(async () => {
  try {
    const data = await fetchLogs(1)
    logs.value = data?.list || []
    total.value = data?.total || 0
  } finally {
    loading.value = false
  }
})

const loadMore = async () => {
  loadingMore.value = true
  try {
    const data = await fetchLogs(page.value + 1)
    logs.value.push(...(data?.list || []))
    page.value++
  } finally {
    loadingMore.value = false
  }
}

const formatTime = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)
</script>

<style scoped>
.logs-page h1 {
  margin: 0 0 var(--space-4);
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}

.logs-table {
  width: 100%;
  border-collapse: collapse;
}

.logs-table th,
.logs-table td {
  padding: var(--space-3);
  text-align: left;
  border-bottom: 1px solid var(--border);
  font-size: var(--text-sm);
}

.logs-table th {
  color: var(--text-muted);
  font-weight: var(--weight-medium);
}

.loading {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-muted);
}

.load-more {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.load-more button {
  padding: var(--space-2) var(--space-4);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.load-more button:hover:not(:disabled) {
  border-color: var(--text-primary);
  color: var(--text-primary);
}
</style>
