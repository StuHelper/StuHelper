<template>
  <div class="reviews-manage">
    <h1>{{ t('admin.reviews.title') }}</h1>

    <div class="toolbar">
      <select v-model="status" class="status-select">
        <option value="all">{{ t('admin.reviews.allStatus') }}</option>
        <option value="published">{{ t('admin.reviews.published') }}</option>
        <option value="hidden">{{ t('admin.reviews.hidden') }}</option>
      </select>
      <div v-if="selected.length > 0" class="batch-actions">
        <span>{{ t('admin.reviews.selectedCount', { count: selected.length }) }}</span>
        <button @click="handleBatch('hide')">{{ t('admin.reviews.batchHide') }}</button>
        <button @click="handleBatch('restore')">{{ t('admin.reviews.batchRestore') }}</button>
      </div>
    </div>

    <div v-if="loading" class="loading">{{ t('common.actions.loading') }}</div>

    <table v-else-if="reviews.length > 0" class="reviews-table">
      <thead>
        <tr>
          <th><input type="checkbox" @change="toggleAll" /></th>
          <th>{{ t('admin.reviews.tableContent') }}</th>
          <th>{{ t('admin.reviews.tableStatus') }}</th>
          <th>{{ t('admin.reviews.tableActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in reviews" :key="r.id">
          <td><input type="checkbox" v-model="selected" :value="r.id" /></td>
          <td class="content-cell">{{ r.content.slice(0, 50) }}...</td>
          <td>
            <span class="status" :class="r.status">
              {{ r.status === 'published' ? t('admin.reviews.published') : t('admin.reviews.hidden') }}
            </span>
          </td>
          <td>
            <button
              v-if="r.status === 'published'"
              @click="handleUpdate(r.id, 'hide')"
            >{{ t('admin.reviews.hide') }}</button>
            <button v-else @click="handleUpdate(r.id, 'restore')">{{ t('admin.reviews.restore') }}</button>
          </td>
        </tr>
      </tbody>
    </table>

    <EmptyState v-else :title="t('admin.reviews.empty')" />

    <div v-if="totalPages > 1" class="pagination">
      <button :disabled="page <= 1" @click="page--">{{ t('admin.pagination.prev') }}</button>
      <span>{{ page }} / {{ totalPages }}</span>
      <button :disabled="page >= totalPages" @click="page++">{{ t('admin.pagination.next') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getAllReviews, updateReviewStatus, batchUpdateReviews } from '@/api/admin'
import type { BatchUpdateParams } from '@/types/admin'
import type { Review } from '@/types/review'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'

const { t } = useI18n()
const toast = useToast()

const status = ref('all')
const loading = ref(true)
const reviews = ref<Review[]>([])
const selected = ref<number[]>([])
const page = ref(1)
const total = ref(0)
const pageSize = 20

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const fetchReviews = async () => {
  loading.value = true
  try {
    const res = await getAllReviews(status.value, page.value, pageSize)
    reviews.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const handleUpdate = async (id: number, action: string) => {
  try {
    await updateReviewStatus(id, action)
    toast.success(t('admin.reviews.updateSuccess'))
    fetchReviews()
  } catch {
    toast.error(t('admin.reviews.updateFailed'))
  }
}

const handleBatch = async (action: BatchUpdateParams['action']) => {
  try {
    await batchUpdateReviews({ ids: selected.value, action })
    toast.success(t('admin.reviews.batchSuccess'))
    selected.value = []
    fetchReviews()
  } catch {
    toast.error(t('admin.reviews.batchFailed'))
  }
}

const toggleAll = (e: Event) => {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? reviews.value.map(r => r.id) : []
}

watch(status, () => {
  selected.value = []
  page.value = 1
  fetchReviews()
})
watch(page, fetchReviews)
onMounted(fetchReviews)
</script>

<style scoped>
.reviews-manage h1 {
  margin: 0 0 var(--space-4);
  font-family: var(--font-sans);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
  color: var(--text-primary);
}

.toolbar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.status-select {
  padding: var(--space-2);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  font-size: var(--text-sm);
}

.batch-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.batch-actions button {
  padding: var(--space-1) var(--space-2);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.batch-actions button:hover {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

.reviews-table {
  width: 100%;
  border-collapse: collapse;
}

.reviews-table th,
.reviews-table td {
  padding: var(--space-3);
  text-align: left;
  border-bottom: 1px solid var(--border);
  font-size: var(--text-sm);
}

.content-cell {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status {
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}

.status.published {
  background: transparent;
  color: var(--text-secondary);
}

.status.hidden {
  background: transparent;
  color: var(--text-muted);
}

.loading {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-muted);
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.pagination button {
  padding: var(--space-1) var(--space-3);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: all var(--duration-fast);
}

.pagination button:hover:not(:disabled) {
  border-color: var(--text-primary);
  color: var(--text-primary);
}

.pagination button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
