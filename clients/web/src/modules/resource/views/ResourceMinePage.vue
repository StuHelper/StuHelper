<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter, type LocationQueryRaw, type LocationQueryValue } from 'vue-router'
import { Eye, EyeOff, FileText, Pencil, PlusCircle, RefreshCw, Search, Tag, Trash2, X } from 'lucide-vue-next'
import { api } from '@/api'
import {
  readResourcePagePayload,
  type ResourceItem,
} from '@/modules/resource/resourcePayload'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

type ResourceMineVisibility = '' | 'public' | 'private'

const query = ref('')
const visibility = ref<ResourceMineVisibility>('')
const resources = ref<ResourceItem[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const errorMessage = ref('')
const loadMoreError = ref('')
const deleteError = ref('')
const deletingIDs = ref<Set<string>>(new Set())
const hasSearched = ref(false)
const PAGE_SIZE = 24
const RESOURCE_MINE_FILTER_KEYS = ['query', 'visibility'] as const
let resourceRequestSeq = 0

interface ResourceMineFilters {
  query: string
  visibility: ResourceMineVisibility
}

const hasFilters = computed(() =>
  Boolean(query.value.trim() || visibility.value),
)
const hasMore = computed(() => resources.value.length < total.value)

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

async function loadResourcePage(nextPage: number, append = false) {
  const requestSeq = ++resourceRequestSeq
  const filters = {
    query: query.value.trim() || undefined,
    visibility: visibility.value || undefined,
  }

  if (append) {
    loadingMore.value = true
    loadMoreError.value = ''
  } else {
    loading.value = true
    errorMessage.value = ''
    loadMoreError.value = ''
    deleteError.value = ''
    hasSearched.value = true
  }
  try {
    const res = await api.resource.listMyResources({
      page: nextPage,
      pageSize: PAGE_SIZE,
      ...filters,
    })
    const data = readResourcePagePayload(
      res.data?.data,
      'Invalid resource list response',
    )
    if (requestSeq !== resourceRequestSeq) return
    resources.value = append ? [...resources.value, ...data.items] : data.items
    total.value = data.total
    page.value = data.page
  } catch (_error) {
    void _error
    if (requestSeq !== resourceRequestSeq) return
    if (append) {
      loadMoreError.value = t('resource.mine.loadMoreFailed')
    } else {
      resources.value = []
      total.value = 0
      page.value = 1
      errorMessage.value = t('resource.mine.loadFailed')
    }
  } finally {
    if (requestSeq === resourceRequestSeq) {
      if (append) {
        loadingMore.value = false
      } else {
        loading.value = false
      }
    }
  }
}

function loadResources() {
  return loadResourcePage(1)
}

function inputResourceFilters(): ResourceMineFilters {
  return {
    query: query.value.trim(),
    visibility: visibility.value,
  }
}

function routeResourceFilters(): ResourceMineFilters {
  return {
    query: readRouteFilter('query'),
    visibility: readRouteVisibility(),
  }
}

function readRouteFilter(key: keyof Pick<ResourceMineFilters, 'query'>): string {
  return firstRouteQueryValue(route.query[key])?.trim() ?? ''
}

function readRouteVisibility(): ResourceMineVisibility {
  const value = firstRouteQueryValue(route.query.visibility)?.trim()
  return value === 'public' || value === 'private' ? value : ''
}

function firstRouteQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? null
  }
  return typeof value === 'string' ? value : null
}

function applyResourceFilters(filters: ResourceMineFilters): void {
  query.value = filters.query
  visibility.value = filters.visibility
}

function resourceFiltersToQuery(filters: ResourceMineFilters): LocationQueryRaw {
  const nextQuery: LocationQueryRaw = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (RESOURCE_MINE_FILTER_KEYS.includes(key as (typeof RESOURCE_MINE_FILTER_KEYS)[number])) continue
    nextQuery[key] = value
  }
  if (filters.query) nextQuery.query = filters.query
  if (filters.visibility) nextQuery.visibility = filters.visibility
  return nextQuery
}

function sameResourceFilters(left: ResourceMineFilters, right: ResourceMineFilters): boolean {
  return RESOURCE_MINE_FILTER_KEYS.every((key) => left[key] === right[key])
}

async function submitFilters(): Promise<void> {
  const filters = inputResourceFilters()
  if (sameResourceFilters(filters, routeResourceFilters())) {
    await loadResources()
    return
  }
  await router.push({ name: 'resource-mine', query: resourceFiltersToQuery(filters) })
}

function loadMoreResources() {
  if (loadingMore.value || loading.value || !hasMore.value) return
  return loadResourcePage(page.value + 1, true)
}

function clearFilters() {
  query.value = ''
  visibility.value = ''
  void router.push({ name: 'resource-mine', query: resourceFiltersToQuery(inputResourceFilters()) })
}

function isDeleting(resource: ResourceItem): boolean {
  return deletingIDs.value.has(String(resource.id))
}

function setDeleting(resource: ResourceItem, value: boolean): void {
  const next = new Set(deletingIDs.value)
  const id = String(resource.id)
  if (value) {
    next.add(id)
  } else {
    next.delete(id)
  }
  deletingIDs.value = next
}

async function deleteResource(resource: ResourceItem): Promise<void> {
  if (isDeleting(resource)) return
  if (!window.confirm(t('resource.mine.deleteConfirm', { title: resource.title }))) {
    return
  }

  deleteError.value = ''
  setDeleting(resource, true)
  try {
    await api.resource.deleteResource(String(resource.id))
    resources.value = resources.value.filter(item => item.id !== resource.id)
    total.value = Math.max(0, total.value - 1)
  } catch (_error) {
    void _error
    deleteError.value = t('resource.mine.deleteFailed')
  } finally {
    setDeleting(resource, false)
  }
}

