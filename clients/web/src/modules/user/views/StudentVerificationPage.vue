<template>
  <main class="mx-auto w-full max-w-6xl px-4 py-6 sm:px-6 lg:py-10" data-student-verification-page>
    <header class="mb-6 flex items-start gap-3 sm:mb-8">
      <button
        type="button"
        class="grid size-11 shrink-0 place-items-center rounded-lg border border-border bg-bg-card text-text-muted transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
        :aria-label="t('common.actions.back')"
        @click="goBack"
      >
        <ArrowLeft class="size-5" aria-hidden="true" />
      </button>
      <div class="min-w-0">
        <p class="m-0 text-xs font-bold uppercase tracking-[0.18em] text-primary">
          {{ t('user.verification.student.platform.eyebrow') }}
        </p>
        <h1 class="m-0 mt-1 text-2xl font-extrabold tracking-tight text-text-primary sm:text-3xl">
          {{ t('user.verification.student.platform.title') }}
        </h1>
        <p class="m-0 mt-2 max-w-2xl text-sm leading-6 text-text-muted sm:text-base">
          {{ t('user.verification.student.platform.subtitle') }}
        </p>
      </div>
    </header>

    <div v-if="loading" class="grid min-h-72 place-items-center rounded-2xl border border-border bg-bg-card shadow-card">
      <div class="text-center" role="status" aria-busy="true">
        <LoaderCircle class="mx-auto size-7 animate-spin text-primary" aria-hidden="true" />
        <p class="m-0 mt-3 text-sm text-text-muted">{{ t('common.actions.loading') }}</p>
      </div>
    </div>

    <section v-else-if="loadError" class="rounded-2xl border border-danger/30 bg-bg-card p-6 shadow-card" role="alert">
      <CircleAlert class="size-6 text-danger" aria-hidden="true" />
      <h2 class="m-0 mt-3 text-lg font-bold text-text-primary">
        {{ t('user.verification.student.platform.errors.loadTitle') }}
      </h2>
      <p class="m-0 mt-2 text-sm leading-6 text-text-muted">{{ loadError }}</p>
      <button class="sv-primary mt-5 w-auto" type="button" @click="loadPlatform">
        <RefreshCw class="size-4" aria-hidden="true" />
        {{ t('common.actions.retry') }}
      </button>
    </section>

    <div v-else class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_280px] lg:items-start">
      <div class="grid gap-6">
        <section
          v-if="activeCredentials.length > 0"
          class="overflow-hidden rounded-2xl border border-success/25 bg-bg-card shadow-card"
          data-active-credentials
        >
          <div class="flex items-start gap-3 border-b border-border bg-success/5 p-5 sm:p-6">
            <span class="grid size-11 shrink-0 place-items-center rounded-xl bg-success/15 text-success">
              <BadgeCheck class="size-6" aria-hidden="true" />
            </span>
            <div>
              <h2 class="m-0 text-base font-extrabold text-text-primary">
                {{ t('user.verification.student.platform.credentials.activeTitle') }}
              </h2>
              <p class="m-0 mt-1 text-sm leading-6 text-text-muted">
                {{ t('user.verification.student.platform.credentials.activeDescription') }}
              </p>
            </div>
          </div>
          <ul class="m-0 divide-y divide-border p-0">
            <li v-for="credential in activeCredentials" :key="credential.id" class="flex items-start justify-between gap-4 p-5 sm:px-6">
              <div class="min-w-0">
                <p class="m-0 font-bold text-text-primary">{{ credential.schoolName }}</p>
                <p class="m-0 mt-1 text-sm text-text-muted">
                  {{ methodLabel(credential.method) }} · {{ credential.subjectDisplay }}
                </p>
                <p class="m-0 mt-2 text-xs text-text-muted">
                  {{ credentialExpiry(credential.expiresAt) }}
                </p>
              </div>
              <button
                type="button"
                class="min-h-11 shrink-0 rounded-lg border border-border bg-transparent px-3 text-xs font-semibold text-text-muted transition-colors hover:border-danger/50 hover:text-danger focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger/30"
                @click="revokeCredential(credential.id)"
              >
                {{ t('user.verification.student.platform.credentials.revoke') }}
              </button>
            </li>
          </ul>
        </section>

        <section class="rounded-2xl border border-border bg-bg-card p-5 shadow-card sm:p-7">
          <template v-if="!selectedSchool">
            <div class="mb-5">
              <span class="sv-step-number">1</span>
              <h2 class="m-0 mt-3 text-xl font-extrabold text-text-primary">
                {{ t('user.verification.student.platform.school.title') }}
              </h2>
              <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
                {{ t('user.verification.student.platform.school.description') }}
              </p>
            </div>

            <label class="sv-field" for="verification-school-search">
              <span>{{ t('user.verification.student.platform.school.search') }}</span>
              <span class="relative">
                <Search class="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-text-muted" aria-hidden="true" />
                <input
                  id="verification-school-search"
                  v-model.trim="schoolSearch"
                  class="sv-control pl-10"
                  name="school_search"
                  type="search"
                  autocomplete="off"
                  spellcheck="false"
                  :placeholder="t('user.verification.student.platform.school.placeholder')"
                />
              </span>
            </label>

            <ul v-if="filteredSchools.length" class="m-0 mt-4 grid gap-2 p-0" :aria-label="t('user.verification.student.platform.school.title')">
              <li v-for="school in filteredSchools" :key="school.code">
                <button
                  type="button"
                  data-verification-school-option
                  class="flex min-h-16 w-full items-center justify-between gap-4 rounded-xl border border-border bg-bg-card px-4 py-3 text-left transition-colors hover:border-primary hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  @click="selectSchool(school.code)"
                >
                  <span>
                    <span class="block text-sm font-bold text-text-primary">{{ school.name }}</span>
                    <span class="mt-1 block text-xs text-text-muted">{{ school.location || school.code }}</span>
                  </span>
                  <ChevronRight class="size-4 shrink-0 text-text-muted" aria-hidden="true" />
                </button>
              </li>
            </ul>

            <div v-else class="mt-4 rounded-xl border border-dashed border-border p-5 text-center">
              <p class="m-0 text-sm font-semibold text-text-primary">
                {{ t('user.verification.student.platform.school.notFound') }}
              </p>
              <button class="sv-secondary mt-3" type="button" @click="showSuggestion = true">
                {{ t('user.verification.student.platform.school.suggest') }}
              </button>
            </div>

            <form v-if="showSuggestion" class="mt-5 grid gap-3 rounded-xl bg-bg-elevated p-4" @submit.prevent="submitSchoolSuggestion">
              <label class="sv-field">
                <span>{{ t('user.verification.student.platform.school.schoolName') }}</span>
                <input v-model.trim="suggestion.name" class="sv-control" name="suggested_school_name" type="text" autocomplete="organization" required maxlength="100" />
              </label>
              <label class="sv-field">
                <span>{{ t('user.verification.student.platform.school.location') }}</span>
                <input v-model.trim="suggestion.location" class="sv-control" name="suggested_school_location" type="text" autocomplete="address-level2" maxlength="100" />
              </label>
              <button class="sv-primary" type="submit" :disabled="suggestionBusy || suggestion.name.length < 2">
                {{ t('user.verification.student.platform.school.submitSuggestion') }}
              </button>
            </form>
          </template>

          <template v-else-if="!selectedMethod && !applicationComplete">
            <div class="flex items-start justify-between gap-4">
              <div>
                <span class="sv-step-number">2</span>
                <h2 class="m-0 mt-3 text-xl font-extrabold text-text-primary">
                  {{ t('user.verification.student.platform.method.title') }}
                </h2>
                <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
                  {{ selectedSchool.name }} · {{ t('user.verification.student.platform.method.description') }}
                </p>
              </div>
              <button
                v-if="!application"
                type="button"
                class="min-h-11 shrink-0 rounded-lg border-0 bg-transparent px-2 text-sm font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                @click="clearSchool"
              >
                {{ t('user.verification.student.platform.school.change') }}
              </button>
            </div>

            <div class="mt-6 grid gap-3 sm:grid-cols-2">
              <button
                v-for="method in selectedSchool.methods"
                :key="method.method"
                type="button"
                :data-verification-method="method.method"
                class="group min-h-36 rounded-xl border p-4 text-left transition-[transform,border-color,box-shadow,background-color] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                :class="method.availability === 'available' && method.privacyNotice
                  ? 'border-border bg-bg-card hover:-translate-y-0.5 hover:border-primary hover:shadow-sm'
                  : 'cursor-not-allowed border-border bg-bg-elevated opacity-60'"
                :disabled="method.availability !== 'available' || !method.privacyNotice || methodBusy"
                @click="chooseMethod(method)"
              >
                <span class="flex items-start justify-between gap-3">
                  <span class="grid size-10 place-items-center rounded-lg bg-primary/10 text-primary">
                    <Fingerprint v-if="method.method === 'real_name_identity_check'" class="size-5" aria-hidden="true" />
                    <KeyRound v-else-if="method.method === 'school_sso'" class="size-5" aria-hidden="true" />
                    <MailCheck v-else-if="method.method.includes('email')" class="size-5" aria-hidden="true" />
                    <ClipboardCheck v-else class="size-5" aria-hidden="true" />
                  </span>
                  <span class="rounded-full px-2 py-1 text-[11px] font-bold" :class="method.availability === 'available' ? 'bg-success/10 text-success' : 'bg-bg-secondary text-text-muted'">
                    {{ method.availability === 'available'
                      ? t('user.verification.student.platform.method.available')
                      : t('user.verification.student.platform.method.unavailable') }}
                  </span>
                </span>
                <span class="mt-3 block text-sm font-extrabold text-text-primary">{{ methodLabel(method.method) }}</span>
                <span class="mt-1 block text-xs leading-5 text-text-muted">{{ method.description }}</span>
              </button>
            </div>
          </template>

          <template v-else-if="selectedMethod && application && !applicationComplete">
            <div class="flex items-start justify-between gap-4 border-b border-border pb-5">
              <div>
                <button
                  type="button"
                  class="mb-2 min-h-11 border-0 bg-transparent p-0 text-sm font-semibold text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                  @click="returnToMethods"
                >
                  ← {{ t('user.verification.student.platform.method.change') }}
                </button>
                <h2 class="m-0 text-xl font-extrabold text-text-primary">{{ methodLabel(selectedMethod.method) }}</h2>
                <p class="m-0 mt-2 text-sm leading-6 text-text-muted">{{ methodExplanation(selectedMethod.method) }}</p>
              </div>
              <span class="rounded-full bg-primary/10 px-3 py-1.5 text-xs font-bold text-primary">3 / 3</span>
            </div>

            <ManualReviewEvidence
              v-if="selectedMethod.method === 'manual_material_review'"
              class="mt-6"
              :application-id="application.id"
              :method="selectedMethod"
              @updated="refreshApplication"
              @submitted="handleManualSubmitted"
            />

            <form v-else data-verification-method-form class="mt-6 grid gap-5" :aria-busy="methodBusy" @submit.prevent="submitMethod">
              <div class="grid gap-4 sm:grid-cols-2">
                <label class="sv-field">
                  <span>{{ t('user.verification.student.platform.fields.studentID') }}</span>
                  <input
                    v-model.trim="form.studentID"
                    data-verification-student-id
                    class="sv-control"
                    name="student_id"
                    type="text"
                    autocomplete="username"
                    autocapitalize="characters"
                    spellcheck="false"
                    maxlength="64"
                    required
                  />
                </label>
                <label v-if="methodNeedsName" class="sv-field">
                  <span>{{ t('user.verification.student.platform.fields.name') }}</span>
                  <input v-model.trim="form.name" data-verification-name class="sv-control" name="full_name" type="text" autocomplete="name" maxlength="100" required />
                </label>
                <label v-if="selectedMethod.method === 'real_name_identity_check'" class="sv-field sm:col-span-2">
                  <span>{{ t('user.verification.student.platform.fields.documentNumber') }}</span>
                  <span class="font-normal text-text-muted">{{ t('user.verification.student.platform.fields.documentHint') }}</span>
                  <input v-model.trim="form.documentNumber" data-verification-document-number class="sv-control" name="document_number" type="text" inputmode="text" autocomplete="off" autocapitalize="characters" spellcheck="false" maxlength="18" required />
                </label>
                <label v-if="selectedMethod.method === 'school_sso'" class="sv-field sm:col-span-2">
                  <span>{{ t('user.verification.student.platform.fields.password') }}</span>
                  <span class="font-normal text-text-muted">{{ t('user.verification.student.platform.fields.passwordHint') }}</span>
                  <input v-model="form.password" data-verification-password class="sv-control" name="password" type="password" autocomplete="current-password" spellcheck="false" maxlength="256" required />
                </label>
              </div>

              <PrivacyConsent
                v-if="selectedMethod.privacyNotice && !emailChallenge && !inboundChallenge"
                v-model="form.consented"
                :notice="selectedMethod.privacyNotice"
              />

              <section v-if="emailChallenge" class="grid gap-4 rounded-xl border border-primary/25 bg-primary/5 p-4">
                <div>
                  <p class="m-0 text-sm font-bold text-text-primary">
                    {{ t('user.verification.student.platform.email.sent', { email: emailChallenge.maskedEmail }) }}
                  </p>
                  <p class="m-0 mt-1 text-xs leading-5 text-text-muted">
                    {{ t('user.verification.student.platform.email.expiry', { time: formatTime(emailChallenge.expiresAt) }) }}
                  </p>
                </div>
                <OtpCodeInput
                  v-model="form.emailCode"
                  :disabled="methodBusy"
                  :aria-label="t('user.verification.student.platform.email.code')"
                />
              </section>

              <section v-if="inboundChallenge" class="rounded-xl border border-primary/25 bg-primary/5 p-4">
                <div class="flex items-start gap-3">
                  <Send class="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
                  <div class="min-w-0">
                    <h3 class="m-0 text-sm font-bold text-text-primary">{{ t('user.verification.student.platform.inbound.title') }}</h3>
                    <ol class="mb-0 mt-3 list-decimal space-y-2 pl-5 text-sm leading-6 text-text-secondary">
                      <li>{{ t('user.verification.student.platform.inbound.from', { email: inboundChallenge.expectedSenderMasked }) }}</li>
                      <li>{{ t('user.verification.student.platform.inbound.to', { email: inboundChallenge.targetAddress }) }}</li>
                      <li>{{ t('user.verification.student.platform.inbound.subject') }} <code class="select-all break-all rounded bg-bg-card px-1.5 py-1 text-xs text-text-primary">{{ inboundChallenge.subject }}</code></li>
                      <li v-if="inboundChallenge.challengeValue">{{ t('user.verification.student.platform.inbound.body') }} <code class="select-all break-all rounded bg-bg-card px-1.5 py-1 text-xs text-text-primary">{{ inboundChallenge.challengeValue }}</code></li>
                    </ol>
                    <p class="m-0 mt-3 text-xs text-text-muted">{{ t('user.verification.student.platform.inbound.waiting') }}</p>
                  </div>
                </div>
              </section>

              <p v-if="methodError" class="m-0 rounded-lg bg-danger/10 p-3 text-sm leading-6 text-danger" role="alert">
                {{ methodError }}
              </p>

              <button v-if="!inboundChallenge" data-verification-submit class="sv-primary" type="submit" :disabled="!canSubmitMethod || methodBusy">
                <LoaderCircle v-if="methodBusy" class="size-4 animate-spin" aria-hidden="true" />
                {{ submitLabel }}
              </button>

              <button
                v-if="methodError && manualMethodAvailable"
                class="sv-secondary"
                type="button"
                @click="switchToManualReview"
              >
                {{ t('user.verification.student.platform.recovery.manual') }}
              </button>
            </form>
          </template>

          <template v-else-if="applicationComplete">
            <div class="py-4 text-center" data-verification-complete>
              <span class="mx-auto grid size-16 place-items-center rounded-2xl" :class="application?.status === 'approved' ? 'bg-success/15 text-success' : 'bg-warning/15 text-warning'">
                <BadgeCheck v-if="application?.status === 'approved'" class="size-8" aria-hidden="true" />
                <Clock3 v-else class="size-8" aria-hidden="true" />
              </span>
              <h2 class="m-0 mt-5 text-xl font-extrabold text-text-primary">
                {{ application?.status === 'approved'
                  ? t('user.verification.student.platform.result.approved')
                  : t('user.verification.student.platform.result.pending') }}
              </h2>
              <p class="mx-auto mb-0 mt-2 max-w-md text-sm leading-6 text-text-muted">
                {{ application?.status === 'approved'
                  ? t('user.verification.student.platform.result.approvedDescription')
                  : t('user.verification.student.platform.result.pendingDescription') }}
              </p>
              <div class="mt-6 flex flex-col justify-center gap-2 sm:flex-row">
                <button class="sv-primary w-auto" type="button" @click="finish">
                  {{ t('user.verification.student.platform.result.done') }}
                </button>
                <button v-if="application?.status === 'approved'" class="sv-secondary" type="button" @click="startAnother">
                  {{ t('user.verification.student.platform.result.addAnother') }}
                </button>
              </div>
            </div>
          </template>
        </section>

        <button
          v-if="application && !applicationComplete"
          type="button"
          data-verification-cancel
          class="min-h-11 justify-self-start border-0 bg-transparent px-2 text-sm font-semibold text-text-muted hover:text-danger hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-danger/30"
          :disabled="methodBusy"
          @click="cancelApplication"
        >
          {{ t('user.verification.student.platform.cancel') }}
        </button>
      </div>

      <aside class="rounded-2xl border border-border bg-bg-card p-5 shadow-card lg:sticky lg:top-24">
        <h2 class="m-0 text-sm font-extrabold text-text-primary">
          {{ t('user.verification.student.platform.trust.title') }}
        </h2>
        <ol class="m-0 mt-5 grid gap-0 p-0">
          <li v-for="(step, index) in trustSteps" :key="step.title" class="relative flex gap-3 pb-6 last:pb-0">
            <span v-if="index < trustSteps.length - 1" class="absolute left-[15px] top-8 h-[calc(100%-2rem)] w-px bg-border" aria-hidden="true" />
            <span class="relative z-10 grid size-8 shrink-0 place-items-center rounded-full border text-xs font-extrabold" :class="step.state === 'done' ? 'border-success bg-success text-white' : step.state === 'active' ? 'border-primary bg-primary text-white' : 'border-border bg-bg-elevated text-text-muted'">
              <Check v-if="step.state === 'done'" class="size-4" aria-hidden="true" />
              <span v-else>{{ index + 1 }}</span>
            </span>
            <span class="pt-1">
              <span class="block text-sm font-bold text-text-primary">{{ step.title }}</span>
              <span class="mt-1 block text-xs leading-5 text-text-muted">{{ step.description }}</span>
            </span>
          </li>
        </ol>
        <div class="mt-5 border-t border-border pt-5">
          <div class="flex gap-3 text-xs leading-5 text-text-muted">
            <ShieldCheck class="mt-0.5 size-4 shrink-0 text-primary" aria-hidden="true" />
            <p class="m-0">{{ t('user.verification.student.platform.trust.separation') }}</p>
          </div>
          <RouterLink class="mt-4 inline-flex min-h-11 items-center text-xs font-bold text-primary hover:underline" to="/user/phone-binding">
            {{ t('user.verification.student.platform.trust.phoneLink') }}
            <ChevronRight class="ml-1 size-3.5" aria-hidden="true" />
          </RouterLink>
        </div>
      </aside>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  BadgeCheck,
  Check,
  ChevronRight,
  CircleAlert,
  ClipboardCheck,
  Fingerprint,
  KeyRound,
  LoaderCircle,
  MailCheck,
  RefreshCw,
  Search,
  Send,
  ShieldCheck,
  Clock3,
} from 'lucide-vue-next'
import type {
  ApiCallResult,
  InboundEmailChallenge,
  StudentEmailOTPChallenge,
  StudentVerificationCredential,
  VerificationApplication,
  VerificationSchool,
} from '@stuhelper/shared/api'
import { extractResultData } from '@stuhelper/shared/api'

