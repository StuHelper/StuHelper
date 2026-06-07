<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { ArrowLeft, FileText, RefreshCw, Save, Upload } from 'lucide-vue-next'
import { api } from '@/api'
import {
  buildCreateResourcePayload,
  buildResourceMetadataPayload,
  createEmptyResourceForm,
  resourceToForm,
} from '@/modules/resource/resourceForm'
import {
  readResourceItemPayload,
  type ResourceItem,
} from '@/modules/resource/resourcePayload'

const MAX_RESOURCE_UPLOAD_SIZE = 10 * 1024 * 1024

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const form = reactive(createEmptyResourceForm())
const selectedFile = ref<File | null>(null)
const existingResource = ref<ResourceItem | null>(null)
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
let loadRequestSeq = 0

const isEditMode = computed(() => route.name === 'resource-edit')
const pageTitle = computed(() =>
  isEditMode.value ? t('resource.form.editTitle') : t('resource.form.createTitle'),
)
const pageSubtitle = computed(() =>
  isEditMode.value ? t('resource.form.editSubtitle') : t('resource.form.createSubtitle'),
)
const submitLabel = computed(() =>
  saving.value
    ? t('resource.form.saving')
    : isEditMode.value
      ? t('resource.form.saveEdit')
      : t('resource.form.publish'),
)

const resourceID = computed(() => {
  const raw = route.params.id
  const value = Array.isArray(raw) ? raw[0] : raw
  if (typeof value !== 'string' || !/^[1-9]\d*$/.test(value)) {
    return null
  }
  return value
})

function resetForm(): void {
  Object.assign(form, createEmptyResourceForm())
  selectedFile.value = null
  existingResource.value = null
}

