<template>
  <div class="space-y-4">
    <!-- Batch action bar -->
    <div
      v-if="selected.length > 0"
      class="flex items-center gap-3 p-3 bg-primary/5 border border-primary/20 rounded-xl animate-fade-in-down"
    >
      <span class="text-sm font-medium text-primary">{{ t('admin.reviews.selectedCount', { count: selected.length }) }}</span>
      <div class="flex items-center gap-2 ml-auto">
        <button
          class="px-3 py-1.5 text-xs font-medium text-warning border border-warning/30 rounded-lg transition-colors duration-fast hover:bg-warning/10"
          @click="handleBatch('hide')"
        >{{ t('admin.reviews.batchHide') }}</button>
        <button
          class="px-3 py-1.5 text-xs font-medium text-success border border-success/30 rounded-lg transition-colors duration-fast hover:bg-success/10"
          @click="handleBatch('restore')"
        >{{ t('admin.reviews.batchRestore') }}</button>
        <button
          class="px-3 py-1.5 text-xs font-medium text-danger border border-danger/30 rounded-lg transition-colors duration-fast hover:bg-danger/10"
          @click="handleBatch('delete')"
        >{{ t('admin.reviews.batchDelete') }}</button>
      </div>
    </div>

    <!-- Toolbar -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary">{{ t('admin.reviews.title') }}</h1>
        <TabBar v-model="status" :tabs="statusTabs" />
      </div>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <div class="relative flex-1 sm:flex-initial">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-text-muted pointer-events-none" />
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="t('admin.reviews.searchPlaceholder')"
            class="w-full sm:w-56 pl-9 pr-3 py-2 text-sm bg-bg-card border border-border rounded-lg text-text-primary placeholder:text-text-muted transition-colors duration-fast focus:border-primary focus:outline-none"
          />
        </div>
        <button
          class="flex items-center gap-1.5 px-3 py-2 text-sm text-text-secondary border border-border rounded-lg transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
          @click="handleExport"
        >
          <Download class="size-3.5" />
          <span class="hidden sm:inline">{{ t('admin.reviews.export') }}</span>
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div v-for="i in 5" :key="i" class="h-16 border-b border-border animate-pulse" />
    </div>

    <!-- Table -->
    <div v-else-if="filteredReviews.length > 0" class="bg-bg-card border border-border rounded-xl shadow-card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <tr class="bg-bg-secondary">
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted w-10">
                <input
                  type="checkbox"
                  :checked="isAllSelected"
                  :indeterminate="isIndeterminate"
                  :aria-label="t('admin.reviews.selectAll')"
                  @change="toggleAll"
                />
              </th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.reviews.tableContent') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.reviews.tableCourse') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden lg:table-cell">{{ t('admin.reviews.tableTeacher') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden sm:table-cell">{{ t('admin.reviews.tableRating') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.reviews.tableStatus') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted hidden md:table-cell">{{ t('admin.reviews.tableTime') }}</th>
              <th class="sticky top-0 p-3 text-left text-xs font-medium text-text-muted">{{ t('admin.reviews.tableActions') }}</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="r in filteredReviews" :key="r.id">
              <tr
                class="border-t border-border transition-colors duration-fast hover:bg-bg-hover cursor-pointer"
                @click="toggleExpand(r.id)"
              >
                <td class="p-3" @click.stop><input type="checkbox" v-model="selected" :value="r.id" /></td>
                <td class="p-3 text-sm text-text-primary max-w-[280px]">
                  <div class="flex items-center gap-1.5">
                    <ChevronRight class="size-3.5 text-text-muted shrink-0 transition-transform duration-fast" :class="expandedIds.has(r.id) && 'rotate-90'" />
                    <span :title="r.content">{{ r.content.length > 80 ? r.content.slice(0, 80) + '...' : r.content }}</span>
                  </div>
                </td>
                <td class="p-3 text-sm text-text-secondary hidden md:table-cell whitespace-nowrap">{{ r.courseName || '-' }}</td>
                <td class="p-3 text-sm text-text-secondary hidden lg:table-cell whitespace-nowrap">{{ r.teacherName || '-' }}</td>
                <td class="p-3 hidden sm:table-cell">
                  <span
                    v-if="avgRating(r)"
                    class="inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-md"
                    :class="ratingClass(avgRating(r)!)"
                  >
                    {{ avgRating(r)!.toFixed(1) }}
                  </span>
                  <span v-else class="text-xs text-text-muted">-</span>
                </td>
                <td class="p-3">
                  <span
                    class="inline-flex px-2 py-0.5 text-xs font-medium rounded-md"
                    :class="statusBadgeClass(r.status)"
                  >
                    {{ statusLabel(r.status) }}
                  </span>
                </td>
                <td class="p-3 text-xs text-text-muted hidden md:table-cell whitespace-nowrap" :title="formatAbsolute(r.createdAt)">
                  {{ formatRelative(r.createdAt) }}
                </td>
                <td class="p-3" @click.stop>
                  <div class="flex items-center gap-1">
                    <button
                      v-if="r.status === 'published'"
                      class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-warning/10 hover:text-warning"
                      :title="t('admin.reviews.hide')"
                      @click="handleUpdate(r.id, 'hide')"
                    >
                      <EyeOff class="size-4" />
                    </button>
                    <button
                      v-else
                      class="p-1.5 text-text-muted rounded-md transition-colors duration-fast hover:bg-success/10 hover:text-success"
                      :title="t('admin.reviews.restore')"
                      @click="handleUpdate(r.id, 'restore')"
                    >
                      <Eye class="size-4" />
                    </button>
                  </div>
                </td>
              </tr>
              <!-- Expandable detail row -->
              <tr v-if="expandedIds.has(r.id)" class="border-t border-border bg-bg-secondary/50">
                <td :colspan="8" class="p-4">
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                    <div>
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailTitle') }}</span>
                      <p class="mt-0.5 text-text-primary">{{ r.title || t('admin.reviews.detailNoTitle') }}</p>
                    </div>
                    <div>
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailTerm') }}</span>
                      <p class="mt-0.5 text-text-primary">{{ r.termName || r.termID || t('admin.reviews.detailNoTerm') }}</p>
                    </div>
                    <div>
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailGrade') }}</span>
                      <p class="mt-0.5 text-text-primary">{{ r.grade || t('admin.reviews.detailNoGrade') }}</p>
                    </div>
                    <div>
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailInteraction') }}</span>
                      <div class="mt-0.5 flex items-center gap-3 text-text-primary">
                        <span>{{ t('admin.reviews.detailLikes') }}: {{ r.likeCount }}</span>
                        <span>{{ t('admin.reviews.detailDislikes') }}: {{ r.dislikeCount }}</span>
                        <span>{{ t('admin.reviews.detailReplies') }}: {{ r.replyCount }}</span>
                      </div>
                    </div>
                    <div v-if="Object.keys(r.ratings || {}).length > 0" class="md:col-span-2">
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailRatings') }}</span>
                      <div class="mt-1 flex flex-wrap gap-2">
                        <span
                          v-for="(val, key) in r.ratings"
                          :key="key"
                          class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-md bg-bg-secondary border border-border"
                        >
                          <span class="text-text-muted">{{ t(`review.ratingEmoji.${key}`, key as string) }}</span>
                          <span class="font-medium" :class="ratingClass(val)">{{ val }}</span>
                        </span>
                      </div>
                    </div>
                    <div class="md:col-span-2">
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailContent') }}</span>
                      <p class="mt-1 text-text-primary whitespace-pre-wrap break-words bg-bg-card border border-border rounded-lg p-3 max-h-[200px] overflow-y-auto">{{ r.content }}</p>
                    </div>
                    <div v-if="r.moderationReason" class="md:col-span-2">
                      <span class="text-text-muted text-xs font-medium">{{ t('admin.reviews.detailModeration') }}</span>
                      <p class="mt-0.5 text-warning">{{ r.moderationReason }}</p>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </div>

    <EmptyState v-else :title="t('admin.reviews.empty')" />

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
      <select
        :value="pageSize"
        class="px-2 py-1.5 bg-bg-card border border-border rounded-lg text-sm text-text-primary"
        @change="onPageSizeChange"
      >
        <option v-for="s in [10, 20, 50]" :key="s" :value="s">{{ t('admin.pagination.pageSize', { size: s }) }}</option>
      </select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Search, Download, EyeOff, Eye, ChevronRight } from 'lucide-vue-next'
import { getAllReviews, updateReviewStatus, batchUpdateReviews, exportData } from '@/api/admin'
import type { BatchUpdateParams } from '@/types/admin'
import type { Review } from '@/types/review'
import TabBar from '@/components/common/TabBar.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime, formatAbsoluteTime } from '@/utils/date'

const { t, locale } = useI18n()
const toast = useToast()

const status = ref('all')
const loading = ref(true)
const reviews = ref<Review[]>([])
const selected = ref<string[]>([])
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const searchQuery = ref('')
const expandedIds = ref(new Set<string>())

function toggleExpand(id: string) {
  const s = new Set(expandedIds.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  expandedIds.value = s
}

const statusTabs = computed(() => [
  { value: 'all', label: t('admin.reviews.allStatus') },
  { value: 'published', label: t('admin.reviews.published') },
  { value: 'hidden', label: t('admin.reviews.hidden') }
])

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

const filteredReviews = computed(() => {
  if (!searchQuery.value.trim()) return reviews.value
  const q = searchQuery.value.toLowerCase()
  return reviews.value.filter(r =>
    r.content.toLowerCase().includes(q) ||
    (r.courseName && r.courseName.toLowerCase().includes(q)) ||
    (r.teacherName && r.teacherName.toLowerCase().includes(q))
  )
})

const isAllSelected = computed(() =>
  filteredReviews.value.length > 0 && selected.value.length === filteredReviews.value.length
)
const isIndeterminate = computed(() =>
  selected.value.length > 0 && selected.value.length < filteredReviews.value.length
)

// Pagination: show max 5 pages + ellipsis
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

function avgRating(r: Review): number | null {
  const vals = Object.values(r.ratings || {})
  if (vals.length === 0) return null
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

function ratingClass(rating: number) {
  if (rating >= 4) return 'bg-success/10 text-success'
  if (rating >= 3) return 'bg-warning/10 text-warning'
  return 'bg-danger/10 text-danger'
}

function statusBadgeClass(s: string) {
  if (s === 'published') return 'bg-success/10 text-success'
  if (s === 'hidden') return 'bg-warning/10 text-warning'
  return 'bg-bg-secondary text-text-muted'
}

function statusLabel(s: string) {
  if (s === 'published') return t('admin.reviews.published')
  if (s === 'hidden') return t('admin.reviews.hidden')
  return s
}

const formatRelative = (dateStr: string) => formatRelativeTime(dateStr, locale.value, t)
const formatAbsolute = (dateStr: string) => formatAbsoluteTime(dateStr, locale.value)

const fetchReviews = async () => {
  loading.value = true
  try {
    const res = await getAllReviews(status.value, page.value, pageSize.value)
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

async function handleExport() {
  try {
    const res = await exportData('ndjson', status.value)
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `reviews-export-${new Date().toISOString().slice(0, 10)}.ndjson`
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.error(t('common.loadFailed'))
  }
}

const toggleAll = (e: Event) => {
  const checked = (e.target as HTMLInputElement).checked
  selected.value = checked ? filteredReviews.value.map(r => r.id) : []
}

const onPageSizeChange = (e: Event) => {
  pageSize.value = Number((e.target as HTMLSelectElement).value)
  page.value = 1
}

watch(status, () => {
  selected.value = []
  page.value = 1
  fetchReviews()
})
watch(page, () => {
  selected.value = []
  fetchReviews()
})
watch(pageSize, () => fetchReviews())
onMounted(fetchReviews)
</script>
