<template>
  <section class="freshman-flow" data-admission-freshman-flow>
    <form class="freshman-flow__form" @submit.prevent="submitFreshmanMaterial">
      <div class="freshman-flow__pair">
        <label class="freshman-flow__field">
          <span class="join-label">学校</span>
          <span class="freshman-flow__select-wrap">
            <select
              :value="schoolCode"
              class="join-select"
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
          </span>
        </label>
        <label class="freshman-flow__field">
          <span class="join-label">材料类型</span>
          <span class="freshman-flow__select-wrap">
            <select
              :value="materialType"
              class="join-select"
              :disabled="applicationLocked"
              @change="updateMaterialType"
            >
              <option value="admission_notice">录取通知书</option>
              <option value="admission_certificate">录取证明</option>
            </select>
          </span>
        </label>
      </div>

      <div
        v-if="schools.length === 0"
        class="freshman-flow__notice join-chip"
        data-freshman-school-empty
      >
        暂无可用学校认证配置，请联系管理员。
      </div>

      <div class="freshman-flow__pair">
        <label class="freshman-flow__field">
          <span class="join-label">姓名</span>
          <input
            :value="applicantName"
            class="join-input"
            data-freshman-applicant-name-input
            type="text"
            :disabled="applicationLocked"
            @input="updateApplicantName"
          >
        </label>
        <label class="freshman-flow__field">
          <span class="join-label">院系或专业</span>
          <input
            :value="departmentOrMajor"
            class="join-input"
            type="text"
            :disabled="applicationLocked"
            @input="updateDepartmentOrMajor"
          >
        </label>
      </div>

      <div class="freshman-flow__camera">
        <div class="freshman-flow__viewport">
          <video
            ref="videoRef"
            autoplay
            class="freshman-flow__video"
            muted
            playsinline
          />
          <span class="freshman-flow__viewfinder" aria-hidden="true" />
        </div>

        <img
          v-if="previewDataURL"
          :src="previewDataURL"
          alt="录取材料预览"
          class="freshman-flow__preview"
        >

        <div class="freshman-flow__actions">
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
          class="freshman-flow__handoff join-chip"
          data-freshman-mobile-handoff
        >
          <p class="freshman-flow__handoff-title">手机拍照上传</p>
          <span v-if="handoffQRCodeDataURL" class="freshman-flow__qr-plate">
            <img
              :src="handoffQRCodeDataURL"
              alt="手机拍照二维码"
              class="freshman-flow__qr"
            >
          </span>
          <p
            class="freshman-flow__handoff-status"
            :class="{
              'join-tone-info': handoff.status === 'pending',
              'join-tone-warning': handoff.status === 'uploaded' || handoff.status === 'expired',
              'join-tone-success': handoff.status === 'locked',
            }"
          >
            {{ handoffStatusText }}
          </p>
          <a
            v-if="handoff.mobileURL"
            class="freshman-flow__handoff-link"
            :href="handoff.mobileURL"
            target="_blank"
            rel="noreferrer"
          >
            {{ handoff.mobileURL }}
          </a>
        </section>
      </div>

      <p v-if="errorMessage" class="freshman-flow__error">{{ errorMessage }}</p>
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
let submissionEmitted = false

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
    application.value &&
    !submissionEmitted
  ) {
    submissionEmitted = true
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
  if (!current && application.value) {
    errorMessage.value = ''
    emit('submitted', application.value)
    return true
  }
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
/*
 * 品牌玻璃风（.join-* 原语与按钮样式由 join-theme.css 提供，
 * AdmissionShell 已在祖先链上挂载 .join-surface，这里不再重复引入）。
 * 测试契约（勿动）：.secondary-button 的 DOM 顺序（打开摄像头/拍摄
 * 必须先于手机扫码按钮）、.primary-button 提交按钮、全部 data-* 选择器、
 * label 包裹原生 select/input、两张 img 的 alt 文案。
 */
.freshman-flow,
.freshman-flow__form,
.freshman-flow__camera {
  display: grid;
  gap: 16px;
}

/* ── 表单字段 ─────────────────────────────────────── */
.freshman-flow__pair {
  display: grid;
  gap: 14px;
}

.freshman-flow__field {
  display: block;
  min-width: 0;
}

.freshman-flow__select-wrap {
  display: block;
  position: relative;
}

.freshman-flow__select-wrap .join-select {
  appearance: none;
  padding-right: 42px;
}

.freshman-flow__select-wrap::after {
  border-bottom: 2px solid var(--join-ink-muted);
  border-right: 2px solid var(--join-ink-muted);
  content: "";
  height: 9px;
  pointer-events: none;
  position: absolute;
  right: 18px;
  top: 50%;
  transform: translateY(-65%) rotate(45deg);
  width: 9px;
}

/* ── 空配置提示：玻璃信息卡 + 警示圆点 ─────────────── */
.freshman-flow__notice {
  align-items: flex-start;
  color: var(--join-ink-soft);
  display: flex;
  font-size: 14px;
  gap: 10px;
  line-height: 22px;
  padding: 14px 16px;
}

.freshman-flow__notice::before {
  background: var(--join-tone-warning-fg);
  border-radius: 999px;
  content: "";
  flex: none;
  height: 8px;
  margin-top: 7px;
  width: 8px;
}

/* ── 取景器：渐变描边玻璃相框 + 固定深色画面 ───────── */
.freshman-flow__viewport {
  background:
    linear-gradient(var(--join-glass-bg-heavy), var(--join-glass-bg-heavy)) padding-box,
    var(--join-gradient-cta) border-box;
  border: 1px solid transparent;
  border-radius: var(--join-radius-card);
  box-shadow: var(--shadow-card);
  padding: 6px;
  position: relative;
}

/* 取景画面保持深色（与真实相机一致，明暗主题不翻转） */
.freshman-flow__video {
  aspect-ratio: 16 / 9;
  background: #10121d;
  border-radius: calc(var(--join-radius-card) - 6px);
  display: block;
  object-fit: cover;
  width: 100%;
}

/* 对角取景框角标：纯装饰，固定浅色以浮在深色画面上 */
.freshman-flow__viewfinder {
  inset: 20px;
  pointer-events: none;
  position: absolute;
}

.freshman-flow__viewfinder::before,
.freshman-flow__viewfinder::after {
  content: "";
  height: 22px;
  position: absolute;
  width: 22px;
}

.freshman-flow__viewfinder::before {
  border-left: 2px solid rgba(255, 255, 255, 0.55);
  border-top: 2px solid rgba(255, 255, 255, 0.55);
  border-top-left-radius: 6px;
  left: 0;
  top: 0;
}

.freshman-flow__viewfinder::after {
  border-bottom: 2px solid rgba(255, 255, 255, 0.55);
  border-right: 2px solid rgba(255, 255, 255, 0.55);
  border-bottom-right-radius: 6px;
  bottom: 0;
  right: 0;
}

/* ── 已拍摄预览 ───────────────────────────────────── */
.freshman-flow__preview {
  aspect-ratio: 16 / 9;
  background: var(--join-chip-bg);
  border: 1px solid var(--join-glass-border);
  border-radius: var(--join-radius-card);
  object-fit: contain;
  width: 100%;
}

/* ── 操作区：移动端整列堆叠，桌面端按内容横排 ───────── */
.freshman-flow__actions {
  display: grid;
  gap: 10px;
}

/* ── 手机接力面板：玻璃 chip 卡，二维码居中 ─────────── */
.freshman-flow__handoff {
  display: grid;
  gap: 12px;
  justify-items: center;
  padding: 20px 16px;
  text-align: center;
}

.freshman-flow__handoff-title {
  color: var(--join-ink);
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
  margin: 0;
}

/* 二维码衬底固定纯白，保证暗色主题下的扫码对比度 */
.freshman-flow__qr-plate {
  background: #ffffff;
  border-radius: var(--join-radius-control);
  box-shadow: var(--shadow-card);
  display: inline-block;
  line-height: 0;
  padding: 10px;
}

.freshman-flow__qr {
  display: block;
  height: 160px;
  width: 160px;
}

/* 状态句的语义底色由 .join-tone-* 提供（模板按 handoff.status 绑定） */
.freshman-flow__handoff-status {
  border-radius: var(--radius-lg);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  margin: 0;
  max-width: 46ch;
  padding: 10px 14px;
}

.freshman-flow__handoff-link {
  color: var(--join-tone-info-fg);
  font-size: 13px;
  line-height: 20px;
  overflow-wrap: anywhere;
  text-decoration: underline;
  text-underline-offset: 3px;
}

.freshman-flow__handoff-link:hover {
  color: var(--color-primary);
}

.freshman-flow__handoff-link:focus-visible {
  border-radius: 4px;
  outline: 3px solid rgba(91, 124, 247, 0.45);
  outline-offset: 2px;
}

/* ── 错误反馈气泡 ─────────────────────────────────── */
.freshman-flow__error {
  background: var(--join-tone-danger-bg);
  border-radius: var(--radius-lg);
  color: var(--join-tone-danger-fg);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  margin: 0;
  padding: 10px 14px;
}

/* ── ≥640px：双列字段 + 横排操作 ─────────────────── */
@media (min-width: 640px) {
  .freshman-flow__pair {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .freshman-flow__actions {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }

  .freshman-flow__handoff {
    padding: 22px 20px;
  }
}
</style>
