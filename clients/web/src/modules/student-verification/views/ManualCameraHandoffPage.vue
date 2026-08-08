<template>
  <main class="handoff-shell min-h-screen bg-bg-base px-4 py-6 text-text-primary sm:py-10" data-manual-camera-handoff-page>
    <section class="mx-auto w-full max-w-xl overflow-hidden rounded-2xl border border-border bg-bg-card shadow-card">
      <header class="border-b border-border p-5 sm:p-6">
        <p class="m-0 text-xs font-bold uppercase tracking-[0.18em] text-primary">StuHelper</p>
        <h1 class="m-0 mt-2 text-2xl font-extrabold tracking-tight">
          {{ t('user.verification.student.platform.mobileCamera.title') }}
        </h1>
        <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
          {{ t('user.verification.student.platform.mobileCamera.subtitle') }}
        </p>
      </header>

      <div class="p-5 sm:p-6">
        <div v-if="state === 'loading'" class="grid min-h-52 place-items-center" role="status">
          <LoaderCircle class="size-7 animate-spin text-primary" aria-hidden="true" />
          <span class="sr-only">{{ t('common.actions.loading') }}</span>
        </div>

        <section v-else-if="state === 'ready'" class="grid gap-4">
          <div class="rounded-xl border border-primary/20 bg-primary/5 p-4 text-sm leading-6 text-text-secondary">
            <div class="flex gap-3">
              <ShieldCheck class="mt-0.5 size-5 shrink-0 text-primary" aria-hidden="true" />
              <p class="m-0">{{ t('user.verification.student.platform.mobileCamera.privacy') }}</p>
            </div>
          </div>
          <video
            v-show="stream"
            ref="videoRef"
            autoplay
            muted
            playsinline
            class="aspect-video w-full rounded-xl bg-black object-cover"
          />
          <img
            v-if="preview"
            :src="preview"
            :alt="t('user.verification.student.platform.mobileCamera.previewAlt')"
            width="1280"
            height="720"
            class="aspect-video w-full rounded-xl border border-border bg-bg-elevated object-contain"
          />
          <button v-if="!stream" class="handoff-secondary" type="button" :disabled="!cameraSupported" @click="openCamera">
            <Camera class="size-4" aria-hidden="true" />
            {{ cameraSupported
              ? t('user.verification.student.platform.mobileCamera.openCamera')
              : t('user.verification.student.platform.mobileCamera.unsupported') }}
          </button>
          <button v-else class="handoff-primary" type="button" :disabled="uploading" @click="captureAndUpload">
            <ScanLine class="size-4" aria-hidden="true" />
            {{ uploading
              ? t('user.verification.student.platform.mobileCamera.uploading')
              : t('user.verification.student.platform.mobileCamera.capture') }}
          </button>
        </section>

        <section v-else-if="state === 'uploaded'" class="py-4 text-center">
          <span class="mx-auto grid size-14 place-items-center rounded-2xl bg-success/15 text-success">
            <BadgeCheck class="size-7" aria-hidden="true" />
          </span>
          <h2 class="m-0 mt-4 text-lg font-extrabold">
            {{ t('user.verification.student.platform.mobileCamera.uploadedTitle') }}
          </h2>
          <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
            {{ t('user.verification.student.platform.mobileCamera.uploadedDescription') }}
          </p>
          <div class="mt-5 grid gap-2">
            <button class="handoff-primary" type="button" :disabled="choosing" @click="chooseContinuation('mobile')">
              {{ t('user.verification.student.platform.mobileCamera.continueMobile') }}
            </button>
            <button class="handoff-secondary" type="button" :disabled="choosing" @click="chooseContinuation('desktop')">
              {{ t('user.verification.student.platform.mobileCamera.returnDesktop') }}
            </button>
          </div>
        </section>

        <section v-else-if="state === 'desktop'" class="py-8 text-center" role="status">
          <MonitorCheck class="mx-auto size-10 text-success" aria-hidden="true" />
          <h2 class="m-0 mt-4 text-lg font-extrabold">
            {{ t('user.verification.student.platform.mobileCamera.desktopTitle') }}
          </h2>
          <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
            {{ t('user.verification.student.platform.mobileCamera.desktopDescription') }}
          </p>
        </section>

        <section v-else-if="state === 'expired'" class="py-8 text-center" role="status">
          <TimerOff class="mx-auto size-10 text-warning" aria-hidden="true" />
          <h2 class="m-0 mt-4 text-lg font-extrabold">
            {{ t('user.verification.student.platform.mobileCamera.expiredTitle') }}
          </h2>
          <p class="m-0 mt-2 text-sm leading-6 text-text-muted">
            {{ t('user.verification.student.platform.mobileCamera.expiredDescription') }}
          </p>
        </section>

        <section v-else class="py-8 text-center" role="alert">
          <CircleAlert class="mx-auto size-10 text-danger" aria-hidden="true" />
          <h2 class="m-0 mt-4 text-lg font-extrabold">
            {{ t('user.verification.student.platform.mobileCamera.errorTitle') }}
          </h2>
          <p class="m-0 mt-2 text-sm leading-6 text-text-muted">{{ errorMessage }}</p>
        </section>

        <p v-if="errorMessage && state !== 'error'" class="m-0 mt-4 rounded-lg bg-danger/10 p-3 text-sm text-danger" role="alert">
          {{ errorMessage }}
        </p>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  BadgeCheck,
  Camera,
  CircleAlert,
  LoaderCircle,
  MonitorCheck,
  ScanLine,
  ShieldCheck,
  TimerOff,
} from 'lucide-vue-next'
import type { ApiCallResult, ManualCameraHandoff } from '@stuhelper/shared/api'
import { extractResultData } from '@stuhelper/shared/api'

