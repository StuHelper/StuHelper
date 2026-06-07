<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Download, FileText, Link2, Pencil, RefreshCw, Tag, Trash2 } from 'lucide-vue-next'
import { api } from '@/api'
import { getErrorStatus, isApiError } from '@/api/errors'
import { updatePageMeta } from '@/composables/usePageMeta'
import {
  readResourceDownloadURLPayload,
  readResourceItemPayload,
  type ResourceItem,
} from '@/modules/resource/resourcePayload'
import { useAuthStore } from '@/stores/auth'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const resource = ref<ResourceItem | null>(null)
const loadedResourceID = ref<string | null>(null)
const loading = ref(true)
const downloadLoading = ref(false)
const deleteLoading = ref(false)
const errorMessage = ref('')
const canRetryLoad = ref(false)
const downloadError = ref('')
const deleteError = ref('')
let loadRequestSeq = 0
let downloadRequestSeq = 0

const resourceID = computed(() => {
  const raw = route.params.id
  const value = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) {
    return null
  }
  return value
})

const canEditResource = computed(() =>
  Boolean(
    resource.value?.ownerUserID &&
      authStore.user?.id &&
      resource.value.ownerUserID === authStore.user.id,
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

function isResourceNotFoundError(error: unknown) {
  return getErrorStatus(error) === 404 || (isApiError(error) && error.code === 'A0000404')
}

async function loadResource() {
  const requestSeq = ++loadRequestSeq
  downloadRequestSeq += 1
  const id = resourceID.value

  if (id === null) {
    resource.value = null
    loadedResourceID.value = null
    loading.value = false
    downloadLoading.value = false
    deleteLoading.value = false
    errorMessage.value = t('resource.detail.notFound')
    canRetryLoad.value = false
    downloadError.value = ''
    deleteError.value = ''
    updatePageMeta({
      title: t('resource.detail.titleFallback'),
      description: t('resource.detail.notFound'),
    })
    return
  }

  loading.value = true
  errorMessage.value = ''
  canRetryLoad.value = false
  downloadError.value = ''
  deleteError.value = ''
  deleteLoading.value = false
  try {
    const res = await api.resource.getResource(id)
    const nextResource = readResourceItemPayload(
      res.data?.data,
      'Invalid resource response',
    )
    if (requestSeq !== loadRequestSeq) return
    resource.value = nextResource
    loadedResourceID.value = id
    updatePageMeta({
      title: nextResource.title || t('resource.detail.titleFallback'),
      description: nextResource.description || t('resource.detail.noDescription'),
    })
  } catch (error) {
    if (requestSeq !== loadRequestSeq) return
    resource.value = null
    loadedResourceID.value = null
    const messageKey = isResourceNotFoundError(error)
      ? 'resource.detail.notFound'
      : 'resource.detail.loadFailed'
    errorMessage.value = t(messageKey)
    canRetryLoad.value = messageKey === 'resource.detail.loadFailed'
    updatePageMeta({
      title: t('resource.detail.titleFallback'),
      description: t(messageKey),
    })
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

async function deleteResource() {
  if (!resource.value) return
  const resourceID = loadedResourceID.value
  if (resourceID === null) return
  if (!window.confirm(t('resource.detail.deleteConfirm', { title: resource.value.title }))) {
    return
  }

  deleteLoading.value = true
  deleteError.value = ''
  try {
    await api.resource.deleteResource(resourceID)
    await router.push({ name: 'resource-mine' })
  } catch (_error) {
    void _error
    deleteError.value = t('resource.detail.deleteFailed')
  } finally {
    deleteLoading.value = false
  }
}

async function downloadResource() {
  if (!resource.value) return
  const resourceID = loadedResourceID.value
  if (resourceID === null) return
  const requestSeq = ++downloadRequestSeq
  downloadLoading.value = true
  downloadError.value = ''
  try {
    const res = await api.resource.getDownloadURL(resourceID)
    const data = readResourceDownloadURLPayload(
      res.data?.data,
      'Invalid resource download response',
    )
    if (requestSeq !== downloadRequestSeq || loadedResourceID.value !== resourceID) return
    window.location.assign(data.url)
  } catch (_error) {
    void _error
    if (requestSeq !== downloadRequestSeq || loadedResourceID.value !== resourceID) return
    downloadError.value = t('resource.detail.downloadFailed')
  } finally {
    if (requestSeq === downloadRequestSeq && loadedResourceID.value === resourceID) {
      downloadLoading.value = false
    }
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
        :to="{ name: 'resource-list', query: route.query }"
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
          v-if="canRetryLoad"
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary/90"
          data-resource-detail-retry-button
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
            <div class="flex shrink-0 flex-wrap gap-2">
              <RouterLink
                v-if="canEditResource && loadedResourceID"
                :to="{ name: 'resource-edit', params: { id: loadedResourceID } }"
                class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-base px-4 text-sm font-semibold text-text-secondary no-underline transition-colors hover:bg-bg-elevated hover:text-text-primary"
              >
                <Pencil :size="16" />
                {{ t('common.actions.edit') }}
              </RouterLink>
              <button
                v-if="canEditResource"
                type="button"
                class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-danger/30 px-4 text-sm font-semibold text-danger transition-colors hover:bg-danger/10 disabled:cursor-wait disabled:opacity-70"
                :disabled="deleteLoading"
                @click="deleteResource"
              >
                <RefreshCw v-if="deleteLoading" :size="16" class="animate-spin" />
                <Trash2 v-else :size="16" />
                {{ deleteLoading ? t('common.actions.deleting') : t('common.actions.delete') }}
              </button>
              <button
                type="button"
                class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-4 text-sm font-semibold text-white transition-colors hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
                :disabled="downloadLoading"
                @click="downloadResource"
              >
                <Download :size="16" />
                {{ downloadLoading ? t('resource.detail.downloading') : t('resource.detail.download') }}
              </button>
            </div>
          </div>
          <p
            v-if="downloadError"
            role="alert"
            class="rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {{ downloadError }}
          </p>
          <p
            v-if="deleteError"
            role="alert"
            class="mt-2 rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {{ deleteError }}
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
