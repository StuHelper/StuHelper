<template>
  <div>
    <h1 class="mb-4 font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.logs.title') }}</h1>

    <div v-if="loading" class="text-center p-8 text-text-muted">{{ t('common.actions.loading') }}</div>

    <table v-else-if="logs.length > 0" class="w-full border-collapse">
      <thead>
        <tr>
          <th class="p-3 text-left border-b border-border text-sm text-text-muted font-medium">{{ t('admin.logs.operator') }}</th>
          <th class="p-3 text-left border-b border-border text-sm text-text-muted font-medium">{{ t('admin.logs.action') }}</th>
          <th class="p-3 text-left border-b border-border text-sm text-text-muted font-medium">{{ t('admin.logs.resource') }}</th>
          <th class="p-3 text-left border-b border-border text-sm text-text-muted font-medium">{{ t('admin.logs.time') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="log in logs" :key="log.id">
          <td class="p-3 text-left border-b border-border text-sm">{{ log.adminUsername }}</td>
          <td class="p-3 text-left border-b border-border text-sm">{{ log.action }}</td>
          <td class="p-3 text-left border-b border-border text-sm">{{ log.resourceType }} #{{ log.resourceID }}</td>
          <td class="p-3 text-left border-b border-border text-sm">{{ formatTime(log.createdAt) }}</td>
        </tr>
      </tbody>
    </table>

    <EmptyState v-else :title="t('admin.logs.empty')" />

    <div v-if="hasMore" class="flex justify-center p-4">
      <button
        class="px-4 py-2 bg-transparent border border-border rounded-sm text-text-secondary text-sm cursor-pointer transition-all duration-fast hover:not-disabled:border-text-primary hover:not-disabled:text-text-primary"
        @click="loadMore"
        :disabled="loadingMore"
      >
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
