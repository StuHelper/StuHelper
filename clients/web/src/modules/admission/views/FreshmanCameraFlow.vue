<template>
  <section data-admission-freshman-flow class="mt-5 border-t border-slate-200 pt-5">
    <form class="grid gap-4" @submit.prevent="submitFreshmanMaterial">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="field-label">
          学校
          <select
            :value="schoolCode"
            class="field-control"
            data-freshman-school-select
            :disabled="applicationLocked"
            @change="updateSchoolCode"
          >
            <option value="" disabled>请选择学校</option>
            <option
              v-for="school in schools"
              :key="school.schoolCode"
              :value="school.schoolCode"
            >
              {{ school.schoolName }}（{{ school.schoolCode }}）
            </option>
          </select>
        </label>
        <label class="field-label">
          材料类型
          <select
            :value="materialType"
            class="field-control"
            :disabled="applicationLocked"
            @change="updateMaterialType"
          >
            <option value="admission_notice">录取通知书</option>
            <option value="admission_certificate">录取证明</option>
          </select>
        </label>
      </div>

      <div
        v-if="schools.length === 0"
        class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
        data-freshman-school-empty
      >
        暂无可用学校认证配置，请联系管理员。
      </div>

      <label class="field-label">
        姓名
        <input
          :value="applicantName"
          class="field-control"
          data-freshman-applicant-name-input
          type="text"
          :disabled="applicationLocked"
          @input="updateApplicantName"
        >
      </label>

      <label class="field-label">
        院系或专业
        <input
          :value="departmentOrMajor"
          class="field-control"
          type="text"
          :disabled="applicationLocked"
          @input="updateDepartmentOrMajor"
        >
      </label>

      <div class="grid gap-4">
        <video
          ref="videoRef"
          autoplay
          class="aspect-video w-full rounded-lg bg-slate-950 object-cover"
          muted
          playsinline
        />

        <img
          v-if="previewDataURL"
          :src="previewDataURL"
          alt="录取材料预览"
          class="aspect-video w-full rounded-lg border border-slate-200 object-contain"
        >

        <div class="flex flex-wrap gap-3">
          <button
            v-if="!streamActive"
            class="secondary-button"
            type="button"
            :disabled="!cameraSupported || desktopCaptureLocked"
            @click="openCamera"
          >
            {{ cameraSupported ? '打开摄像头' : '当前浏览器不支持摄像头' }}
          </button>
          <button
            v-if="streamActive"
            class="secondary-button"
            type="button"
            :disabled="desktopCaptureLocked"
            @click="captureMaterial"
          >
            拍摄
          </button>
          <button
            v-if="capturedPayload"
            class="secondary-button"
            type="button"
            :disabled="desktopCaptureLocked"
            @click="retake"
          >
            重拍
          </button>
          <button
            class="primary-button"
            data-freshman-submit-button
            type="submit"
            :disabled="submitting || !canSubmit"
          >
            {{ submitting ? '提交中...' : '提交材料' }}
          </button>
          <button
            class="secondary-button"
            data-freshman-mobile-handoff-button
            type="button"
            :disabled="handoffBusy || !canCreateApplication || mobileContinueLocked || handoffActive"
            @click="startMobileHandoff"
          >
            {{ handoffBusy ? '生成中...' : '手机扫码拍照' }}
          </button>
        </div>

        <section
          v-if="handoff"
          class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4"
          data-freshman-mobile-handoff
        >
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start">
            <img
              v-if="handoffQRCodeDataURL"
              :src="handoffQRCodeDataURL"
              alt="手机拍照二维码"
              class="h-40 w-40 rounded-md border border-slate-200 bg-white p-2"
            >
            <div class="grid gap-2 text-sm text-slate-700">
              <p class="font-medium text-slate-900">手机拍照上传</p>
              <p>{{ handoffStatusText }}</p>
              <a
                v-if="handoff.mobileURL"
                class="break-all text-blue-700 hover:text-blue-800"
                :href="handoff.mobileURL"
                target="_blank"
                rel="noreferrer"
              >
                {{ handoff.mobileURL }}
              </a>
            </div>
          </div>
        </section>
      </div>

      <p v-if="errorMessage" class="text-sm text-red-600">{{ errorMessage }}</p>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { toDataURL as createQRCodeDataURL } from 'qrcode'

