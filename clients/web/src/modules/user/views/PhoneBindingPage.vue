<template>
  <main class="mx-auto w-full max-w-3xl px-4 py-6 sm:px-6 lg:py-10" data-phone-binding-page>
    <header class="mb-6 flex items-start gap-3">
      <button
        type="button"
        class="grid size-11 shrink-0 place-items-center rounded-lg border border-border bg-bg-card text-text-muted transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="t('common.actions.back')"
        @click="goBack"
      >
        <ArrowLeft class="size-5" aria-hidden="true" />
      </button>
      <div>
        <p class="m-0 text-xs font-bold uppercase tracking-[0.18em] text-primary">
          {{ t('user.verification.phone.platform.eyebrow') }}
        </p>
        <h1 class="m-0 mt-1 text-2xl font-extrabold tracking-tight text-text-primary sm:text-3xl">
          {{ t('user.verification.phone.platform.title') }}
        </h1>
        <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
          {{ t('user.verification.phone.platform.subtitle') }}
        </p>
      </div>
    </header>

    <div v-if="loading" class="grid min-h-64 place-items-center rounded-2xl border border-border bg-bg-card shadow-card" role="status" aria-busy="true">
      <LoaderCircle class="size-7 animate-spin text-primary" aria-hidden="true" />
      <span class="sr-only">{{ t('common.actions.loading') }}</span>
    </div>

    <div v-else class="grid gap-5">
      <section
        class="rounded-2xl border bg-bg-card p-5 shadow-card sm:p-6"
        :class="phoneStatus?.state === 'verified' ? 'border-success/30' : 'border-border'"
      >
        <div class="flex items-start gap-4">
          <span
            class="grid size-12 shrink-0 place-items-center rounded-xl"
            :class="phoneStatus?.state === 'verified' ? 'bg-success/15 text-success' : phoneStatus?.state === 'syncing' ? 'bg-warning/15 text-warning' : 'bg-bg-elevated text-text-muted'"
          >
            <BadgeCheck v-if="phoneStatus?.state === 'verified'" class="size-6" aria-hidden="true" />
            <RefreshCw v-else-if="phoneStatus?.state === 'syncing'" class="size-6 animate-spin" aria-hidden="true" />
            <Phone v-else class="size-6" aria-hidden="true" />
          </span>
          <div class="min-w-0 flex-1">
            <h2 class="m-0 text-base font-extrabold text-text-primary">
              {{ statusTitle }}
            </h2>
            <p v-if="phoneStatus?.maskedPhone" class="m-0 mt-1 font-mono text-sm font-semibold text-text-secondary">
              {{ phoneStatus.maskedPhone }}
            </p>
            <p class="m-0 mt-2 text-sm leading-6 text-text-muted">{{ statusDescription }}</p>
            <dl v-if="phoneStatus?.state === 'verified'" class="m-0 mt-4 grid gap-2 border-t border-border pt-4 text-xs sm:grid-cols-2">
              <div>
                <dt class="text-text-muted">{{ t('user.verification.phone.platform.status.method') }}</dt>
                <dd class="m-0 mt-1 font-semibold text-text-primary">{{ verificationMethodLabel }}</dd>
              </div>
              <div>
                <dt class="text-text-muted">{{ t('user.verification.phone.platform.status.verifiedAt') }}</dt>
                <dd class="m-0 mt-1 font-semibold text-text-primary">{{ formatDateTime(phoneStatus.verifiedAt) }}</dd>
              </div>
            </dl>
          </div>
        </div>
      </section>

      <section v-if="!showEditor && phoneStatus?.state === 'verified'" class="rounded-2xl border border-border bg-bg-card p-5 shadow-card sm:p-6">
        <h2 class="m-0 text-base font-extrabold text-text-primary">
          {{ t('user.verification.phone.platform.manage.title') }}
        </h2>
        <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
          {{ t('user.verification.phone.platform.manage.description') }}
        </p>
        <div class="mt-5 flex flex-col gap-2 sm:flex-row">
          <button class="phone-primary sm:w-auto" type="button" @click="beginChange">
            {{ t('user.verification.phone.platform.manage.change') }}
          </button>
          <button class="phone-danger" type="button" :disabled="busy" @click="unbindPhone">
            {{ t('user.verification.phone.platform.manage.unbind') }}
          </button>
        </div>
      </section>

      <section v-else-if="phoneStatus?.state !== 'syncing'" class="rounded-2xl border border-border bg-bg-card p-5 shadow-card sm:p-6">
        <template v-if="!operation || operation.verificationStep === 'sms_otp'">
          <div v-if="!operation">
            <h2 class="m-0 text-base font-extrabold text-text-primary">
              {{ changeMode
                ? t('user.verification.phone.platform.form.changeTitle')
                : t('user.verification.phone.platform.form.bindTitle') }}
            </h2>
            <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
              {{ t('user.verification.phone.platform.form.description') }}
            </p>
            <form class="mt-5 grid gap-5" data-phone-operation-form @submit.prevent="createOperation">
              <label class="phone-field" for="account-phone-number">
                <span>{{ t('user.verification.phone.platform.form.phone') }}</span>
                <span class="relative">
                  <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-semibold text-text-secondary">+86</span>
                  <input
                    id="account-phone-number"
                    v-model.trim="phone"
                    data-phone-number
                    class="phone-control pl-14"
                    name="phone_number"
                    type="tel"
                    inputmode="numeric"
                    autocomplete="tel"
                    spellcheck="false"
                    maxlength="11"
                    placeholder="138 0000 0000"
                    required
                  />
                </span>
              </label>

              <div class="rounded-xl border border-primary/20 bg-primary/5 p-4 text-xs leading-5 text-text-secondary">
                <div class="flex gap-3">
                  <ShieldCheck class="mt-0.5 size-4 shrink-0 text-primary" aria-hidden="true" />
                  <p class="m-0">{{ t('user.verification.phone.platform.form.authority') }}</p>
                </div>
              </div>

              <p v-if="errorMessage" class="m-0 rounded-lg bg-danger/10 p-3 text-sm text-danger" role="alert">
                {{ errorMessage }}
              </p>

              <button class="phone-primary" type="submit" :disabled="!validPhone || busy">
                <LoaderCircle v-if="busy" class="size-4 animate-spin" aria-hidden="true" />
                {{ t('user.verification.phone.platform.form.continue') }}
              </button>
              <button v-if="changeMode" class="phone-secondary" type="button" :disabled="busy" @click="cancelChange">
                {{ t('common.actions.cancel') }}
              </button>
            </form>
          </div>

          <div v-else data-phone-sms-step>
            <button class="mb-3 min-h-11 border-0 bg-transparent p-0 text-sm font-semibold text-primary hover:underline" type="button" @click="resetOperation">
              ← {{ t('user.verification.phone.platform.sms.changeNumber') }}
            </button>
            <h2 class="m-0 text-base font-extrabold text-text-primary">
              {{ t('user.verification.phone.platform.sms.title') }}
            </h2>
            <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
              {{ t('user.verification.phone.platform.sms.sent', { phone: operation.maskedPhone ?? '' }) }}
            </p>
            <form class="mt-5 grid gap-5" @submit.prevent="verifySMS">
              <div class="phone-field">
                <span>{{ t('user.verification.phone.platform.sms.code') }}</span>
                <OtpCodeInput v-model="smsCode" :disabled="busy" :aria-label="t('user.verification.phone.platform.sms.code')" />
              </div>
              <div class="flex items-center justify-between gap-3">
                <button
                  class="min-h-11 border-0 bg-transparent px-2 text-sm font-semibold text-primary disabled:text-text-muted"
                  type="button"
                  :disabled="busy || resendSeconds > 0"
                  @click="sendSMS"
                >
                  {{ resendSeconds > 0
                    ? t('user.verification.phone.platform.sms.resendIn', { seconds: resendSeconds })
                    : t('user.verification.phone.platform.sms.resend') }}
                </button>
              </div>
              <p v-if="errorMessage" class="m-0 rounded-lg bg-danger/10 p-3 text-sm text-danger" role="alert">
                {{ errorMessage }}
              </p>
              <button class="phone-primary" type="submit" :disabled="smsCode.length !== 6 || busy">
                <LoaderCircle v-if="busy" class="size-4 animate-spin" aria-hidden="true" />
                {{ t('user.verification.phone.platform.sms.verify') }}
              </button>
            </form>
          </div>
        </template>

        <div v-else class="py-4 text-center" role="status">
          <LoaderCircle class="mx-auto size-8 animate-spin text-primary" aria-hidden="true" />
          <h2 class="m-0 mt-4 text-base font-extrabold text-text-primary">
            {{ t('user.verification.phone.platform.syncing.title') }}
          </h2>
          <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
            {{ t('user.verification.phone.platform.syncing.description') }}
          </p>
        </div>
      </section>

      <section class="rounded-2xl border border-border bg-bg-elevated/70 p-5 text-sm leading-6 text-text-secondary">
        <div class="flex gap-3">
          <Info class="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
          <div>
            <h2 class="m-0 text-sm font-extrabold text-text-primary">
              {{ t('user.verification.phone.platform.why.title') }}
            </h2>
            <p class="m-0 mt-1">{{ t('user.verification.phone.platform.why.description') }}</p>
            <p class="m-0 mt-2 font-semibold text-text-primary">
              {{ t('user.verification.phone.platform.why.notStudentEvidence') }}
            </p>
          </div>
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  BadgeCheck,
  Info,
  LoaderCircle,
  Phone,
  RefreshCw,
  ShieldCheck,
} from 'lucide-vue-next'
import type { ApiCallResult, PhoneBindingOperation, PhoneStatus } from '@stuhelper/shared/api'
import { extractResultData } from '@stuhelper/shared/api'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import {
  isCrossOriginAccountFlowRedirect,
  readAccountFlowRedirect,
} from '@/utils/internalRedirect'
import OtpCodeInput from '@/components/common/OtpCodeInput.vue'
import { useToast } from '@/composables/useToast'

