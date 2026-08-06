<template>
  <section class="grid gap-5" data-manual-review-flow>
    <div
      v-if="reviewCase && reviewCase.status !== 'draft' && reviewCase.status !== 'supplement_required'"
      class="rounded-xl border p-5"
      :class="statusPanelClass"
      role="status"
    >
      <div class="flex items-start gap-3">
        <Clock3 v-if="reviewCase.status === 'pending'" class="mt-0.5 size-5 shrink-0" aria-hidden="true" />
        <BadgeCheck v-else-if="reviewCase.status === 'approved'" class="mt-0.5 size-5 shrink-0" aria-hidden="true" />
        <CircleAlert v-else class="mt-0.5 size-5 shrink-0" aria-hidden="true" />
        <div>
          <h3 class="m-0 text-sm font-bold">{{ reviewStatusTitle }}</h3>
          <p class="m-0 mt-1 text-sm leading-6">{{ reviewStatusDescription }}</p>
          <p v-if="reviewCase.userVisibleReason" class="m-0 mt-2 text-sm">
            {{ reviewCase.userVisibleReason }}
          </p>
        </div>
      </div>
    </div>

    <form
      v-if="canEdit"
      class="grid gap-4"
      :aria-busy="saving"
      @submit.prevent="saveDraft"
    >
      <div class="grid gap-4 sm:grid-cols-2">
        <label class="sv-field">
          <span>{{ t('user.verification.student.platform.manual.materialType') }}</span>
          <select v-model="materialType" class="sv-control" name="material_type" required>
            <option value="campus_card">{{ t('user.verification.student.platform.manual.campusCard') }}</option>
            <option value="student_card">{{ t('user.verification.student.platform.manual.studentCard') }}</option>
            <option value="admission_notice">{{ t('user.verification.student.platform.manual.admissionNotice') }}</option>
            <option value="other_approved">{{ t('user.verification.student.platform.manual.otherApproved') }}</option>
          </select>
        </label>

        <label
          v-for="field in method.formFields"
          :key="field.key"
          class="sv-field"
          :class="field.inputType === 'textarea' ? 'sm:col-span-2' : ''"
        >
          <span>{{ field.label }}</span>
          <span v-if="field.helpText" class="font-normal text-text-muted">{{ field.helpText }}</span>
          <select
            v-if="field.inputType === 'select'"
            v-model="formValues[field.key]"
            class="sv-control"
            :name="`manual_${field.key}`"
            :required="field.required"
          >
            <option value="" disabled>—</option>
            <option v-for="option in field.options" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
          <textarea
            v-else-if="field.inputType === 'textarea'"
            v-model.trim="formValues[field.key]"
            class="sv-control min-h-28 resize-y"
            :name="`manual_${field.key}`"
            :required="field.required"
            :maxlength="field.maxLength ?? 500"
            :autocomplete="field.autocomplete"
          />
          <input
            v-else
            v-model.trim="formValues[field.key]"
            class="sv-control"
            :name="`manual_${field.key}`"
            :type="field.inputType"
            :required="field.required"
            :maxlength="field.maxLength ?? 500"
            :autocomplete="field.autocomplete"
          />
        </label>
      </div>

      <PrivacyConsent
        v-if="method.privacyNotice"
        v-model="consented"
        :notice="method.privacyNotice"
      />

      <button class="sv-primary" type="submit" :disabled="!canSave || saving">
        <LoaderCircle v-if="saving" class="size-4 animate-spin" aria-hidden="true" />
        {{ reviewCase ? t('user.verification.student.platform.manual.updateDraft') : t('user.verification.student.platform.manual.createDraft') }}
      </button>
    </form>

    <template v-if="reviewCase && canEdit">
      <section
        v-if="reviewCase.emailVerificationRequired"
        class="rounded-xl border border-border bg-bg-elevated/60 p-4"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <h3 class="m-0 text-sm font-bold text-text-primary">
              {{ t('user.verification.student.platform.manual.emailTitle') }}
            </h3>
            <p class="m-0 mt-1 text-sm leading-6 text-text-muted">
              {{ reviewCase.emailVerified
                ? t('user.verification.student.platform.manual.emailVerified')
                : t('user.verification.student.platform.manual.emailDescription') }}
            </p>
          </div>
          <BadgeCheck v-if="reviewCase.emailVerified" class="size-5 shrink-0 text-success" aria-hidden="true" />
        </div>
        <div v-if="!reviewCase.emailVerified" class="mt-4 grid gap-3">
          <button
            v-if="!emailChallenge"
            class="sv-secondary"
            type="button"
            :disabled="emailBusy"
            @click="requestEmailCode"
          >
            {{ t('user.verification.student.platform.manual.sendEmailCode') }}
          </button>
          <template v-else>
            <p class="m-0 text-sm text-text-secondary">
              {{ t('user.verification.student.platform.manual.codeSentTo', { email: emailChallenge.maskedEmail }) }}
            </p>
            <OtpCodeInput
              v-model="emailCode"
              :disabled="emailBusy"
              :aria-label="t('user.verification.student.platform.email.code')"
            />
            <button
              class="sv-secondary"
              type="button"
              :disabled="emailCode.length !== 6 || emailBusy"
              @click="verifyEmailCode"
            >
              {{ t('user.verification.student.platform.email.verifyCode') }}
            </button>
          </template>
        </div>
      </section>

      <section class="rounded-xl border border-border p-4" data-manual-camera>
        <div class="flex items-start justify-between gap-4">
          <div>
            <h3 class="m-0 text-sm font-bold text-text-primary">
              {{ t('user.verification.student.platform.manual.cameraTitle') }}
            </h3>
            <p class="m-0 mt-1 text-sm leading-6 text-text-muted">
              {{ t('user.verification.student.platform.manual.cameraDescription') }}
            </p>
          </div>
          <span class="rounded-full bg-bg-elevated px-2.5 py-1 text-xs font-semibold text-text-secondary">
            {{ reviewCase.materials.length }} / 5
          </span>
        </div>

        <div class="mt-4 grid gap-3">
          <video
            v-show="stream"
            ref="videoRef"
            autoplay
            muted
            playsinline
            class="aspect-video w-full rounded-xl bg-black object-cover"
          />
          <img
            v-if="capturePreview"
            :src="capturePreview"
            :alt="t('user.verification.student.platform.manual.capturePreview')"
            width="1280"
            height="720"
            class="aspect-video w-full rounded-xl border border-border bg-bg-elevated object-contain"
          />

          <div class="flex flex-wrap gap-2">
            <button v-if="!stream" class="sv-secondary" type="button" :disabled="!cameraSupported" @click="openCamera">
              <Camera class="size-4" aria-hidden="true" />
              {{ cameraSupported
                ? t('user.verification.student.platform.manual.openCamera')
                : t('user.verification.student.platform.manual.cameraUnsupported') }}
            </button>
            <button v-else class="sv-secondary" type="button" :disabled="captureBusy" @click="captureAndUpload">
              <ScanLine class="size-4" aria-hidden="true" />
              {{ captureBusy
                ? t('user.verification.student.platform.manual.uploading')
                : t('user.verification.student.platform.manual.capture') }}
            </button>
            <button class="sv-secondary" type="button" :disabled="handoffBusy || handoffActive" @click="createHandoff">
              <QrCode class="size-4" aria-hidden="true" />
              {{ t('user.verification.student.platform.manual.usePhone') }}
            </button>
          </div>
        </div>

        <div v-if="handoff" class="mt-4 grid gap-4 rounded-xl bg-bg-elevated p-4 sm:grid-cols-[auto_1fr]">
          <img
            v-if="handoffQRCode"
            :src="handoffQRCode"
            :alt="t('user.verification.student.platform.manual.qrAlt')"
            width="160"
            height="160"
            class="size-40 rounded-lg bg-white p-2"
          />
          <div>
            <p class="m-0 text-sm font-bold text-text-primary">{{ handoffStatusTitle }}</p>
            <p class="m-0 mt-2 text-sm leading-6 text-text-muted">{{ handoffStatusDescription }}</p>
            <a
              v-if="handoff.mobileURL"
              class="mt-3 inline-flex min-h-11 items-center text-sm font-semibold text-primary underline-offset-4 hover:underline"
              :href="absoluteHandoffURL"
              target="_blank"
              rel="noopener noreferrer"
            >
              {{ t('user.verification.student.platform.manual.openPhoneLink') }}
            </a>
          </div>
        </div>
      </section>

      <div class="rounded-xl border border-primary/20 bg-primary/5 p-4 text-sm leading-6 text-text-secondary">
        {{ t('user.verification.student.platform.manual.retentionConfirmation') }}
      </div>

      <button class="sv-primary" type="button" :disabled="!canSubmit || submitting" @click="submitReview">
        <Send class="size-4" aria-hidden="true" />
        {{ submitting
          ? t('user.verification.student.platform.manual.submitting')
          : t('user.verification.student.platform.manual.submit') }}
      </button>
    </template>

    <p v-if="errorMessage" class="m-0 rounded-lg bg-danger/10 p-3 text-sm text-danger" role="alert">
      {{ errorMessage }}
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { toDataURL as createQRCodeDataURL } from 'qrcode'
import {
  BadgeCheck,
  Camera,
  CircleAlert,
  Clock3,
  LoaderCircle,
  QrCode,
  ScanLine,
  Send,
} from 'lucide-vue-next'
import type {
  ApiCallResult,
  ManualCameraHandoff,
  ManualEmailOTPChallenge,
  ManualReviewCase,
  VerificationSchool,
} from '@stuhelper/shared/api'
import { extractResultData } from '@stuhelper/shared/api'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import OtpCodeInput from '@/components/common/OtpCodeInput.vue'
import {
  captureFrameAsBase64,
  describeCameraCaptureError,
  startCameraStream,
  stopCameraStream,
  supportsCameraCapture,
} from '@/utils/cameraCapture'
import PrivacyConsent from './PrivacyConsent.vue'