import type {
  CameraCaptureRequest,
  FreshmanApplication,
  FreshmanCameraHandoff,
} from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import {
  isAdmissionSessionExpiredError,
  isFreshmanCameraHandoffLockedError,
} from '../admissionToken'
import {
  captureFrameAsBase64,
  describeCameraCaptureError,
  startCameraStream,
  stopCameraStream,
  supportsCameraCapture,
} from '../cameraCapture'
import type { AdmissionSchoolOption } from '../oldStudentAdmission'

const HANDOFF_POLL_INTERVAL_MS = 1500
const QRCODE_WIDTH = 192

const props = defineProps<{
  admissionSessionId?: string
  maxMaterialBytes?: number
  schools: AdmissionSchoolOption[]
}>()

const emit = defineEmits<{
  expired: []
  submitted: [application: FreshmanApplication]
}>()

const schoolCode = ref('')
const applicantName = ref('')
const departmentOrMajor = ref('')
const materialType = ref<CameraMaterialType>('admission_notice')
const videoRef = ref<HTMLVideoElement | null>(null)
const stream = ref<MediaStream | null>(null)
const capturedPayload = ref<CameraCaptureRequest | null>(null)
const previewDataURL = ref('')
const submitting = ref(false)
const handoffBusy = ref(false)
const errorMessage = ref('')
const cameraSupported = ref(supportsCameraCapture())
const application = ref<FreshmanApplication | null>(null)
const handoff = ref<FreshmanCameraHandoff | null>(null)
const handoffQRCodeDataURL = ref('')
let handoffPollingID: number | undefined
let handoffEventSource: EventSource | undefined
let desktopContinueEmitted = false

type CameraMaterialType = 'admission_notice' | 'admission_certificate'

const selectedSchool = computed(() => {
  return props.schools.find((school) => school.schoolCode === schoolCode.value) ?? null
})
const streamActive = computed(() => stream.value !== null)
const applicationLocked = computed(() => application.value !== null)
const mobileContinueLocked = computed(() => {
  return handoff.value?.status === 'locked' && handoff.value.continueOn === 'mobile'
})
const mobileUploadedAwaitingChoice = computed(() => {
  return handoff.value?.status === 'uploaded'
})
const handoffActive = computed(() => {
  return handoff.value?.status === 'pending' || handoff.value?.status === 'uploaded'
})
const desktopCaptureLocked = computed(() => {
  return mobileUploadedAwaitingChoice.value || mobileContinueLocked.value
})
const canCreateApplication = computed(() => {
  return Boolean(
    selectedSchool.value &&
    applicantName.value.trim(),
  )
})
const canSubmit = computed(() => {
  return Boolean(
    canCreateApplication.value &&
    capturedPayload.value,
  ) && !desktopCaptureLocked.value && !mobileUploadedAwaitingChoice.value
})
const handoffStatusText = computed(() => {
  const current = handoff.value
  if (!current) return ''
  if (current.status === 'pending') {
    return '请用手机扫描二维码并完成拍照上传。电脑端会自动刷新上传状态。'
  }
  if (current.status === 'uploaded') {
    return '手机已上传材料，请在手机上选择回到电脑端继续或在手机端继续。'
  }
  if (current.status === 'locked' && current.continueOn === 'desktop') {
    return '手机已上传材料，流程已切回电脑端。'
  }
  if (current.status === 'locked' && current.continueOn === 'mobile') {
    return '手机端已继续处理，电脑端已锁定，避免重复提交。'
  }
  return '手机拍照链接已过期，请重新生成。'
})

watch(
  () => props.schools,
  (schools) => {
    if (schools.some((school) => school.schoolCode === schoolCode.value)) return
    schoolCode.value = schools[0]?.schoolCode ?? ''
  },
  { immediate: true },
)

async function openCamera(): Promise<void> {
  errorMessage.value = ''
  try {
    stream.value = await startCameraStream()
    if (videoRef.value) {
      videoRef.value.srcObject = stream.value
      await videoRef.value.play()
    }
  } catch (error) {
    errorMessage.value = describeCameraCaptureError(error, '无法打开摄像头。')
  }
}