const PHONE_PATTERN = /^1[3-9]\d{9}$/
const OPERATION_POLL_INTERVAL_MS = 1500

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()

const loading = ref(true)
const busy = ref(false)
const phoneStatus = ref<PhoneStatus | null>(null)
const operation = ref<PhoneBindingOperation | null>(null)
const phone = ref('')
const smsCode = ref('')
const changeMode = ref(false)
const showEditor = ref(false)
const errorMessage = ref('')
const resendSeconds = ref(0)
let pollTimer: number | undefined
let countdownTimer: number | undefined

const validPhone = computed(() => PHONE_PATTERN.test(phone.value))
const statusTitle = computed(() => {
  if (phoneStatus.value?.state === 'verified') return t('user.verification.phone.platform.status.verified')
  if (phoneStatus.value?.state === 'syncing') return t('user.verification.phone.platform.status.syncing')
  if (phoneStatus.value?.state === 'review_required') return t('user.verification.phone.platform.status.reviewRequired')
  return t('user.verification.phone.platform.status.unbound')
})
const statusDescription = computed(() => {
  if (phoneStatus.value?.state === 'verified') return t('user.verification.phone.platform.status.verifiedDescription')
  if (phoneStatus.value?.state === 'syncing') return t('user.verification.phone.platform.status.syncingDescription')
  if (phoneStatus.value?.state === 'review_required') return t('user.verification.phone.platform.status.reviewDescription')
  return t('user.verification.phone.platform.status.unboundDescription')
})
const verificationMethodLabel = computed(() => {
  if (phoneStatus.value?.method === 'school_roster_phone_match') return t('user.verification.phone.platform.status.schoolConfirmed')
  if (phoneStatus.value?.method === 'sms_possession') return t('user.verification.phone.platform.status.smsConfirmed')
  return t('user.verification.phone.platform.status.verifiedMethod')
})

