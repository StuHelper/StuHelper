<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute } from 'vue-router'
import { ArrowLeft, Download, FileText, Link2, RefreshCw, Tag } from 'lucide-vue-next'
import { api } from '@/api'
import {
  readResourceDownloadURLPayload,
  readResourceItemPayload,
  type ResourceItem,
} from '@/modules/resource/resourcePayload'

const { t, locale } = useI18n()
const route = useRoute()

const resource = ref<ResourceItem | null>(null)
const loading = ref(true)
const downloadLoading = ref(false)
const errorMessage = ref('')
const downloadError = ref('')
let loadRequestSeq = 0

const resourceID = computed(() => {
  const raw = route.params.id
  const value = Array.isArray(raw) ? raw[0] : raw
  const id = typeof value === 'string' ? Number(value) : NaN
  return Number.isInteger(id) && id > 0 ? id : null
})

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(locale.value, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
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

async function loadResource() {
  const requestSeq = ++loadRequestSeq
  const id = resourceID.value

  if (id === null) {
    resource.value = null
    loading.value = false
    errorMessage.value = t('resource.detail.notFound')
    downloadError.value = ''
    return
  }

  loading.value = true
  errorMessage.value = ''
  downloadError.value = ''
  try {
    const res = await api.resource.getResource(id)
    const nextResource = readResourceItemPayload(
      res.data?.data,
      'Invalid resource response',
    )
    if (requestSeq !== loadRequestSeq) return
    resource.value = nextResource
  } catch (_error) {
    void _error
    if (requestSeq !== loadRequestSeq) return
    resource.value = null
    errorMessage.value = t('resource.detail.loadFailed')
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

async function downloadResource() {
  if (!resource.value) return
  downloadLoading.value = true
  downloadError.value = ''
  try {
    const res = await api.resource.getDownloadURL(resource.value.id)
    const data = readResourceDownloadURLPayload(
      res.data?.data,
      'Invalid resource download response',
    )
    window.location.assign(data.url)
  } catch (_error) {
    void _error
    downloadError.value = t('resource.detail.downloadFailed')
  } finally {
    downloadLoading.value = false
  }
}

watch(resourceID, () => {
  void loadResource()
}, { immediate: true })
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <section class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <RouterLink
        :to="{ name: 'resource-list' }"
        class="mb-6 inline-flex items-center gap-2 rounded-lg px-0 text-sm font-semibold text-text-secondary no-underline transition-colors hover:text-primary"
      >
        <ArrowLeft :size="16" />
        {{ t('resource.detail.back') }}
      </RouterLink>

      <div
        v-if="loading"
        class="space-y-4"
      >
        <div class="h-28 rounded-lg bg-bg-card shadow-xs animate-pulse" />
        <div class="h-72 rounded-lg bg-bg-card shadow-xs animate-pulse" />
      </div>

      <div
        v-else-if="errorMessage"
        role="alert"
        class="flex min-h-[320px] flex-col items-center justify-center gap-4 rounded-lg border border-border-light bg-bg-card p-8 text-center"
      >
        <FileText :size="42" class="text-text-muted" />
        <p class="text-sm font-medium text-danger">{{ errorMessage }}</p>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary/90"
          @click="loadResource"
        >
          <RefreshCw :size="16" />
          {{ t('resource.list.retry') }}
        </button>
      </div>

      <article
        v-else-if="resource"
        class="space-y-6"
      >
        <header class="rounded-lg border border-border-light bg-bg-card p-6 shadow-xs">
          <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="min-w-0">
              <p class="mb-2 text-xs font-semibold uppercase text-primary">
                {{ t('resource.visibility.' + resource.visibility) }}
              </p>
              <h1 class="text-3xl font-extrabold tracking-tight text-text-primary">
                {{ resource.title || t('resource.detail.titleFallback') }}
              </h1>
              <p class="mt-3 max-w-2xl text-sm leading-6 text-text-secondary">
                {{ resource.description || t('resource.detail.noDescription') }}
              </p>
            </div>
            <button
              type="button"
              class="inline-flex h-11 shrink-0 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white transition-colors hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
              :disabled="downloadLoading"
              @click="downloadResource"
            >
              <Download :size="16" />
              {{ downloadLoading ? t('resource.detail.downloading') : t('resource.detail.download') }}
            </button>
          </div>
          <p
            v-if="downloadError"
            role="alert"
            class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {{ downloadError }}
          </p>
        </header>

        <div class="grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(260px,0.8fr)]">
          <section class="rounded-lg border border-border-light bg-bg-card p-5 shadow-xs">
            <h2 class="mb-4 text-lg font-bold text-text-primary">
              {{ t('resource.detail.fileInfo') }}
            </h2>
            <dl class="grid gap-4 text-sm">
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.filename') }}</dt>
                <dd class="break-all text-text-primary">{{ resource.latestVersion.filename }}</dd>
              </div>
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.contentType') }}</dt>
                <dd class="text-text-primary">{{ resource.latestVersion.contentType }}</dd>
              </div>
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.size') }}</dt>
                <dd class="text-text-primary">{{ formatFileSize(resource.latestVersion.sizeBytes) }}</dd>
              </div>
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.version') }}</dt>
                <dd class="text-text-primary">v{{ resource.latestVersion.versionNo }}</dd>
              </div>
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.uploadedAt') }}</dt>
                <dd class="text-text-primary">{{ formatDate(resource.latestVersion.createdAt) }}</dd>
              </div>
            </dl>
          </section>

          <section class="rounded-lg border border-border-light bg-bg-card p-5 shadow-xs">
            <h2 class="mb-4 text-lg font-bold text-text-primary">
              {{ t('resource.detail.metadata') }}
            </h2>
            <dl class="grid gap-4 text-sm">
              <div class="grid gap-1">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.category') }}</dt>
                <dd class="text-text-primary">{{ resource.category || '-' }}</dd>
              </div>
              <div class="grid gap-2">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.tags') }}</dt>
                <dd class="flex flex-wrap gap-2">
                  <span
                    v-for="item in resource.tags"
                    :key="item"
                    class="inline-flex items-center gap-1 rounded-full bg-primary-alpha px-2.5 py-1 text-xs font-medium text-primary"
                  >
                    <Tag :size="12" />
                    {{ item }}
                  </span>
                  <span v-if="resource.tags.length === 0" class="text-text-muted">-</span>
                </dd>
              </div>
              <div class="grid gap-2">
                <dt class="text-xs font-semibold text-text-muted">{{ t('resource.detail.bindings') }}</dt>
                <dd class="flex flex-wrap gap-2">
                  <span
                    v-for="binding in resource.bindings"
                    :key="`${binding.type}:${binding.value}`"
                    class="inline-flex items-center gap-1 rounded-full bg-bg-elevated px-2.5 py-1 text-xs font-medium text-text-secondary"
                  >
                    <Link2 :size="12" />
                    {{ binding.type }}: {{ binding.value }}
                  </span>
                  <span v-if="resource.bindings.length === 0" class="text-text-muted">-</span>
                </dd>
              </div>
            </dl>
          </section>
        </div>
      </article>
    </section>
  </div>
</template>