import { api } from '@/api'
import {
  isCrossOriginAccountFlowRedirect,
  readAccountFlowRedirect,
} from '@/utils/internalRedirect'
import { getErrorMessage } from '@/api/errors'
import OtpCodeInput from '@/components/common/OtpCodeInput.vue'
import { useToast } from '@/composables/useToast'
import ManualReviewEvidence from '@/modules/student-verification/components/ManualReviewEvidence.vue'
import PrivacyConsent from '@/modules/student-verification/components/PrivacyConsent.vue'

type MethodCapability = VerificationSchool['methods'][number]
type Method = MethodCapability['method'] | StudentVerificationCredential['method']
type TrustStepState = 'pending' | 'active' | 'done'

const VERIFICATION_METHODS = new Set<MethodCapability['method']>([
  'real_name_identity_check',
  'school_sso',
  'student_email_outbound_otp',
  'student_email_inbound_challenge',
  'manual_material_review',
])

const ACTIVE_APPLICATION_STORAGE_KEY = 'stuhelper.studentVerification.activeApplication.v1'
const INBOUND_POLL_INTERVAL_MS = 3000

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()

const loading = ref(true)
const loadError = ref('')
const schools = ref<VerificationSchool[]>([])
const credentials = ref<StudentVerificationCredential[]>([])
const selectedSchoolCode = ref('')
const selectedMethod = ref<MethodCapability | null>(null)
const application = ref<VerificationApplication | null>(null)
const eligibility = ref<boolean | null>(null)
const schoolSearch = ref('')
const showSuggestion = ref(false)
const suggestionBusy = ref(false)
const methodBusy = ref(false)
const methodError = ref('')
const emailChallenge = ref<StudentEmailOTPChallenge | null>(null)
const inboundChallenge = ref<InboundEmailChallenge | null>(null)
const suggestion = reactive({ name: '', location: '' })
const form = reactive({
  studentID: '',
  name: '',
  documentNumber: '',
  password: '',
  emailCode: '',
  consented: false,
})
let inboundPollTimer: number | undefined