onMounted(() => {
  applyResourceFilters(routeResourceFilters())
  void loadResources()
})

watch(
  () => route.query,
  () => {
    applyResourceFilters(routeResourceFilters())
    void loadResources()
  },
)
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <section class="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 class="text-3xl font-extrabold tracking-tight text-text-primary">
            {{ t('resource.mine.title') }}
          </h1>
          <p class="mt-2 text-sm text-text-secondary">
            {{ t('resource.mine.subtitle') }}
          </p>
        </div>
        <RouterLink
          :to="{ name: 'resource-new' }"
          class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white no-underline transition-colors hover:bg-primary/90"
        >
          <PlusCircle :size="16" />
          {{ t('resource.list.publish') }}
        </RouterLink>
      </header>

      <form
        class="mb-6 grid gap-3 rounded-lg border border-border-light bg-bg-card p-4 shadow-xs md:grid-cols-[minmax(0,2fr)_minmax(160px,1fr)_auto]"
        @submit.prevent="submitFilters"
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
              :placeholder="t('resource.mine.searchPlaceholder')"
            />
          </span>
        </label>

        <label class="grid gap-1 text-xs font-semibold text-text-secondary">
          {{ t('resource.detail.visibility') }}
          <select
            v-model="visibility"
            class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors focus:border-primary"
          >
            <option value="">{{ t('resource.mine.visibilityAll') }}</option>
            <option value="public">{{ t('resource.visibility.public') }}</option>
            <option value="private">{{ t('resource.visibility.private') }}</option>
          </select>
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

      <p
        v-if="deleteError"
        role="alert"
        class="mb-4 rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
      >
        {{ deleteError }}
      </p>

      <div class="mb-4 flex min-h-6 items-center justify-between text-sm text-text-muted">
        <span v-if="!errorMessage && !loading">
          {{ t('resource.mine.total', { count: total }) }}
        </span>
      </div>

      <div
        v-if="loading"
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      >
        <div
          v-for="i in 6"
          :key="i"
          class="h-48 rounded-lg bg-bg-card shadow-xs animate-pulse"
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
          {{ t('resource.mine.emptyTitle') }}
        </h2>
        <p class="text-sm text-text-secondary">
          {{ t('resource.mine.emptyDesc') }}
        </p>
        <RouterLink
          :to="{ name: 'resource-new' }"
          class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white no-underline transition-colors hover:bg-primary/90"
        >
          <PlusCircle :size="16" />
          {{ t('resource.list.publish') }}
        </RouterLink>
      </div>

      <template v-else>
        <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <article
            v-for="resource in resources"
            :key="resource.id"
            class="flex min-h-48 flex-col rounded-lg border border-border-light bg-bg-card p-5 shadow-xs"
          >
            <div class="mb-3 flex items-start justify-between gap-3">
              <div class="min-w-0">
                <RouterLink
                  :to="{ name: 'resource-detail', params: { id: String(resource.id) } }"
                  class="text-base font-bold text-text-primary no-underline transition-colors hover:text-primary"
                >
                  {{ resource.title }}
                </RouterLink>
                <p class="mt-1 truncate text-xs text-text-muted">
                  {{ resourceMeta(resource) }}
                </p>
              </div>
              <span class="inline-flex shrink-0 items-center gap-1 rounded-full bg-bg-elevated px-2.5 py-1 text-xs font-medium text-text-secondary">
                <Eye v-if="resource.visibility === 'public'" :size="12" />
                <EyeOff v-else :size="12" />
                {{ t('resource.visibility.' + resource.visibility) }}
              </span>
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

            <div class="mt-auto flex items-center justify-between gap-3 pt-4">
              <span class="text-xs text-text-muted">
                {{ formatDate(resource.updatedAt) }}
              </span>
              <div class="flex items-center gap-2">
                <RouterLink
                  :to="{ name: 'resource-edit', params: { id: String(resource.id) } }"
                  class="inline-flex h-9 items-center justify-center gap-1 rounded-lg border border-border-light px-3 text-xs font-semibold text-text-secondary no-underline transition-colors hover:bg-bg-elevated hover:text-text-primary"
                >
                  <Pencil :size="14" />
                  {{ t('common.actions.edit') }}
                </RouterLink>
                <button
                  type="button"
                  class="inline-flex h-9 items-center justify-center gap-1 rounded-lg border border-danger/30 px-3 text-xs font-semibold text-danger transition-colors hover:bg-danger/10 disabled:cursor-wait disabled:opacity-70"
                  :disabled="isDeleting(resource)"
                  @click="deleteResource(resource)"
                >
                  <RefreshCw v-if="isDeleting(resource)" :size="14" class="animate-spin" />
                  <Trash2 v-else :size="14" />
                  {{ t('common.actions.delete') }}
                </button>
              </div>
            </div>
          </article>
        </div>

        <div
          v-if="hasMore || loadMoreError"
          class="mt-6 flex flex-col items-center gap-3"
        >
          <p
            v-if="loadMoreError"
            role="alert"
            class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {{ loadMoreError }}
          </p>
          <button
            v-if="hasMore"
            type="button"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-card px-5 text-sm font-semibold text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary disabled:cursor-wait disabled:opacity-70"
            :disabled="loadingMore"
            @click="loadMoreResources"
          >
            <RefreshCw v-if="loadingMore" :size="16" class="animate-spin" />
            <span>{{ loadingMore ? t('common.actions.loading') : t('common.actions.loadMore') }}</span>
          </button>
        </div>
      </template>
    </section>
  </div>
</template>