type VerificationMethodCapability = VerificationSchool['methods'][number]
type MaterialType = ManualReviewCase['materialType']

const props = defineProps<{
  applicationId: string
  method: VerificationMethodCapability
}>()

const emit = defineEmits<{
  updated: [reviewCase: ManualReviewCase]
  submitted: [reviewCase: ManualReviewCase]
}>()

const { t } = useI18n()
const formValues = reactive<Record<string, string>>({})
const materialType = ref<MaterialType>('student_card')
const consented = ref(false)
const reviewCase = ref<ManualReviewCase | null>(null)
const emailChallenge = ref<ManualEmailOTPChallenge | null>(null)
const emailCode = ref('')
const saving = ref(false)
const emailBusy = ref(false)
const captureBusy = ref(false)
const handoffBusy = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const cameraSupported = ref(supportsCameraCapture())
const videoRef = ref<HTMLVideoElement | null>(null)
const stream = ref<MediaStream | null>(null)
const capturePreview = ref('')
const handoff = ref<ManualCameraHandoff | null>(null)
const handoffQRCode = ref('')
let pollTimer: number | undefined

const canEdit = computed(() => {
  const status = reviewCase.value?.status
  return !status || status === 'draft' || status === 'supplement_required'
})
const canSave = computed(() => {
  if (!props.method.privacyNotice || !consented.value) return false
  return props.method.formFields.every((field) => !field.required || Boolean(formValues[field.key]?.trim()))
})
const canSubmit = computed(() => {
  const current = reviewCase.value
  return Boolean(
    current &&
    current.materials.length > 0 &&
    (!current.emailVerificationRequired || current.emailVerified),
  )
})
const handoffActive = computed(() => handoff.value?.status === 'pending' || handoff.value?.status === 'uploaded')
const absoluteHandoffURL = computed(() => {
  const url = handoff.value?.mobileURL
  if (!url) return ''
  return new URL(url, window.location.origin).href
})
const reviewStatusTitle = computed(() => {
  if (reviewCase.value?.status === 'pending') return t('user.verification.student.platform.manual.pendingTitle')
  if (reviewCase.value?.status === 'approved') return t('user.verification.student.platform.manual.approvedTitle')
  if (reviewCase.value?.status === 'rejected') return t('user.verification.student.platform.manual.rejectedTitle')
  return t('user.verification.student.platform.manual.closedTitle')
})
const reviewStatusDescription = computed(() => {
  if (reviewCase.value?.status === 'pending') return t('user.verification.student.platform.manual.pendingDescription')
  if (reviewCase.value?.status === 'approved') return t('user.verification.student.platform.manual.approvedDescription')
  return t('user.verification.student.platform.manual.closedDescription')
})
const statusPanelClass = computed(() => {
  if (reviewCase.value?.status === 'approved') return 'border-success/30 bg-success/10 text-success'
  if (reviewCase.value?.status === 'pending') return 'border-warning/30 bg-warning/10 text-text-primary'
  return 'border-danger/30 bg-danger/10 text-danger'
})
const handoffStatusTitle = computed(() => {
  if (handoff.value?.status === 'pending') return t('user.verification.student.platform.manual.scanTitle')
  if (handoff.value?.status === 'uploaded') return t('user.verification.student.platform.manual.uploadedTitle')
  if (handoff.value?.status === 'locked') return t('user.verification.student.platform.manual.returnedTitle')
  return t('user.verification.student.platform.manual.expiredTitle')
})
const handoffStatusDescription = computed(() => {
  if (handoff.value?.status === 'pending') return t('user.verification.student.platform.manual.scanDescription')
  if (handoff.value?.status === 'uploaded') return t('user.verification.student.platform.manual.chooseDeviceDescription')
  if (handoff.value?.status === 'locked') return t('user.verification.student.platform.manual.returnedDescription')
  return t('user.verification.student.platform.manual.expiredDescription')
})