const selectedSchool = computed(() => schools.value.find((school) => school.code === selectedSchoolCode.value) ?? null)
const filteredSchools = computed(() => {
  const query = schoolSearch.value.toLocaleLowerCase()
  if (!query) return schools.value
  return schools.value.filter((school) => `${school.name} ${school.location ?? ''} ${school.code}`.toLocaleLowerCase().includes(query))
})
const activeCredentials = computed(() => credentials.value.filter((credential) => credential.status === 'active' || credential.status === 'review_required'))
const manualMethodAvailable = computed(() => selectedSchool.value?.methods.some((method) => method.method === 'manual_material_review' && method.availability === 'available' && method.privacyNotice) ?? false)
const methodNeedsName = computed(() => selectedMethod.value?.method !== 'school_sso')
const applicationComplete = computed(() => application.value?.status === 'approved' || application.value?.status === 'pending_manual_review')
const canSubmitMethod = computed(() => {
  const method = selectedMethod.value
  if (!method || !method.privacyNotice) return false
  if (emailChallenge.value) return /^\d{6}$/.test(form.emailCode)
  if (!form.consented || !form.studentID) return false
  if (method.method === 'real_name_identity_check') return Boolean(form.name && /^[0-9]{17}[0-9Xx]$/.test(form.documentNumber))
  if (method.method === 'school_sso') return Boolean(form.password)
  return Boolean(form.name)
})
const submitLabel = computed(() => {
  if (methodBusy.value) return t('user.verification.student.platform.processing')
  if (emailChallenge.value) return t('user.verification.student.platform.email.verifyCode')
  if (selectedMethod.value?.method === 'student_email_outbound_otp') return t('user.verification.student.platform.email.sendCode')
  if (selectedMethod.value?.method === 'student_email_inbound_challenge') return t('user.verification.student.platform.inbound.create')
  return t('user.verification.student.platform.verify')
})
const trustSteps = computed<Array<{ title: string; description: string; state: TrustStepState }>>(() => [
  {
    title: t('user.verification.student.platform.trust.school'),
    description: selectedSchool.value?.name ?? t('user.verification.student.platform.trust.schoolPending'),
    state: selectedSchool.value ? 'done' : 'active',
  },
  {
    title: t('user.verification.student.platform.trust.method'),
    description: selectedMethod.value ? methodLabel(selectedMethod.value.method) : t('user.verification.student.platform.trust.methodPending'),
    state: applicationComplete.value ? 'done' : selectedSchool.value ? 'active' : 'pending',
  },
  {
    title: t('user.verification.student.platform.trust.credential'),
    description: application.value?.status === 'approved'
      ? t('user.verification.student.platform.trust.credentialDone')
      : t('user.verification.student.platform.trust.credentialPending'),
    state: application.value?.status === 'approved' ? 'done' : application.value ? 'active' : 'pending',
  },
])