function captureMaterial(): void {
  errorMessage.value = ''
  if (!videoRef.value) {
    throw new Error('Camera video element is not mounted')
  }
  try {
    const payload = captureFrameAsBase64(videoRef.value, {
      maxBytes: props.maxMaterialBytes,
    })
    capturedPayload.value = payload
    previewDataURL.value = `data:${payload.contentType};base64,${payload.imageBase64}`
  } catch (error) {
    errorMessage.value = describeCameraCaptureError(error, '拍摄失败，请重试。')
  }
}

function retake(): void {
  capturedPayload.value = null
  previewDataURL.value = ''
}

function updateSchoolCode(event: Event): void {
  schoolCode.value = readControlValue(event).trim()
}

function updateMaterialType(event: Event): void {
  const value = readControlValue(event)
  if (value !== 'admission_notice' && value !== 'admission_certificate') {
    throw new Error('Unsupported freshman material type')
  }
  materialType.value = value
}

function updateApplicantName(event: Event): void {
  applicantName.value = readControlValue(event).trim()
}

function updateDepartmentOrMajor(event: Event): void {
  departmentOrMajor.value = readControlValue(event).trim()
}

async function submitFreshmanMaterial(): Promise<void> {
  if (submitting.value || !canSubmit.value) return
  errorMessage.value = ''
  if (!capturedPayload.value) {
    throw new Error('Camera capture is required before freshman submission')
  }

  submitting.value = true
  try {
    const application = await ensureFreshmanApplication()
    const reviewed = await admissionApi.uploadCameraCapture(
      application.id,
      capturedPayload.value,
    )
    emit('submitted', reviewed)
  } catch (error) {
    if (handleAdmissionExpiredError(error)) return
    if (await handleFreshmanCameraHandoffLockedError(error)) return
    errorMessage.value = readErrorMessage(error, '材料提交失败。')
  } finally {
    submitting.value = false
  }
}

async function startMobileHandoff(): Promise<void> {
  if (
    handoffBusy.value ||
    !canCreateApplication.value ||
    mobileContinueLocked.value ||
    handoffActive.value
  ) {
    return
  }
  errorMessage.value = ''
  handoffBusy.value = true
  try {
    const currentApplication = await ensureFreshmanApplication()
    const nextHandoff = await admissionApi.createFreshmanCameraHandoff(
      currentApplication.id,
    )
    await applyFreshmanCameraHandoff(nextHandoff)
    startHandoffStatusUpdates()
  } catch (error) {
    if (handleAdmissionExpiredError(error)) return
    if (await handleFreshmanCameraHandoffLockedError(error)) return
    errorMessage.value = readErrorMessage(error, '手机拍照链接生成失败。')
  } finally {
    handoffBusy.value = false
  }
}

async function ensureFreshmanApplication(): Promise<FreshmanApplication> {
  if (application.value) {
    return application.value
  }
  const created = await admissionApi.submitFreshmanApplication(
    buildFreshmanApplicationPayload(),
  )
  application.value = created
  return created
}

async function applyFreshmanCameraHandoff(
  nextHandoff: FreshmanCameraHandoff,
  expectedHandoffID?: string,
): Promise<void> {
  if (expectedHandoffID) {
    const current = handoff.value
    if (nextHandoff.id !== expectedHandoffID || current?.id !== expectedHandoffID) {
      return
    }
  }
  handoff.value = nextHandoff
  if (nextHandoff.mobileURL) {
    handoffQRCodeDataURL.value = await createQRCodeDataURL(nextHandoff.mobileURL, {
      errorCorrectionLevel: 'M',
      margin: 1,
      width: QRCODE_WIDTH,
    })
  }
  if (
    nextHandoff.status === 'locked' &&
    nextHandoff.continueOn === 'desktop' &&
    application.value &&
    !desktopContinueEmitted
  ) {
    desktopContinueEmitted = true
    emit('submitted', application.value)
  }
  if (nextHandoff.status === 'locked' || nextHandoff.status === 'expired') {
    stopHandoffPolling()
    stopHandoffStatusUpdates()
  }
}

function startHandoffStatusUpdates(): void {
  stopHandoffPolling()
  stopHandoffStatusUpdates()
  const current = handoff.value
  if (!current) return
  if (typeof window.EventSource !== 'function') {
    startHandoffPolling()
    return
  }
  const source = new window.EventSource(buildHandoffEventsPath(current.id))
  handoffEventSource = source
  source.addEventListener('handoff', (event) => {
    try {
      void applyFreshmanCameraHandoff(parseHandoffEvent(event), current.id)
    } catch (error) {
      errorMessage.value = readErrorMessage(error, '手机拍照状态解析失败。')
    }
  })
  source.onerror = () => {
    stopHandoffStatusUpdates()
    startHandoffPolling()
  }
}