import { api } from '@/api'
import { getErrorMessage, getErrorStatus } from '@/api/errors'
import {
  captureFrameAsBase64,
  describeCameraCaptureError,
  startCameraStream,
  stopCameraStream,
  supportsCameraCapture,
} from '@/utils/cameraCapture'

type PageState = 'loading' | 'ready' | 'uploaded' | 'desktop' | 'expired' | 'error'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const token = computed(() => String(route.params.token ?? ''))
const state = ref<PageState>('loading')
const handoff = ref<ManualCameraHandoff | null>(null)
const videoRef = ref<HTMLVideoElement | null>(null)
const stream = ref<MediaStream | null>(null)
const preview = ref('')
const uploading = ref(false)
const choosing = ref(false)
const cameraSupported = ref(supportsCameraCapture())
const errorMessage = ref('')
let loadSequence = 0

watch(token, (value) => void loadHandoff(value), { immediate: true })

async function readPayload<T>(request: Promise<unknown>, fallback: string): Promise<T> {
  const result = await request as ApiCallResult<T>
  const data = extractResultData(result)
  if (data === undefined) throw new Error(fallback)
  return data
}

async function loadHandoff(value: string): Promise<void> {
  const sequence = ++loadSequence
  state.value = 'loading'
  errorMessage.value = ''
  stopCameraStream(stream.value)
  stream.value = null
  try {
    const current = await readPayload<ManualCameraHandoff>(
      api.studentVerification.previewManualCameraHandoff(value),
      'Unable to open camera handoff',
    )
    if (sequence !== loadSequence) return
    handoff.value = current
    applyHandoffState(current)
  } catch (error) {
    if (sequence !== loadSequence) return
    if (getErrorStatus(error) === 410) state.value = 'expired'
    else state.value = 'error'
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.mobileCamera.errorDescription'))
  }
}

function applyHandoffState(current: ManualCameraHandoff): void {
  if (current.status === 'pending') state.value = 'ready'
  else if (current.status === 'uploaded') state.value = 'uploaded'
  else if (current.status === 'locked' && current.continueOn === 'desktop') state.value = 'desktop'
  else if (current.status === 'locked' && current.continueOn === 'mobile') {
    void router.replace({ path: '/user/student-verification', query: { handoff: token.value } })
  } else state.value = 'expired'
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
    errorMessage.value = describeCameraCaptureError(error, t('user.verification.student.platform.mobileCamera.cameraError'))
  }
}

async function captureAndUpload(): Promise<void> {
  if (!videoRef.value || uploading.value) return
  uploading.value = true
  errorMessage.value = ''
  try {
    const frame = captureFrameAsBase64(videoRef.value, { maxBytes: handoff.value?.maxMaterialBytes })
    preview.value = `data:${frame.contentType};base64,${frame.imageBase64}`
    handoff.value = await readPayload<ManualCameraHandoff>(
      api.studentVerification.uploadManualHandoffCameraCapture(token.value, {
        ...frame,
        captureSource: 'web_camera',
        requestedFacingMode: 'environment',
      }),
      'Unable to upload camera material',
    )
    stopCameraStream(stream.value)
    stream.value = null
    applyHandoffState(handoff.value)
  } catch (error) {
    errorMessage.value = describeCameraCaptureError(error, t('user.verification.student.platform.mobileCamera.uploadError'))
  } finally {
    uploading.value = false
  }
}

async function chooseContinuation(continueOn: 'desktop' | 'mobile'): Promise<void> {
  if (choosing.value) return
  choosing.value = true
  errorMessage.value = ''
  try {
    handoff.value = await readPayload<ManualCameraHandoff>(
      api.studentVerification.chooseManualCameraContinuation(token.value, { continueOn }),
      'Unable to choose continuation device',
    )
    if (continueOn === 'mobile') {
      await router.replace({ path: '/user/student-verification', query: { handoff: token.value } })
    } else {
      state.value = 'desktop'
    }
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('user.verification.student.platform.mobileCamera.choiceError'))
  } finally {
    choosing.value = false
  }
}

onBeforeUnmount(() => {
  loadSequence += 1
  stopCameraStream(stream.value)
})
</script>

<style scoped>
@reference "../../../styles/tailwind.css";

.handoff-shell {
  min-height: 100dvh;
  padding-top: max(1.5rem, env(safe-area-inset-top));
  padding-bottom: max(1.5rem, env(safe-area-inset-bottom));
}

.handoff-primary {
  @apply inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border-0 bg-primary px-5 py-2.5 text-sm font-bold text-white transition-colors hover:bg-primary-dark focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-bg-card disabled:cursor-not-allowed disabled:opacity-50;
}

.handoff-secondary {
  @apply inline-flex min-h-11 w-full items-center justify-center gap-2 rounded-lg border border-border bg-bg-card px-4 py-2.5 text-sm font-semibold text-text-primary transition-colors hover:border-primary hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 disabled:cursor-not-allowed disabled:opacity-50;
}
</style>