async function readPayload<T>(request: Promise<unknown>, fallback: string): Promise<T> {
  const result = await request as ApiCallResult<T>
  const data = extractResultData(result)
  if (data === undefined) throw new Error(fallback)
  return data
}

async function loadPlatform(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    const [schoolList, credentialList] = await Promise.all([
      readPayload<VerificationSchool[]>(api.studentVerification.listSchools(), 'Unable to load schools'),
      readPayload<StudentVerificationCredential[]>(api.studentVerification.listCredentials(), 'Unable to load credentials'),
    ])
    schools.value = schoolList
    credentials.value = credentialList
    await recoverApplication()
  } catch (error) {
    loadError.value = getErrorMessage(error, t('user.verification.student.platform.errors.load'))
  } finally {
    loading.value = false
  }
}

async function recoverApplication(): Promise<void> {
  const handoffToken = typeof route.query.handoff === 'string' ? route.query.handoff : ''
  if (handoffToken) {
    const resumed = await readPayload<VerificationApplication>(
      api.studentVerification.resumeManualCameraHandoff(handoffToken),
      'Unable to resume mobile camera handoff',
    )
    application.value = resumed
    selectedSchoolCode.value = resumed.school.code
    selectedMethod.value = selectedSchool.value?.methods.find((method) => method.method === 'manual_material_review') ?? null
    storeApplication(resumed.id, resumed.school.code)
    const nextQuery = { ...route.query }
    delete nextQuery.handoff
    await router.replace({ query: nextQuery })
    return
  }
  const stored = readStoredApplication()
  if (!stored || !schools.value.some((school) => school.code === stored.schoolCode)) return
  try {
    const recovered = await readPayload<VerificationApplication>(
      api.studentVerification.getApplication(stored.applicationID),
      'Unable to recover application',
    )
    selectedSchoolCode.value = stored.schoolCode
    application.value = recovered
    if (recovered.currentMethod) {
      selectedMethod.value = selectedSchool.value?.methods.find((method) => method.method === recovered.currentMethod) ?? null
    }
    if (recovered.status === 'cancelled' || recovered.status === 'expired' || recovered.status === 'rejected') {
      clearStoredApplication()
      application.value = null
      selectedMethod.value = null
    }
  } catch {
    clearStoredApplication()
  }
}