async function readPayload<T>(request: Promise<unknown>, fallback: string): Promise<T> {
  const result = await request as ApiCallResult<T>
  const data = extractResultData(result)
  if (data === undefined) throw new Error(fallback)
  return data
}

async function loadStatus(): Promise<void> {
  loading.value = true
  errorMessage.value = ''
  try {
    phoneStatus.value = await readPayload<PhoneStatus>(api.studentVerification.getPhoneStatus(), 'Unable to load phone status')
    showEditor.value = phoneStatus.value.state !== 'verified'
    if (phoneStatus.value.state === 'syncing') scheduleStatusPoll()
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.phone.platform.errors.load'))
    phoneStatus.value = { state: 'unbound', publishingRequirementSatisfied: false, revision: 1 }
    showEditor.value = true
  } finally {
    loading.value = false
  }
}

async function createOperation(): Promise<void> {
  if (!validPhone.value || busy.value) return
  busy.value = true
  errorMessage.value = ''
  try {
    operation.value = await readPayload<PhoneBindingOperation>(
      changeMode.value
        ? api.studentVerification.createPhoneChangeOperation({ phone: phone.value })
        : api.studentVerification.createPhoneOperation({ phone: phone.value }),
      'Unable to create phone operation',
    )
    phone.value = ''
    if (operation.value.status === 'completed') {
      await completeFlow()
      return
    }
    if (operation.value.verificationStep === 'sms_otp') {
      await sendSMS()
      return
    }
    scheduleOperationPoll()
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.phone.platform.errors.create'))
  } finally {
    busy.value = false
  }
}