function stopHandoffStatusUpdates(): void {
  if (!handoffEventSource) return
  handoffEventSource.close()
  handoffEventSource = undefined
}

function startHandoffPolling(): void {
  stopHandoffPolling()
  handoffPollingID = window.setInterval(() => {
    void refreshHandoff()
  }, HANDOFF_POLL_INTERVAL_MS)
}

function stopHandoffPolling(): void {
  if (handoffPollingID !== undefined) {
    window.clearInterval(handoffPollingID)
    handoffPollingID = undefined
  }
}

async function refreshHandoff(): Promise<void> {
  const current = handoff.value
  if (!current) return
  try {
    await applyFreshmanCameraHandoff(
      await admissionApi.getFreshmanCameraHandoff(current.id),
      current.id,
    )
  } catch (error) {
    if (handleAdmissionExpiredError(error)) return
    errorMessage.value = readErrorMessage(error, '手机拍照状态刷新失败。')
  }
}

function handleAdmissionExpiredError(error: unknown): boolean {
  if (!isAdmissionSessionExpiredError(error)) return false
  emit('expired')
  stopHandoffPolling()
  stopHandoffStatusUpdates()
  return true
}

async function handleFreshmanCameraHandoffLockedError(error: unknown): Promise<boolean> {
  if (!isFreshmanCameraHandoffLockedError(error)) return false
  if (handoff.value) {
    await refreshHandoff()
  }
  const current = handoff.value
  if (current?.status === 'uploaded') {
    errorMessage.value = '手机端已上传材料，请先在手机上选择回到电脑端继续或在手机端继续。'
    return true
  }
  if (current?.status === 'locked' && current.continueOn === 'desktop') {
    errorMessage.value = ''
    return true
  }
  if (current?.status === 'locked' && current.continueOn === 'mobile') {
    errorMessage.value = '手机端已继续处理，电脑端已锁定，避免重复提交。'
    return true
  }
  errorMessage.value = '当前材料流程已在另一端继续或已提交，请刷新页面确认最新状态。'
  return true
}

function buildFreshmanApplicationPayload() {
  if (!selectedSchool.value) {
    throw new Error('请选择学校')
  }
  return {
    schoolCode: selectedSchool.value.schoolCode,
    ...(props.admissionSessionId ? { admissionSessionID: props.admissionSessionId } : {}),
    applicantName: applicantName.value.trim(),
    departmentOrMajor: departmentOrMajor.value.trim() || undefined,
    materialType: materialType.value,
  }
}

function buildHandoffEventsPath(handoffID: string): string {
  return `/api/v1/admission/freshman/camera-handoffs/${encodeURIComponent(handoffID)}/events`
}

function parseHandoffEvent(event: Event): FreshmanCameraHandoff {
  if (!(event instanceof MessageEvent) || typeof event.data !== 'string') {
    throw new Error('Invalid freshman camera handoff event')
  }
  const parsed = JSON.parse(event.data) as Partial<FreshmanCameraHandoff>
  if (
    typeof parsed.id !== 'string' ||
    typeof parsed.applicationID !== 'string' ||
    typeof parsed.status !== 'string' ||
    typeof parsed.expiresAt !== 'string'
  ) {
    throw new Error('Invalid freshman camera handoff event')
  }
  return parsed as FreshmanCameraHandoff
}

function readErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}

function readControlValue(event: Event): string {
  const target = event.target
  if (
    target instanceof HTMLInputElement ||
    target instanceof HTMLSelectElement
  ) {
    return target.value
  }
  throw new Error('Admission form event target is invalid')
}

onBeforeUnmount(() => {
  stopCameraStream(stream.value)
  stopHandoffPolling()
  stopHandoffStatusUpdates()
})
</script>

<style scoped>
.field-label {
  color: #334155;
  display: grid;
  font-size: 14px;
  font-weight: 600;
  gap: 6px;
}

.field-control {
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  color: #0f172a;
  min-height: 40px;
  padding: 8px 10px;
}
</style>
