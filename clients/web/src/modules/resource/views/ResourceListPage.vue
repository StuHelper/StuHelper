<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { Download, FileText, RefreshCw, Search, Tag, X } from 'lucide-vue-next'
import { api } from '@/api'
import {
  readResourcePagePayload,
  type ResourceItem,
} from '@/modules/resource/resourcePayload'

const { t, locale } = useI18n()

const query = ref('')
const tag = ref('')
const bindingType = ref('')
const bindingValue = ref('')
const resources = ref<ResourceItem[]>([])
const total = ref(0)
const loading = ref(false)
const errorMessage = ref('')
const hasSearched = ref(false)

const hasFilters = computed(() =>
  Boolean(
    query.value.trim() ||
      tag.value.trim() ||
      bindingType.value.trim() ||
      bindingValue.value.trim(),
  ),
)

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

function formatFileSize(bytes: number) {
  if (!Number.isFinite(bytes) || bytes < 0) {
    return ''
  }
  if (bytes < 1024) return `${bytes} B`
  const units = ['KB', 'MB', 'GB']
  let size = bytes / 1024
  let unitIndex = 0
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024
    unitIndex += 1
  }
  return `${size.toFixed(size >= 10 ? 0 : 1)} ${units[unitIndex]}`
}

function resourceMeta(resource: ResourceItem) {
  return [
    resource.category,
    resource.latestVersion.filename,
    formatFileSize(resource.latestVersion.sizeBytes),
  ].filter(Boolean).join(' · ')
}

async function loadResources() {
  loading.value = true
  errorMessage.value = ''
  hasSearched.value = true
  try {
    const res = await api.resource.listResources({
      page: 1,
      pageSize: 24,
      query: query.value.trim() || undefined,
      tag: tag.value.trim() || undefined,
      bindingType: bindingType.value.trim() || undefined,
      bindingValue: bindingValue.value.trim() || undefined,
    })
    const data = readResourcePagePayload(
      res.data?.data,
      'Invalid resource list response',
    )
    resources.value = data.items
    total.value = data.total
  } catch (_error) {
    void _error
    resources.value = []
    total.value = 0
    errorMessage.value = t('resource.list.loadFailed')
  } finally {
    loading.value = false
  }
}

function clearFilters() {
  query.value = ''
  tag.value = ''
  bindingType.value = ''
  bindingValue.value = ''
  void loadResources()
}