async function sendSMS(): Promise<void> {
  if (!operation.value) return
  const wasBusy = busy.value
  busy.value = true
  errorMessage.value = ''
  try {
    operation.value = await readPayload<PhoneBindingOperation>(
      api.studentVerification.sendPhoneSMS(operation.value.id),
      'Unable to send phone code',
    )
    startResendCountdown(operation.value.smsResendAvailableAt)
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.phone.platform.errors.sms'))
  } finally {
    if (!wasBusy) busy.value = false
  }
}

async function verifySMS(): Promise<void> {
  if (!operation.value || smsCode.value.length !== 6 || busy.value) return
  busy.value = true
  errorMessage.value = ''
  try {
    operation.value = await readPayload<PhoneBindingOperation>(
      api.studentVerification.verifyPhoneSMS(operation.value.id, { code: smsCode.value }),
      'Unable to verify phone code',
    )
    smsCode.value = ''
    if (operation.value.status === 'completed') await completeFlow()
    else scheduleOperationPoll()
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.phone.platform.errors.invalidCode'))
  } finally {
    busy.value = false
  }
}

function scheduleOperationPoll(): void {
  if (!operation.value || operation.value.status === 'completed' || operation.value.status === 'failed') return
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
  pollTimer = window.setTimeout(() => void pollOperation(), OPERATION_POLL_INTERVAL_MS)
}

async function pollOperation(): Promise<void> {
  if (!operation.value) return
  try {
    operation.value = await readPayload<PhoneBindingOperation>(
      api.studentVerification.getPhoneOperation(operation.value.id),
      'Unable to refresh phone operation',
    )
    if (operation.value.status === 'completed') {
      await completeFlow()
      return
    }
    if (operation.value.status === 'failed' || operation.value.status === 'expired') {
      errorMessage.value = t('user.verification.phone.platform.errors.sync')
      return
    }
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.phone.platform.errors.sync'))
  }
  scheduleOperationPoll()
}

function scheduleStatusPoll(): void {
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
  pollTimer = window.setTimeout(() => void refreshStatusUntilSettled(), OPERATION_POLL_INTERVAL_MS)
}