async function loadResourceForEdit(): Promise<void> {
  const requestSeq = ++loadRequestSeq
  errorMessage.value = ''

  if (!isEditMode.value) {
    resetForm()
    loading.value = false
    return
  }

  const id = resourceID.value
  if (id === null) {
    resetForm()
    loading.value = false
    errorMessage.value = t('resource.detail.notFound')
    return
  }

  loading.value = true
  try {
    const res = await api.resource.getResource(id)
    const data = readResourceItemPayload(res.data?.data, 'Invalid resource response')
    if (requestSeq !== loadRequestSeq) return
    existingResource.value = data
    Object.assign(form, resourceToForm(data))
  } catch (_error) {
    void _error
    if (requestSeq !== loadRequestSeq) return
    resetForm()
    errorMessage.value = t('resource.form.loadFailed')
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

function handleFileChange(event: Event): void {
  const target = event.target
  if (!(target instanceof HTMLInputElement)) return
  selectedFile.value = target.files?.[0] ?? null
}

function validateMetadata(): ReturnType<typeof buildResourceMetadataPayload> | null {
  try {
    const payload = buildResourceMetadataPayload(form)
    if (!payload.title) {
      errorMessage.value = t('resource.form.titleRequired')
      return null
    }
    return payload
  } catch (_error) {
    void _error
    errorMessage.value = t('resource.form.invalidBindings')
    return null
  }
}

function validateSelectedFile(): File | null {
  const file = selectedFile.value
  if (!file) {
    errorMessage.value = t('resource.form.fileRequired')
    return null
  }
  if (file.size <= 0 || file.size > MAX_RESOURCE_UPLOAD_SIZE) {
    errorMessage.value = t('resource.form.fileSizeInvalid')
    return null
  }
  return file
}

async function submitForm(): Promise<void> {
  if (saving.value || loading.value) return
  errorMessage.value = ''

  const metadata = validateMetadata()
  if (!metadata) return

  const id = resourceID.value
  if (isEditMode.value && id === null) {
    errorMessage.value = t('resource.detail.notFound')
    return
  }

  saving.value = true
  try {
    if (isEditMode.value && id !== null) {
      const res = await api.resource.updateResource(id, metadata)
      const updated = readResourceItemPayload(res.data?.data, 'Invalid resource response')
      await router.push({ name: 'resource-detail', params: { id: String(updated.id) } })
      return
    }

    const file = validateSelectedFile()
    if (!file) return
    const payload = await buildCreateResourcePayload(form, file)
    const res = await api.resource.createResource(payload)
    const created = readResourceItemPayload(res.data?.data, 'Invalid resource response')
    await router.push({ name: 'resource-detail', params: { id: String(created.id) } })
  } catch (_error) {
    void _error
    errorMessage.value = t('resource.form.saveFailed')
  } finally {
    saving.value = false
  }
}

watch(
  () => [route.name, route.params.id] as const,
  () => {
    void loadResourceForEdit()
  },
  { immediate: true },
)
</script>

<template>
  <div class="min-h-screen bg-bg-base">
    <section class="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <RouterLink
        :to="{ name: 'resource-mine' }"
        class="mb-6 inline-flex items-center gap-2 rounded-lg px-0 text-sm font-semibold text-text-secondary no-underline transition-colors hover:text-primary"
      >
        <ArrowLeft :size="16" />
        {{ t('resource.form.backToMine') }}
      </RouterLink>

      <header class="mb-6">
        <h1 class="text-3xl font-extrabold tracking-tight text-text-primary">
          {{ pageTitle }}
        </h1>
        <p class="mt-2 text-sm text-text-secondary">
          {{ pageSubtitle }}
        </p>
      </header>

      <div
        v-if="loading"
        class="space-y-4"
      >
        <div class="h-24 rounded-lg bg-bg-card shadow-xs animate-pulse" />
        <div class="h-96 rounded-lg bg-bg-card shadow-xs animate-pulse" />
      </div>

      <div
        v-else-if="errorMessage && isEditMode && !existingResource"
        role="alert"
        class="flex min-h-[320px] flex-col items-center justify-center gap-4 rounded-lg border border-border-light bg-bg-card p-8 text-center"
      >
        <FileText :size="42" class="text-text-muted" />
        <p class="text-sm font-medium text-danger">{{ errorMessage }}</p>
        <button
          type="button"
          class="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary/90"
          @click="loadResourceForEdit"
        >
          <RefreshCw :size="16" />
          {{ t('resource.list.retry') }}
        </button>
      </div>

      <form
        v-else
        class="rounded-lg border border-border-light bg-bg-card p-5 shadow-xs"
        @submit.prevent="submitForm"
      >
        <p
          v-if="errorMessage"
          role="alert"
          class="mb-4 rounded-lg border border-danger/30 bg-danger/10 px-3 py-2 text-sm text-danger"
        >
          {{ errorMessage }}
        </p>

        <div class="grid gap-4">
          <label class="grid gap-1 text-xs font-semibold text-text-secondary">
            {{ t('resource.form.titleLabel') }}
            <input
              v-model="form.title"
              type="text"
              autocomplete="off"
              class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :placeholder="t('resource.form.titlePlaceholder')"
            />
          </label>

          <label class="grid gap-1 text-xs font-semibold text-text-secondary">
            {{ t('resource.form.descriptionLabel') }}
            <textarea
              v-model="form.description"
              rows="4"
              class="rounded-lg border border-border-light bg-bg-base px-3 py-2 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :placeholder="t('resource.form.descriptionPlaceholder')"
            />
          </label>

          <div class="grid gap-4 md:grid-cols-2">
            <label class="grid gap-1 text-xs font-semibold text-text-secondary">
              {{ t('resource.form.categoryLabel') }}
              <input
                v-model="form.category"
                type="text"
                autocomplete="off"
                class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
                :placeholder="t('resource.form.categoryPlaceholder')"
              />
            </label>

            <label class="grid gap-1 text-xs font-semibold text-text-secondary">
              {{ t('resource.form.visibilityLabel') }}
              <select
                v-model="form.visibility"
                class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors focus:border-primary"
              >
                <option value="public">{{ t('resource.visibility.public') }}</option>
                <option value="private">{{ t('resource.visibility.private') }}</option>
              </select>
            </label>
          </div>

          <label class="grid gap-1 text-xs font-semibold text-text-secondary">
            {{ t('resource.form.tagsLabel') }}
            <input
              v-model="form.tagsText"
              type="text"
              autocomplete="off"
              class="h-11 rounded-lg border border-border-light bg-bg-base px-3 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :placeholder="t('resource.form.tagsPlaceholder')"
            />
          </label>

          <label class="grid gap-1 text-xs font-semibold text-text-secondary">
            {{ t('resource.form.bindingsLabel') }}
            <textarea
              v-model="form.bindingsText"
              rows="3"
              class="rounded-lg border border-border-light bg-bg-base px-3 py-2 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary"
              :placeholder="t('resource.form.bindingsPlaceholder')"
            />
          </label>

          <label
            v-if="!isEditMode"
            class="grid gap-1 text-xs font-semibold text-text-secondary"
          >
            {{ t('resource.form.fileLabel') }}
            <span class="flex min-h-28 flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border-light bg-bg-base px-4 py-5 text-center transition-colors hover:border-primary/50">
              <Upload :size="22" class="text-text-muted" />
              <span class="text-sm font-medium text-text-primary">
                {{ selectedFile?.name || t('resource.form.filePlaceholder') }}
              </span>
              <span class="text-xs text-text-muted">
                {{ t('resource.form.fileHint') }}
              </span>
              <input
                type="file"
                class="sr-only"
                @change="handleFileChange"
              />
            </span>
          </label>

          <div
            v-else-if="existingResource"
            class="rounded-lg border border-border-light bg-bg-base p-4 text-sm text-text-secondary"
          >
            <p class="font-semibold text-text-primary">{{ t('resource.form.currentFile') }}</p>
            <p class="mt-1 break-all">{{ existingResource.latestVersion.filename }}</p>
          </div>
        </div>

        <div class="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
          <RouterLink
            :to="{ name: 'resource-mine' }"
            class="inline-flex h-11 items-center justify-center rounded-lg border border-border-light px-4 text-sm font-semibold text-text-secondary no-underline transition-colors hover:bg-bg-elevated hover:text-text-primary"
          >
            {{ t('common.actions.cancel') }}
          </RouterLink>
          <button
            type="submit"
            class="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-primary px-5 text-sm font-semibold text-white transition-colors hover:bg-primary/90 disabled:cursor-wait disabled:opacity-70"
            :disabled="saving"
          >
            <RefreshCw v-if="saving" :size="16" class="animate-spin" />
            <Save v-else :size="16" />
            {{ submitLabel }}
          </button>
        </div>
      </form>
    </section>
  </div>
</template>