onMounted(() => {
  void loadResources()
})
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <section class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header class="mb-6 flex flex-col gap-2">
        <h1 class="text-3xl font-extrabold tracking-tight text-text-primary">
          {{ t('resource.list.title') }}
        </h1>
        <p class="text-sm text-text-secondary">
          {{ t('resource.list.subtitle') }}
        </p>
      </header>

      <form
        class="mb-6 grid gap-3 rounded-lg border border-border-light bg-bg-card p-4 shadow-xs md:grid-cols-[minmax(0,2fr)_minmax(140px,1fr)_minmax(140px,1fr)_minmax(140px,1fr)_auto]"
        @submit.prevent="loadResources"
      >
        <label class="grid gap-1 text-xs font-semibold text-text-secondary">
          {{ t('resource.list.searchLabel') }}
          <span class="relative">
            <Search
              class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
              :size="16"
            />
            <input
              v-model="query"
              type="search"
              autocomplete="off"
              data-1p-ignore
              data-lpignore="true"
              data-form-type="other"
              class="h-11 w-full rounded-lg border border-border-light bg-bg-base pl-9 pr-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :placeholder="t('resource.list.searchPlaceholder')"
            />
          </span>
        </label>

        <label class="grid gap-1 text-xs font-semibold text-text-secondary">
          {{ t('resource.list.tagLabel') }}
          <input
            v-model="tag"
            type="text"
            autocomplete="off"
            class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
            :placeholder="t('resource.list.tagPlaceholder')"
          />
        </label>

        <label class="grid gap-1 text-xs font-semibold text-text-secondary">
          {{ t('resource.list.bindingTypeLabel') }}
          <input
            v-model="bindingType"
            type="text"
            autocomplete="off"
            class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
            :placeholder="t('resource.list.bindingTypePlaceholder')"
          />
        </label>

        <label class="grid gap-1 text-xs font-semibold text-text-secondary">
          {{ t('resource.list.bindingValueLabel') }}
          <input
            v-model="bindingValue"
            type="text"
            autocomplete="off"
            class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
            :placeholder="t('resource.list.bindingValuePlaceholder')"
          />
        </label>

        <div class="flex items-end gap-2">
          <button
            type="submit"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white transition-colors hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
            :disabled="loading"
          >
            <Search :size="16" />
            <span>{{ t('resource.list.searchButton') }}</span>
          </button>
          <button
            v-if="hasFilters"
            type="button"
            class="inline-flex h-11 w-11 items-center justify-center rounded-lg border border-border-light text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary"
            :aria-label="t('resource.list.clearButton')"
            :title="t('resource.list.clearButton')"
            @click="clearFilters"
          >
            <X :size="16" />
          </button>
        </div>
      </form>

      <div class="mb-4 flex min-h-6 items-center justify-between text-sm text-text-muted">
        <span v-if="!errorMessage && !loading">
          {{ t('resource.list.total', { count: total }) }}
        </span>
      </div>

      <div
        v-if="loading"
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <div
          v-for="i in 6"
          :key="i"
          class="h-44 rounded-lg bg-bg-card shadow-xs animate-pulse"
        />
      </div>

      <div
        v-else-if="errorMessage"
        role="alert"
        class="flex min-h-[260px] flex-col items-center justify-center gap-4 rounded-lg border border-border-light bg-bg-card p-8 text-center"
      >
        <FileText :size="40" class="text-text-muted" />
        <p class="text-sm font-medium text-danger">{{ errorMessage }}</p>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary/90"
          @click="loadResources"
        >
          <RefreshCw :size="16" />
          {{ t('resource.list.retry') }}
        </button>
      </div>

      <div
        v-else-if="hasSearched && resources.length === 0"
        class="flex min-h-[260px] flex-col items-center justify-center gap-3 rounded-lg border border-border-light bg-bg-card p-8 text-center"
      >
        <FileText :size="40" class="text-text-muted" />
        <h2 class="text-lg font-bold text-text-primary">
          {{ t('resource.list.emptyTitle') }}
        </h2>
        <p class="text-sm text-text-secondary">
          {{ t('resource.list.emptyDesc') }}
        </p>
      </div>

      <div
        v-else
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <RouterLink
          v-for="resource in resources"
          :key="resource.id"
          :to="{ name: 'resource-detail', params: { id: resource.id } }"
          class="group flex min-h-44 flex-col rounded-lg border border-border-light bg-bg-card p-5 no-underline shadow-xs transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
        >
          <div class="mb-3 flex items-start justify-between gap-3">
            <div class="min-w-0">
              <h2 class="truncate text-base font-bold text-text-primary group-hover:text-primary">
                {{ resource.title }}
              </h2>
              <p class="mt-1 truncate text-xs text-text-muted">
                {{ resourceMeta(resource) }}
              </p>
            </div>
            <Download :size="18" class="shrink-0 text-text-muted group-hover:text-primary" />
          </div>

          <p class="line-clamp-3 min-h-[3.75rem] text-sm leading-5 text-text-secondary">
            {{ resource.description || t('resource.detail.noDescription') }}
          </p>

          <div class="mt-4 flex flex-wrap gap-2">
            <span
              v-for="item in resource.tags"
              :key="item"
              class="inline-flex items-center gap-1 rounded-full bg-primary-alpha px-2.5 py-1 text-xs font-medium text-primary"
            >
              <Tag :size="12" />
              {{ item }}
            </span>
          </div>

          <div class="mt-auto pt-4 text-xs text-text-muted">
            {{ formatDate(resource.updatedAt) }}
          </div>
        </RouterLink>
      </div>
    </section>
  </div>
</template>
