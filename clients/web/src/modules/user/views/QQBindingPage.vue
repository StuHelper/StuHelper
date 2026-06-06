<template>
  <div class="max-w-[680px] mx-auto p-6 animate-fade-in max-sm:p-4">
    <header class="flex items-center gap-3 mb-6">
      <button
        class="p-2 bg-transparent rounded-lg text-text-muted cursor-pointer transition-all duration-fast hover:border-text-primary hover:text-text-primary"
        :aria-label="t('common.actions.back')"
        @click="goBack"
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
        <div class="flex flex-wrap items-center justify-between gap-3">
          <p class="text-sm font-semibold text-text-primary m-0">
            {{ t('user.verification.qq.instruction') }}
          </p>
          <span
            class="inline-flex items-center gap-2 rounded-md border border-border bg-bg-card px-2.5 py-1 text-xs font-medium text-text-muted"
          >
            <Bot class="size-3.5" aria-hidden="true" />
            {{ botEntryLabel }}
          </span>
        </div>
        <div class="mt-3 flex items-stretch overflow-hidden rounded-lg border border-border bg-bg-card">
          <code class="min-w-0 flex-1 px-4 py-3 font-mono text-sm text-text-primary break-all">
            {{ bindingCommand }}
          </code>
          <button
            type="button"
            class="inline-flex w-12 shrink-0 items-center justify-center border-0 border-l border-solid border-border bg-bg-card text-text-muted transition-colors duration-fast hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
            :aria-label="t('user.verification.qq.copyCommand')"
            :disabled="copying"
            @click="copyBindingCommand"
          >
            <Check v-if="commandCopied" class="size-4 text-green-500" aria-hidden="true" />
            <Clipboard v-else class="size-4" aria-hidden="true" />
          </button>
        </div>
        <p class="text-xs text-text-muted m-0 mt-3">
          {{ t('user.verification.qq.expiresAt') }}：{{ formatTime(qqBindingCode.expiresAt) }}
        </p>
        <button
          type="button"
          class="mt-4 inline-flex items-center gap-2 rounded-lg border border-border bg-bg-card px-3 py-2 text-sm font-medium text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="checking"
          @click="onRefreshStatus"
        >
          <RefreshCw class="size-4" :class="{ 'animate-spin': checking }" aria-hidden="true" />
          {{
            checking
              ? t('user.verification.qq.checkingStatus')
              : t('user.verification.qq.refreshStatus')
          }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ArrowLeft, Bot, Check, Clipboard, RefreshCw } from 'lucide-vue-next'

import { getErrorStatus } from '@/api/errors'
import { useToast } from '@/composables/useToast'
import { useVerificationStore } from '@/stores/verification'

const DEFAULT_QQ_BIND_COMMAND = '绑定'
const QQ_BINDING_STATUS_POLL_INTERVAL_MS = 3000
const COPY_FEEDBACK_RESET_MS = 1600
const configuredBotEntry = import.meta.env.VITE_QQ_BOT_ENTRY?.trim() || ''
const configuredBindCommand =
  import.meta.env.VITE_QQ_BIND_COMMAND?.trim() || DEFAULT_QQ_BIND_COMMAND

const { t } = useI18n()
const router = useRouter()
const toast = useToast()
const verificationStore = useVerificationStore()

const creating = ref(false)
const checking = ref(false)
const copying = ref(false)
const commandCopied = ref(false)
let statusPollTimer: ReturnType<typeof setInterval> | null = null
let copyFeedbackTimer: ReturnType<typeof setTimeout> | null = null

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
  return `${configuredBindCommand} ${qqBindingCode.value.code}`
})
const botEntryLabel = computed(() => {
  if (configuredBotEntry) {
    return configuredBotEntry
  }
  return t('user.verification.qq.botEntryUnavailable')
})

async function onCreateCode() {
  creating.value = true
  try {
    await verificationStore.createQQBindingCode()
    startStatusPolling()
    toast.success(t('user.verification.qq.codeCreated'))
  } catch (error) {
    if (getErrorStatus(error) === 409) {
      stopStatusPolling()
      await verificationStore.fetchQQBinding()
      toast.error(t('user.verification.qq.alreadyBound'))
      return
    }
    toast.error(t('common.actions.operationFailed'))
  } finally {
    creating.value = false
  }
}

async function refreshQQBindingStatus(options: { silent?: boolean } = {}) {
  if (checking.value) {
    return
  }
  checking.value = true
  try {
    const binding = await verificationStore.fetchQQBinding()
    if (binding) {
      stopStatusPolling()
      toast.success(t('user.verification.qq.statusUpdated'))
      return
    }
    if (!options.silent) {
      toast.info(t('user.verification.qq.notYetBound'))
    }
  } catch {
    if (!options.silent) {
      toast.error(t('common.loadFailed'))
    }
  } finally {
    checking.value = false
  }
}

function onRefreshStatus() {
  void refreshQQBindingStatus()
}

async function copyBindingCommand() {
  if (!bindingCommand.value || copying.value) {
    return
  }
  copying.value = true
  try {
    if (!navigator.clipboard?.writeText) {
      throw new Error('clipboard unavailable')
    }
    await navigator.clipboard.writeText(bindingCommand.value)
    commandCopied.value = true
    if (copyFeedbackTimer) {
      clearTimeout(copyFeedbackTimer)
    }
    copyFeedbackTimer = setTimeout(() => {
      commandCopied.value = false
    }, COPY_FEEDBACK_RESET_MS)
    toast.success(t('user.verification.qq.commandCopied'))
  } catch {
    toast.error(t('user.verification.qq.copyCommandFailed'))
  } finally {
    copying.value = false
  }
}

function startStatusPolling() {
  stopStatusPolling()
  statusPollTimer = setInterval(() => {
    void refreshQQBindingStatus({ silent: true })
  }, QQ_BINDING_STATUS_POLL_INTERVAL_MS)
}

function stopStatusPolling() {
  if (statusPollTimer) {
    clearInterval(statusPollTimer)
    statusPollTimer = null
  }
}

function formatTime(value?: string | null) {
  if (!value) {
    return '--'
  }
  return new Date(value).toLocaleString()
}

function goBack() {
  void router.push('/identity')
}

onMounted(() => {
  void verificationStore.fetchStatus().catch(() => {
    toast.error(t('common.loadFailed'))
  })
})

onUnmounted(() => {
  stopStatusPolling()
  if (copyFeedbackTimer) {
    clearTimeout(copyFeedbackTimer)
    copyFeedbackTimer = null
  }
})
</script>
