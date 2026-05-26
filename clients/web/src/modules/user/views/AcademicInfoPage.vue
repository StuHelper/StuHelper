<template>
  <div class="max-w-2xl mx-auto p-6 animate-fade-in max-sm:p-4">
    <header class="flex items-center gap-3 mb-6">
      <button
        class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:text-text-primary"
        :aria-label="t('common.actions.back')"
        @click="goBack"
      >
        <ArrowLeft class="size-5" />
      </button>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.verification.academic.title') }}
      </h1>
    </header>

    <!-- Loading -->
    <div v-if="loading" class="bg-bg-card rounded-xl p-8 flex justify-center" role="status" aria-busy="true">
      <div class="size-8 border-2 border-border border-t-primary rounded-full animate-spin">
        <span class="sr-only">{{ t('common.actions.loading') }}</span>
      </div>
    </div>

    <!-- Forbidden: needs verification -->
    <div v-else-if="errorState === 'forbidden'" class="bg-bg-card rounded-xl p-8 text-center shadow-card">
      <p class="text-text-primary mb-4">{{ t('user.verification.academic.needVerification') }}</p>
      <router-link
        to="/user/student-verification"
        class="inline-block px-4 py-2 bg-text-primary text-bg-base rounded-lg text-sm font-medium no-underline transition-all duration-fast hover:bg-accent hover:text-white"
      >
        {{ t('user.verification.academic.goVerify') }}
      </router-link>
    </div>

    <!-- Not found: empty -->
    <div v-else-if="errorState === 'not-found'" class="bg-bg-card rounded-xl p-8 text-center text-text-muted shadow-card">
      {{ t('user.verification.academic.noData') }}
    </div>

    <!-- Generic error -->
    <div v-else-if="errorState === 'unknown'" class="bg-bg-card rounded-xl p-8 text-center shadow-card">
      <p class="text-red-500 mb-4">{{ t('common.loadFailed') }}</p>
      <button
        class="px-4 py-2 bg-text-primary text-bg-base rounded-lg text-sm font-medium border-0 cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white"
        @click="load"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>

    <!-- Success: render fields -->
    <div v-else-if="info" class="bg-bg-card rounded-xl p-6 shadow-card">
      <dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-4">
        <div v-for="f in displayFields" :key="f.key">
          <dt class="text-xs text-text-muted mb-1">{{ f.label }}</dt>
          <dd class="text-sm text-text-primary font-medium m-0">{{ f.value }}</dd>
        </div>
      </dl>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import { api } from '@/api'
import { ApiError } from '@/api/errors'
import type { components } from '@stuhelper/shared/types'

type AcademicStudentInfo = components['schemas']['AcademicStudentInfo']
type ErrorState = 'forbidden' | 'not-found' | 'unknown' | null

const { t } = useI18n()
const router = useRouter()

const loading = ref(false)
const errorState = ref<ErrorState>(null)
const info = ref<AcademicStudentInfo | null>(null)

function readAcademicInfoPayload(payload: unknown): AcademicStudentInfo {
  if (!payload || typeof payload !== 'object') {
    throw new Error('invalid academic info response')
  }

  const raw = payload as Record<string, unknown>
  if (typeof raw.xh !== 'string' || raw.xh.trim() === '') {
    throw new Error('invalid academic info response')
  }
  if (typeof raw.xm !== 'string' || raw.xm.trim() === '') {
    throw new Error('invalid academic info response')
  }

  for (const field of ['yxdm', 'zydm', 'bjdm', 'xznj', 'rxnj', 'pyccdm', 'sjh', 'dzxx'] as const) {
    if (raw[field] !== undefined && typeof raw[field] !== 'string') {
      throw new Error('invalid academic info response')
    }
  }

  return raw as AcademicStudentInfo
}

const displayFields = computed(() => {
  const v = info.value
  if (!v) return []
  const all: { key: string; label: string; value?: string }[] = [
    { key: 'xh', label: t('user.verification.academic.studentId'), value: v.xh },
    { key: 'xm', label: t('user.verification.academic.name'), value: v.xm },
    { key: 'yxdm', label: t('user.verification.academic.college'), value: v.yxdm },
    { key: 'zydm', label: t('user.verification.academic.major'), value: v.zydm },
    { key: 'bjdm', label: t('user.verification.academic.class'), value: v.bjdm },
    { key: 'xznj', label: t('user.verification.academic.currentGrade'), value: v.xznj },
    { key: 'rxnj', label: t('user.verification.academic.enrollGrade'), value: v.rxnj },
    { key: 'pyccdm', label: t('user.verification.academic.eduLevel'), value: v.pyccdm },
    { key: 'sjh', label: t('user.verification.academic.phone'), value: v.sjh },
    { key: 'dzxx', label: t('user.verification.academic.email'), value: v.dzxx },
  ]
  return all.filter((f) => f.value !== undefined && f.value !== null && f.value !== '')
})

function goBack(): void {
  if (window.history.length > 1) {
    router.back()
  } else {
    void router.push('/user/reviews')
  }
}

async function load(): Promise<void> {
  loading.value = true
  errorState.value = null
  try {
    const res = await api.identity.getAcademicInfo()
    info.value = readAcademicInfoPayload(res.data?.data)
  } catch (err: unknown) {
    const status = err instanceof ApiError ? err.status : undefined
    if (status === 401) {
      void router.push('/login')
      return
    }
    if (status === 403) {
      errorState.value = 'forbidden'
    } else if (status === 404) {
      errorState.value = 'not-found'
    } else {
      errorState.value = 'unknown'
    }
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>