async function refreshStatusUntilSettled(): Promise<void> {
  try {
    phoneStatus.value = await readPayload<PhoneStatus>(api.studentVerification.getPhoneStatus(), 'Unable to refresh phone status')
  } catch {
    scheduleStatusPoll()
    return
  }
  if (phoneStatus.value.state === 'syncing') scheduleStatusPoll()
}

async function completeFlow(): Promise<void> {
  phoneStatus.value = await readPayload<PhoneStatus>(api.studentVerification.getPhoneStatus(), 'Unable to refresh phone status')
  operation.value = null
  showEditor.value = false
  changeMode.value = false
  toast.success(t('user.verification.phone.platform.success'))
  if (route.query.redirect) {
    const target = readAccountFlowRedirect(route.query.redirect)
    if (isCrossOriginAccountFlowRedirect(target)) {
      window.location.assign(target)
      return
    }
    await router.push(target)
  }
}

function beginChange(): void {
  changeMode.value = true
  showEditor.value = true
  operation.value = null
  errorMessage.value = ''
}

function cancelChange(): void {
  changeMode.value = false
  showEditor.value = false
  phone.value = ''
  errorMessage.value = ''
}

function resetOperation(): void {
  operation.value = null
  smsCode.value = ''
  errorMessage.value = ''
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
}

async function unbindPhone(): Promise<void> {
  if (busy.value || !window.confirm(t('user.verification.phone.platform.manage.unbindConfirm'))) return
  busy.value = true
  try {
    operation.value = await readPayload<PhoneBindingOperation>(api.studentVerification.unbindPhone(), 'Unable to unbind phone')
    if (operation.value.status === 'completed') await loadStatus()
    else scheduleOperationPoll()
  } catch (error) {
    toast.error(getErrorMessage(error, t('user.verification.phone.platform.errors.unbind')))
  } finally {
    busy.value = false
  }
}

function startResendCountdown(resendAt?: string | null): void {
  if (countdownTimer !== undefined) window.clearInterval(countdownTimer)
  const update = () => {
    const remaining = resendAt ? Math.ceil((new Date(resendAt).getTime() - Date.now()) / 1000) : 60
    resendSeconds.value = Math.max(0, remaining)
    if (resendSeconds.value === 0 && countdownTimer !== undefined) {
      window.clearInterval(countdownTimer)
      countdownTimer = undefined
    }
  }
  update()
  countdownTimer = window.setInterval(update, 1000)
}

function formatDateTime(value?: string | null): string {
  return value ? new Date(value).toLocaleString() : '—'
}

function goBack(): void {
  void router.push('/identity')
}

onMounted(() => void loadStatus())
onBeforeUnmount(() => {
  smsCode.value = ''
  phone.value = ''
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
  if (countdownTimer !== undefined) window.clearInterval(countdownTimer)
})
</script>

<style scoped>
@reference "../../../styles/tailwind.css";

.phone-field {
  @apply grid gap-2 text-sm font-semibold text-text-primary;
}

.phone-control {
  @apply min-h-11 w-full rounded-lg border border-border bg-bg-card px-3 py-2.5 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60;
}

.phone-primary {
  @apply inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border-0 bg-primary px-5 py-2.5 text-sm font-bold text-white transition-colors hover:bg-primary-dark focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-bg-card disabled:cursor-not-allowed disabled:opacity-50;
}

.phone-secondary {
  @apply inline-flex min-h-11 items-center justify-center rounded-lg border border-border bg-bg-card px-4 py-2.5 text-sm font-semibold text-text-primary transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50;
}

.phone-danger {
  @apply inline-flex min-h-11 items-center justify-center rounded-lg border border-danger/30 bg-bg-card px-4 py-2.5 text-sm font-semibold text-danger transition-colors hover:bg-danger/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger/30 disabled:cursor-not-allowed disabled:opacity-50;
}
</style>