async function selectSchool(code: string): Promise<void> {
  selectedSchoolCode.value = code
  schoolSearch.value = ''
  void loadEligibility(code)
  await applyRequestedMethod()
}

async function applyRequestedMethod(): Promise<void> {
  const requestedMethod = typeof route.query.method === 'string' && VERIFICATION_METHODS.has(route.query.method as MethodCapability['method'])
    ? route.query.method as MethodCapability['method']
    : null
  if (!requestedMethod || application.value || selectedMethod.value) return

  const method = selectedSchool.value?.methods.find((candidate) => candidate.method === requestedMethod)
  if (!method || method.availability !== 'available' || !method.privacyNotice) return

  if (!(await chooseMethod(method))) return

  const nextQuery = { ...route.query }
  delete nextQuery.method
  await router.replace({ query: nextQuery })
}

function clearSchool(): void {
  selectedSchoolCode.value = ''
  selectedMethod.value = null
  eligibility.value = null
}

async function chooseMethod(method: MethodCapability): Promise<boolean> {
  if (method.availability !== 'available' || !method.privacyNotice || methodBusy.value) return false
  methodBusy.value = true
  methodError.value = ''
  try {
    if (!application.value) {
      const continuation = typeof route.query.continuation === 'string' ? route.query.continuation : undefined
      application.value = await readPayload<VerificationApplication>(
        api.studentVerification.createApplication({
          schoolCode: selectedSchoolCode.value,
          ...(continuation ? { continuationToken: continuation } : {}),
        }),
        'Unable to create verification application',
      )
      storeApplication(application.value.id, selectedSchoolCode.value)
    }
    selectedMethod.value = method
    resetMethodState()
    return true
  } catch (error) {
    methodError.value = getErrorMessage(error, t('user.verification.student.platform.errors.createApplication'))
    return false
  } finally {
    methodBusy.value = false
  }
}