async function readPayload<T>(request: Promise<unknown>, fallback: string): Promise<T> {
  const result = await request as ApiCallResult<T>
  const data = extractResultData(result)
  if (data === undefined) throw new Error(fallback)
  return data
}

async function saveDraft(): Promise<void> {
  if (!canSave.value || saving.value || !props.method.privacyNotice) return
  saving.value = true
  errorMessage.value = ''
  try {
    reviewCase.value = await readPayload<ManualReviewCase>(
      api.studentVerification.upsertManualReview(props.applicationId, {
        materialType: materialType.value,
        formValues: { ...formValues },
        privacyNoticeVersion: props.method.privacyNotice.version,
        sensitiveDataConsent: true,
      }),
      'Unable to save manual review draft',
    )
    emit('updated', reviewCase.value)
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.saveDraft'))
  } finally {
    saving.value = false
  }
}

async function requestEmailCode(): Promise<void> {
  emailBusy.value = true
  errorMessage.value = ''
  try {
    emailChallenge.value = await readPayload<ManualEmailOTPChallenge>(
      api.studentVerification.requestManualEmailOTP(props.applicationId),
      'Unable to request email code',
    )
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.emailCode'))
  } finally {
    emailBusy.value = false
  }
}

async function verifyEmailCode(): Promise<void> {
  if (emailCode.value.length !== 6) return
  emailBusy.value = true
  errorMessage.value = ''
  try {
    reviewCase.value = await readPayload<ManualReviewCase>(
      api.studentVerification.verifyManualEmailOTP(props.applicationId, { code: emailCode.value }),
      'Unable to verify email code',
    )
    emailCode.value = ''
    emit('updated', reviewCase.value)
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.invalidCode'))
  } finally {
    emailBusy.value = false
  }
}

