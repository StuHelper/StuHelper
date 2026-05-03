<template>
  <section data-admission-freshman-flow class="mt-5 border-t border-slate-200 pt-5">
    <form class="grid gap-4" @submit.prevent="submitFreshmanMaterial">
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="field-label">
          学校 ID
          <input
            :value="schoolID ?? ''"
            class="field-control"
            min="1"
            type="number"
            @input="updateSchoolID"
          >
        </label>
        <label class="field-label">
          材料类型
          <select
            :value="materialType"
            class="field-control"
            @change="updateMaterialType"
          >
            <option value="admission_notice">录取通知书</option>
            <option value="admission_certificate">录取证明</option>
          </select>
        </label>
      </div>

      <label class="field-label">
        姓名
        <input
          :value="applicantName"
          class="field-control"
          type="text"
          @input="updateApplicantName"
        >
      </label>

      <label class="field-label">
        院系或专业
        <input
          :value="departmentOrMajor"
          class="field-control"
          type="text"
          @input="updateDepartmentOrMajor"
        >
      </label>

      <div
        v-if="!cameraSupported"
        class="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
        data-camera-unavailable
      >
        请用手机浏览器打开此链接，并允许浏览器访问摄像头。
      </div>

      <div v-else class="grid gap-4">
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
            @click="openCamera"
          >
            打开摄像头
          </button>
          <button
            v-if="streamActive"
            class="secondary-button"
            type="button"
            @click="captureMaterial"
          >
            拍摄
          </button>
          <button
            v-if="capturedPayload"
            class="secondary-button"
            type="button"
            @click="retake"
          >
            重拍
          </button>
          <button
            class="primary-button"
            type="submit"
            :disabled="submitting || !canSubmit"
          >
            {{ submitting ? '提交中...' : '提交材料' }}
          </button>
        </div>
      </div>

      <p v-if="errorMessage" class="text-sm text-red-600">{{ errorMessage }}</p>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'

import type {
  CameraCaptureRequest,
  FreshmanApplication,
} from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import {
  captureFrameAsBase64,
  startCameraStream,
  stopCameraStream,
  supportsCameraCapture,
} from '../cameraCapture'

const MIN_SCHOOL_ID = 1

const props = defineProps<{
  maxMaterialBytes?: number
}>()

const emit = defineEmits<{
  submitted: [application: FreshmanApplication]
}>()

const schoolID = ref<number | null>(null)
const applicantName = ref('')
const departmentOrMajor = ref('')
const materialType = ref<CameraMaterialType>('admission_notice')
const videoRef = ref<HTMLVideoElement | null>(null)
const stream = ref<MediaStream | null>(null)
const capturedPayload = ref<CameraCaptureRequest | null>(null)
const previewDataURL = ref('')
const submitting = ref(false)
const errorMessage = ref('')
const cameraSupported = ref(supportsCameraCapture())

type CameraMaterialType = 'admission_notice' | 'admission_certificate'

const streamActive = computed(() => stream.value !== null)
const canSubmit = computed(() => {
  return Boolean(
    schoolID.value &&
    schoolID.value >= MIN_SCHOOL_ID &&
    applicantName.value.trim() &&
    capturedPayload.value,
  )
})

async function openCamera(): Promise<void> {
  errorMessage.value = ''
  try {
    stream.value = await startCameraStream()
    if (videoRef.value) {
      videoRef.value.srcObject = stream.value
      await videoRef.value.play()
    }
  } catch (error) {
    cameraSupported.value = false
    errorMessage.value = readErrorMessage(error, '无法打开摄像头。')
  }
}

function captureMaterial(): void {
  errorMessage.value = ''
  if (!videoRef.value) {
    throw new Error('Camera video element is not mounted')
  }
  const payload = captureFrameAsBase64(videoRef.value, {
    maxBytes: props.maxMaterialBytes,
  })
  capturedPayload.value = payload
  previewDataURL.value = `data:${payload.contentType};base64,${payload.imageBase64}`
}

function retake(): void {
  capturedPayload.value = null
  previewDataURL.value = ''
}

function updateSchoolID(event: Event): void {
  const value = readControlValue(event)
  schoolID.value = value ? Number(value) : null
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
  errorMessage.value = ''
  const payload = buildFreshmanApplicationPayload()
  if (!capturedPayload.value) {
    throw new Error('Camera capture is required before freshman submission')
  }

  submitting.value = true
  try {
    const application = await admissionApi.submitFreshmanApplication(payload)
    const reviewed = await admissionApi.uploadCameraCapture(
      application.id,
      capturedPayload.value,
    )
    emit('submitted', reviewed)
  } catch (error) {
    errorMessage.value = readErrorMessage(error, '材料提交失败。')
  } finally {
    submitting.value = false
  }
}

function buildFreshmanApplicationPayload() {
  const normalizedSchoolID = schoolID.value
  if (!normalizedSchoolID || normalizedSchoolID < MIN_SCHOOL_ID) {
    throw new Error('School ID is required')
  }
  return {
    schoolID: normalizedSchoolID,
    applicantName: applicantName.value.trim(),
    departmentOrMajor: departmentOrMajor.value.trim() || undefined,
    materialType: materialType.value,
  }
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
