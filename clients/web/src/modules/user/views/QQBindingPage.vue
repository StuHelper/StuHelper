<template>
  <div class="max-w-[680px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <header class="flex items-center gap-3 mb-6">
      <button
        class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
        :aria-label="t('common.actions.back')"
        @click="router.back()"
      >
        <ArrowLeft class="size-5" />
      </button>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.verification.qq.title') }}
      </h1>
    </header>

    <div
      v-if="qqBinding"
      class="bg-bg-card border border-green-500/30 rounded-xl p-5 shadow-card"
    >
      <div class="flex items-center gap-3 mb-5">
        <div class="p-2 bg-green-500/10 rounded-lg">
          <Bot class="size-6 text-green-500" />
        </div>
        <div>
          <h2 class="text-base font-bold text-text-primary m-0">
            {{ t('user.verification.qq.bound') }}
          </h2>
          <p class="text-sm text-text-muted m-0 mt-1">
            {{ verificationHint }}
          </p>
        </div>
      </div>

      <dl class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div class="rounded-lg bg-bg-base/60 p-4">
          <dt class="text-xs text-text-muted mb-1">
            {{ t('user.verification.qq.qqNumber') }}
          </dt>
          <dd class="m-0 text-sm font-semibold text-text-primary">
            {{ qqBinding.qqID }}
          </dd>
        </div>
        <div class="rounded-lg bg-bg-base/60 p-4">
          <dt class="text-xs text-text-muted mb-1">
            {{ t('user.verification.qq.nickname') }}
          </dt>
          <dd class="m-0 text-sm font-semibold text-text-primary">
            {{ qqBinding.qqNickname || t('user.verification.qq.emptyNickname') }}
          </dd>
        </div>
        <div class="rounded-lg bg-bg-base/60 p-4 sm:col-span-2">
          <dt class="text-xs text-text-muted mb-1">
            {{ t('user.verification.qq.boundAt') }}
          </dt>
          <dd class="m-0 text-sm font-semibold text-text-primary">
            {{ formatTime(qqBinding.boundAt) }}
          </dd>
        </div>
      </dl>
    </div>

    <div v-else class="bg-bg-card rounded-xl p-5 shadow-card">
      <div class="flex items-start gap-3 mb-5">
        <div class="p-2 bg-accent/10 rounded-lg shrink-0">
          <Bot class="size-6 text-accent" />
        </div>
        <div>
          <h2 class="text-base font-bold text-text-primary m-0">
            {{ t('user.verification.qq.unbound') }}
          </h2>
          <p class="text-sm text-text-muted m-0 mt-1">
            {{ t('user.verification.qq.desc') }}
          </p>
          <p class="text-sm text-text-muted m-0 mt-2">
            {{ verificationHint }}
          </p>
        </div>
      </div>

      <button
        type="button"
        class="w-full py-2.5 bg-text-primary text-bg-base rounded-lg text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white border-0 disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="creating"
        @click="onCreateCode"
      >
        {{ createButtonText }}
      </button>

      <div
        v-if="qqBindingCode"
        class="mt-5 rounded-xl border border-border bg-bg-base/60 p-4"
      >
        <p class="text-sm font-semibold text-text-primary m-0">
          {{ t('user.verification.qq.instruction') }}
        </p>
        <div class="mt-3 rounded-lg bg-bg-card px-4 py-3 font-mono text-sm text-text-primary break-all">
          {{ bindingCommand }}
        </div>
        <p class="text-xs text-text-muted m-0 mt-3">
          {{ t('user.verification.qq.expiresAt') }}：{{ formatTime(qqBindingCode.expiresAt) }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowLeft, Bot } from 'lucide-vue-next'

import { getErrorStatus } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { useVerificationStore } from '@/stores/verification'

const DEFAULT_QQ_BIND_COMMAND = '绑定'

const { t } = useI18n()
const router = useRouter()
const toast = useToast()
const verificationStore = useVerificationStore()

const creating = ref(false)

const qqBinding = computed(() => verificationStore.qqBinding)
const qqBindingCode = computed(() => verificationStore.qqBindingCode)
const verificationHint = computed(() => {
  if (verificationStore.studentVerified) {
    return t('user.verification.qq.verifiedHint')
  }
  return t('user.verification.qq.pendingHint')
})
const createButtonText = computed(() => {
  if (creating.value) {
    return t('user.verification.qq.creating')
  }
  if (qqBindingCode.value) {
    return t('user.verification.qq.regenerateCode')
  }
  return t('user.verification.qq.createCode')
})
const bindingCommand = computed(() => {
  if (!qqBindingCode.value) {
    return ''
  }
  return `${DEFAULT_QQ_BIND_COMMAND} ${qqBindingCode.value.code}`
})

async function onCreateCode() {
  creating.value = true
  try {
    await verificationStore.createQQBindingCode()
    toast.success(t('user.verification.qq.codeCreated'))
  } catch (error) {
    if (getErrorStatus(error) === 409) {
      await verificationStore.fetchQQBinding()
      toast.error(t('user.verification.qq.alreadyBound'))
      return
    }
    toast.error(t('common.actions.operationFailed'))
  } finally {
    creating.value = false
  }
}

function formatTime(value?: string | null) {
  if (!value) {
    return '--'
  }
  return new Date(value).toLocaleString()
}

onMounted(() => {
  void verificationStore.fetchStatus().catch(() => {
    toast.error(t('common.loadFailed'))
  })
})
</script>