async function submitMethod(): Promise<void> {
  const method = selectedMethod.value
  const currentApplication = application.value
  if (!method || !method.privacyNotice || !currentApplication || !canSubmitMethod.value || methodBusy.value) return
  methodBusy.value = true
  methodError.value = ''
  try {
    if (method.method === 'real_name_identity_check') {
      application.value = await readPayload<VerificationApplication>(
        api.studentVerification.verifyRealName(currentApplication.id, {
          studentID: form.studentID,
          name: form.name,
          documentNumber: form.documentNumber,
          privacyNoticeVersion: method.privacyNotice.version,
          sensitiveDataConsent: true,
        }),
        'Unable to verify real-name information',
      )
      clearSensitiveFields()
      await handleApplicationProgress()
      return
    }
    if (method.method === 'school_sso') {
      application.value = await readPayload<VerificationApplication>(
        api.studentVerification.verifySchoolSSO(currentApplication.id, {
          studentID: form.studentID,
          password: form.password,
          privacyNoticeVersion: method.privacyNotice.version,
          sensitiveDataConsent: true,
        }),
        'Unable to verify school account',
      )
      clearSensitiveFields()
      await handleApplicationProgress()
      return
    }
    if (method.method === 'student_email_outbound_otp') {
      if (!emailChallenge.value) {
        emailChallenge.value = await readPayload<StudentEmailOTPChallenge>(
          api.studentVerification.requestStudentEmailOTP(currentApplication.id, {
            studentID: form.studentID,
            name: form.name,
            privacyNoticeVersion: method.privacyNotice.version,
            sensitiveDataConsent: true,
          }),
          'Unable to request email code',
        )
        return
      }
      application.value = await readPayload<VerificationApplication>(
        api.studentVerification.verifyStudentEmailOTP(currentApplication.id, { code: form.emailCode }),
        'Unable to verify email code',
      )
      form.emailCode = ''
      await handleApplicationProgress()
      return
    }
    if (method.method === 'student_email_inbound_challenge') {
      inboundChallenge.value = await readPayload<InboundEmailChallenge>(
        api.studentVerification.createInboundEmailChallenge(currentApplication.id, {
          studentID: form.studentID,
          name: form.name,
          privacyNoticeVersion: method.privacyNotice.version,
          sensitiveDataConsent: true,
        }),
        'Unable to create inbound email challenge',
      )
      scheduleInboundPoll()
    }
  } catch (error) {
    clearSensitiveFields()
    methodError.value = getErrorMessage(error, t('user.verification.student.platform.errors.cannotComplete'))
  } finally {
    methodBusy.value = false
  }
}

