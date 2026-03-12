<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronLeftIcon, ChevronRightIcon } from '@heroicons/vue/24/outline'

const { t } = useI18n()

interface Props {
  currentPage: number
  totalPages: number
  pageSize: number
  totalCount: number
  pageSizeOptions?: number[]
}

const props = withDefaults(defineProps<Props>(), {
  pageSizeOptions: () => [10, 20, 50]
})

const emit = defineEmits<{
  'update:currentPage': [page: number]
  'update:pageSize': [size: number]
}>()

const visiblePages = computed(() => {
  const result: (number | string)[] = []
  const { currentPage, totalPages } = props

  if (totalPages <= 7) {
    for (let i = 1; i <= totalPages; i++) result.push(i)
    return result
  }

  result.push(1)
  if (currentPage > 3) result.push('...')

  const start = Math.max(2, currentPage - 1)
  const end = Math.min(totalPages - 1, currentPage + 1)
  for (let i = start; i <= end; i++) result.push(i)

  if (currentPage < totalPages - 2) result.push('...')
  result.push(totalPages)

  return result
})

function goToPage(page: number) {
  if (page >= 1 && page <= props.totalPages && page !== props.currentPage) {
    emit('update:currentPage', page)
  }
}

function changePageSize(size: number) {
  emit('update:pageSize', size)
  emit('update:currentPage', 1)
}
</script>

<template>
  <div class="flex items-center justify-between px-4 py-3 border-t border-gray-200 dark:border-gray-700">
    <!-- Total count and page size selector -->
    <div class="flex items-center gap-4 text-sm text-gray-600 dark:text-gray-400">
      <span>{{ t('common.pagination.total', { total: totalCount }) }}</span>
      <div class="flex items-center gap-2">
        <span>{{ t('common.pagination.perPage') }}</span>
        <select
          :value="pageSize"
          @change="changePageSize(Number(($event.target as HTMLSelectElement).value))"
          class="px-2 py-1 rounded border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-primary/50"
          :aria-label="t('common.pagination.pageSizeLabel')"
        >
          <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
        </select>
        <span>{{ t('common.pagination.items') }}</span>
      </div>
    </div>

    <!-- Page navigation -->
    <nav class="flex items-center gap-1" role="navigation" :aria-label="t('common.pagination.nav')">
      <button
        @click="goToPage(currentPage - 1)"
        :disabled="currentPage === 1"
        class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        :aria-label="t('common.pagination.prevPage')"
        type="button"
      >
        <ChevronLeftIcon class="w-5 h-5" />
      </button>

      <button
        v-for="(page, index) in visiblePages"
        :key="index"
        @click="typeof page === 'number' && goToPage(page)"
        :disabled="typeof page !== 'number'"
        :class="[
          'min-w-[2.5rem] px-3 py-1 rounded text-sm transition-colors',
          page === currentPage
            ? 'bg-primary text-white'
            : typeof page === 'number'
            ? 'hover:bg-gray-100 dark:hover:bg-gray-700'
            : 'cursor-default'
        ]"
        :aria-label="typeof page === 'number' ? t('common.pagination.pageLabel', { page }) : undefined"
        :aria-current="page === currentPage ? 'page' : undefined"
        type="button"
      >
        {{ page }}
      </button>

      <button
        @click="goToPage(currentPage + 1)"
        :disabled="currentPage === totalPages"
        class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        :aria-label="t('common.pagination.nextPage')"
        type="button"
      >
        <ChevronRightIcon class="w-5 h-5" />
      </button>
    </nav>
  </div>
</template>