async function openCamera(): Promise<void> {
  errorMessage.value = ''
  try {
    stream.value = await startCameraStream()
    if (videoRef.value) {
      videoRef.value.srcObject = stream.value
      await videoRef.value.play()
    }
  } catch (error) {
    errorMessage.value = describeCameraCaptureError(error, t('user.verification.student.platform.errors.camera'))
  }
}

async function captureAndUpload(): Promise<void> {
  if (!videoRef.value || captureBusy.value) return
  captureBusy.value = true
  errorMessage.value = ''
  try {
    const frame = captureFrameAsBase64(videoRef.value)
    capturePreview.value = `data:${frame.contentType};base64,${frame.imageBase64}`
    reviewCase.value = await readPayload<ManualReviewCase>(
      api.studentVerification.uploadManualCameraCapture(props.applicationId, {
        ...frame,
        captureSource: 'web_camera',
        requestedFacingMode: 'environment',
      }),
      'Unable to upload camera capture',
    )
    emit('updated', reviewCase.value)
    stopCameraStream(stream.value)
    stream.value = null
  } catch (error) {
    errorMessage.value = describeCameraCaptureError(error, t('user.verification.student.platform.errors.camera'))
  } finally {
    captureBusy.value = false
  }
}

async function createHandoff(): Promise<void> {
  if (handoffBusy.value || !reviewCase.value) return
  handoffBusy.value = true
  errorMessage.value = ''
  try {
    handoff.value = await readPayload<ManualCameraHandoff>(
      api.studentVerification.createManualCameraHandoff(props.applicationId),
      'Unable to create camera handoff',
    )
    handoffQRCode.value = await createQRCodeDataURL(absoluteHandoffURL.value, {
      width: 192,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
    schedulePoll()
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.handoff'))
  } finally {
    handoffBusy.value = false
  }
}

function schedulePoll(): void {
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
  if (!handoff.value || !handoffActive.value) return
  pollTimer = window.setTimeout(() => void pollHandoff(), 1500)
}

async function pollHandoff(): Promise<void> {
  const current = handoff.value
  if (!current) return
  try {
    handoff.value = await readPayload<ManualCameraHandoff>(
      api.studentVerification.getManualCameraHandoff(props.applicationId, current.id),
      'Unable to refresh camera handoff',
    )
    if (handoff.value.status === 'locked' && handoff.value.continueOn === 'desktop') {
      reviewCase.value = await readPayload<ManualReviewCase>(
        api.studentVerification.getManualReview(props.applicationId),
        'Unable to refresh manual review',
      )
      emit('updated', reviewCase.value)
    }
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.handoff'))
  } finally {
    schedulePoll()
  }
}

async function submitReview(): Promise<void> {
  if (!canSubmit.value || submitting.value) return
  submitting.value = true
  errorMessage.value = ''
  try {
    reviewCase.value = await readPayload<ManualReviewCase>(
      api.studentVerification.submitManualReview(props.applicationId, { confirmMaterialUse: true }),
      'Unable to submit manual review',
    )
    emit('submitted', reviewCase.value)
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.errors.submitReview'))
  } finally {
    submitting.value = false
  }
}

onBeforeUnmount(() => {
  stopCameraStream(stream.value)
  if (pollTimer !== undefined) window.clearTimeout(pollTimer)
})
</script>

<style scoped>
@reference "../../../styles/tailwind.css";

.sv-field {
  @apply grid gap-2 text-sm font-semibold text-text-primary;
}

.sv-control {
  @apply min-h-11 w-full rounded-lg border border-border bg-bg-card px-3 py-2.5 text-sm text-text-primary outline-none transition-colors placeholder:text-text-muted focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:cursor-not-allowed disabled:opacity-60;
}

.sv-primary {
  @apply inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border-0 bg-primary px-4 py-2.5 text-sm font-bold text-white transition-colors hover:bg-primary-dark focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-bg-card disabled:cursor-not-allowed disabled:opacity-50;
}

.sv-secondary {
  @apply inline-flex min-h-11 items-center justify-center gap-2 rounded-lg border border-border bg-bg-card px-4 py-2.5 text-sm font-semibold text-text-primary transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50;
}
</style>