async function handleApplicationProgress(): Promise<void> {
  if (application.value?.status === 'approved') {
    await Promise.all([loadCredentials(), loadEligibility(selectedSchoolCode.value)])
    clearStoredApplication()
  }
}

function scheduleInboundPoll(): void {
  if (inboundPollTimer !== undefined) window.clearTimeout(inboundPollTimer)
  if (inboundChallenge.value?.status !== 'waiting') return
  inboundPollTimer = window.setTimeout(() => void pollInboundChallenge(), INBOUND_POLL_INTERVAL_MS)
}

async function pollInboundChallenge(): Promise<void> {
  if (!application.value) return
  try {
    inboundChallenge.value = await readPayload<InboundEmailChallenge>(
      api.studentVerification.getInboundEmailChallenge(application.value.id),
      'Unable to refresh inbound email challenge',
    )
    if (inboundChallenge.value.status === 'verified') {
      await refreshApplication()
      await handleApplicationProgress()
      return
    }
  } catch (error) {
    methodError.value = getErrorMessage(error, t('user.verification.student.platform.errors.refresh'))
  }
  scheduleInboundPoll()
}

async function refreshApplication(): Promise<void> {
  if (!application.value) return
  application.value = await readPayload<VerificationApplication>(
    api.studentVerification.getApplication(application.value.id),
    'Unable to refresh application',
  )
}

async function handleManualSubmitted(): Promise<void> {
  await refreshApplication()
}

async function cancelApplication(): Promise<void> {
  if (!application.value || methodBusy.value) return
  if (!window.confirm(t('user.verification.student.platform.cancelConfirm'))) return
  methodBusy.value = true
  try {
    await readPayload<VerificationApplication>(
      api.studentVerification.cancelApplication(application.value.id),
      'Unable to cancel application',
    )
    clearStoredApplication()
    application.value = null
    selectedMethod.value = null
    resetMethodState()
  } catch (error) {
    methodError.value = getErrorMessage(error, t('user.verification.student.platform.errors.cancel'))
  } finally {
    methodBusy.value = false
  }
}

async function revokeCredential(credentialID: string): Promise<void> {
  if (!window.confirm(t('user.verification.student.platform.credentials.revokeConfirm'))) return
  try {
    await readPayload<StudentVerificationCredential>(
      api.studentVerification.revokeCredential(credentialID),
      'Unable to revoke credential',
    )
    await loadCredentials()
    if (selectedSchoolCode.value) await loadEligibility(selectedSchoolCode.value)
    toast.success(t('user.verification.student.platform.credentials.revoked'))
  } catch (error) {
    toast.error(getErrorMessage(error, t('user.verification.student.platform.errors.revoke')))
  }
}

async function loadCredentials(): Promise<void> {
  credentials.value = await readPayload<StudentVerificationCredential[]>(
    api.studentVerification.listCredentials(),
    'Unable to load credentials',
  )
}

async function loadEligibility(schoolCode: string): Promise<void> {
  try {
    const result = await readPayload<{ eligible: boolean }>(
      api.studentVerification.getEligibility(schoolCode),
      'Unable to load eligibility',
    )
    eligibility.value = result.eligible
  } catch {
    eligibility.value = null
  }
}

