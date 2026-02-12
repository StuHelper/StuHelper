<template>
  <div>
    <h1 class="mb-4 font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.reviews.title') }}</h1>

    <div class="flex items-center gap-4 mb-4">
      <select
        v-model="status"
        class="p-2 bg-transparent border border-border rounded-sm text-text-primary text-sm"
      >
        <option value="all">{{ t('admin.reviews.allStatus') }}</option>
        <option value="published">{{ t('admin.reviews.published') }}</option>
        <option value="hidden">{{ t('admin.reviews.hidden') }}</option>
      </select>
      <div v-if="selected.length > 0" class="flex items-center gap-2 text-sm">
        <span>{{ t('admin.reviews.selectedCount', { count: selected.length }) }}</span>
        <button
          class="px-2 py-1 bg-transparent border border-border rounded-sm text-sm cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
          @click="handleBatch('hide')"
        >{{ t('admin.reviews.batchHide') }}</button>
        <button
          class="px-2 py-1 bg-transparent border border-border rounded-sm text-sm cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
          @click="handleBatch('restore')"
        >{{ t('admin.reviews.batchRestore') }}</button>
      </div>
    </div>

    <div v-if="loading" class="text-center p-8 text-text-muted">{{ t('common.actions.loading') }}</div>

    <table v-else-if="reviews.length > 0" class="w-full border-collapse">
      <thead>
        <tr>
          <th class="p-3 text-left border-b border-border text-sm"><input type="checkbox" @change="toggleAll" /></th>
          <th class="p-3 text-left border-b border-border text-sm">{{ t('admin.reviews.tableContent') }}</th>
          <th class="p-3 text-left border-b border-border text-sm">{{ t('admin.reviews.tableStatus') }}</th>
          <th class="p-3 text-left border-b border-border text-sm">{{ t('admin.reviews.tableActions') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in reviews" :key="r.id">
          <td class="p-3 text-left border-b border-border text-sm"><input type="checkbox" v-model="selected" :value="r.id" /></td>
          <td class="p-3 text-left border-b border-border text-sm max-w-[400px] overflow-hidden text-ellipsis whitespace-nowrap">{{ r.content.slice(0, 50) }}...</td>
          <td class="p-3 text-left border-b border-border text-sm">
            <span
              class="px-2 py-1 rounded-sm text-xs"
              :class="r.status === 'published' ? 'text-text-secondary' : 'text-text-muted'"
            >
              {{ r.status === 'published' ? t('admin.reviews.published') : t('admin.reviews.hidden') }}
            </span>
          </td>
          <td class="p-3 text-left border-b border-border text-sm">
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

    <div v-if="totalPages > 1" class="flex items-center justify-center gap-3 mt-4 text-sm text-text-secondary">
      <button
        class="px-3 py-1 bg-transparent border border-border rounded-sm text-sm cursor-pointer transition-all duration-fast hover:not-disabled:border-text-primary hover:not-disabled:text-text-primary disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="page <= 1"
        @click="page--"
      >{{ t('admin.pagination.prev') }}</button>
      <span>{{ page }} / {{ totalPages }}</span>
      <button
        class="px-3 py-1 bg-transparent border border-border rounded-sm text-sm cursor-pointer transition-all duration-fast hover:not-disabled:border-text-primary hover:not-disabled:text-text-primary disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="page >= totalPages"
        @click="page++"
      >{{ t('admin.pagination.next') }}</button>
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
const selected = ref<string[]>([])
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

const handleUpdate = async (id: string, action: string) => {
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