async function submitSchoolSuggestion(): Promise<void> {
  if (suggestionBusy.value || suggestion.name.length < 2) return
  suggestionBusy.value = true
  try {
    await readPayload(
      api.studentVerification.suggestSchool({
        schoolName: suggestion.name,
        ...(suggestion.location.length >= 2 ? { schoolLocation: suggestion.location } : {}),
      }),
      'Unable to submit school suggestion',
    )
    toast.success(t('user.verification.student.platform.school.suggestionSent'))
    showSuggestion.value = false
    suggestion.name = ''
    suggestion.location = ''
  } catch (error) {
    toast.error(getErrorMessage(error, t('user.verification.student.platform.errors.suggestion')))
  } finally {
    suggestionBusy.value = false
  }
}

function returnToMethods(): void {
  selectedMethod.value = null
  resetMethodState()
}

function switchToManualReview(): void {
  const method = selectedSchool.value?.methods.find((candidate) => candidate.method === 'manual_material_review')
  if (method) {
    selectedMethod.value = method
    resetMethodState()
  }
}

function resetMethodState(): void {
  form.studentID = ''
  form.name = ''
  clearSensitiveFields()
  form.emailCode = ''
  form.consented = false
  emailChallenge.value = null
  inboundChallenge.value = null
  methodError.value = ''
  if (inboundPollTimer !== undefined) window.clearTimeout(inboundPollTimer)
}

function clearSensitiveFields(): void {
  form.password = ''
  form.documentNumber = ''
}

function methodLabel(method: Method): string {
  const labels: Record<Method, string> = {
    real_name_identity_check: t('user.verification.student.platform.methods.realName'),
    school_sso: t('user.verification.student.platform.methods.sso'),
    student_email_outbound_otp: t('user.verification.student.platform.methods.emailReceive'),
    student_email_inbound_challenge: t('user.verification.student.platform.methods.emailSend'),
    manual_material_review: t('user.verification.student.platform.methods.manual'),
  }
  return labels[method]
}

function methodExplanation(method: MethodCapability['method']): string {
  const key = method === 'real_name_identity_check'
    ? 'realNameDescription'
    : method === 'school_sso'
      ? 'ssoDescription'
      : method === 'student_email_outbound_otp'
        ? 'emailReceiveDescription'
        : method === 'student_email_inbound_challenge'
          ? 'emailSendDescription'
          : 'manualDescription'
  return t(`user.verification.student.platform.methods.${key}`)
}

function credentialExpiry(value?: string | null): string {
  if (!value) return t('user.verification.student.platform.credentials.noExpiry')
  return t('user.verification.student.platform.credentials.expires', { date: new Date(value).toLocaleDateString() })
}

function formatTime(value: string): string {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function storeApplication(applicationID: string, schoolCode: string): void {
  sessionStorage.setItem(ACTIVE_APPLICATION_STORAGE_KEY, JSON.stringify({ applicationID, schoolCode }))
}

function readStoredApplication(): { applicationID: string; schoolCode: string } | null {
  try {
    const raw = sessionStorage.getItem(ACTIVE_APPLICATION_STORAGE_KEY)
    if (!raw) return null
    const value = JSON.parse(raw) as Record<string, unknown>
    if (typeof value.applicationID !== 'string' || typeof value.schoolCode !== 'string') return null
    return { applicationID: value.applicationID, schoolCode: value.schoolCode }
  } catch {
    return null
  }
}

function clearStoredApplication(): void {
  sessionStorage.removeItem(ACTIVE_APPLICATION_STORAGE_KEY)
}

function finish(): void {
  const target = readAccountFlowRedirect(route.query.redirect)
  if (isCrossOriginAccountFlowRedirect(target)) {
    window.location.assign(target)
    return
  }
  void router.push(target)
}

function startAnother(): void {
  application.value = null
  selectedMethod.value = null
  selectedSchoolCode.value = ''
  resetMethodState()
}

function goBack(): void {
  void router.push('/identity')
}

onMounted(() => void loadPlatform())
onBeforeUnmount(() => {
  clearSensitiveFields()
  if (inboundPollTimer !== undefined) window.clearTimeout(inboundPollTimer)
})
</script>

<style scoped>
@reference "../../../styles/tailwind.css";

.sv-step-number {
  @apply inline-grid size-8 place-items-center rounded-full bg-primary text-xs font-extrabold text-white;
}

.sv-field {
  @apply grid gap-2 text-sm font-semibold text-text-primary;
}

.sv-control {
  @apply min-h-11 w-full rounded-lg border border-border bg-bg-card px-3 py-2.5 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60;
}

.sv-primary {
  @apply inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border-0 bg-primary px-5 py-2.5 text-sm font-bold text-white transition-colors hover:bg-primary-dark focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-bg-card disabled:cursor-not-allowed disabled:opacity-50;
}

.sv-secondary {
  @apply inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-border bg-bg-card px-4 py-2.5 text-sm font-semibold text-text-primary transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50;
}
</style>
