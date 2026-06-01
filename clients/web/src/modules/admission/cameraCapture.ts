import type { CameraCaptureRequest } from '@stuhelper/shared/api'

const DEFAULT_CAPTURE_CONTENT_TYPE = 'image/jpeg'
const DEFAULT_CAPTURE_QUALITY = 0.92
const BASE64_CHARS_PER_CHUNK = 4
const BASE64_BYTES_PER_CHUNK = 3
const CANVAS_CONTEXT_2D = '2d'

type CameraNavigator = {
  readonly mediaDevices?: {
    readonly getUserMedia?: MediaDevices['getUserMedia']
  }
}

export interface CameraCaptureOptions {
  readonly contentType?: CameraCaptureRequest['contentType']
  readonly maxBytes?: number
  readonly quality?: number
}

export function supportsCameraCapture(source: CameraNavigator = navigator): boolean {
  return typeof source.mediaDevices?.getUserMedia === 'function'
}

export function buildCameraConstraints(): MediaStreamConstraints {
  return {
    audio: false,
    video: {
      facingMode: { ideal: 'environment' },
    },
  }
}

export async function startCameraStream(
  mediaDevices: MediaDevices = navigator.mediaDevices,
): Promise<MediaStream> {
  if (!supportsCameraCapture({ mediaDevices })) {
    throw new Error('Camera capture requires getUserMedia support')
  }
  return mediaDevices.getUserMedia(buildCameraConstraints())
}

export function describeCameraCaptureError(error: unknown, fallback: string): string {
  const message = error instanceof Error ? error.message : ''
  const name = error instanceof DOMException || error instanceof Error ? error.name : ''
  if (name === 'NotAllowedError' || /permission|denied/i.test(message)) {
    return '摄像头权限被浏览器拒绝。请在地址栏站点权限中允许摄像头；如果仍然失败，请确认当前页面使用 HTTPS 且没有被浏览器策略禁止摄像头。'
  }
  if (name === 'NotFoundError' || name === 'DevicesNotFoundError') {
    return '当前设备没有可用摄像头。可以改用另一台设备打开手机拍照链接。'
  }
  if (name === 'NotReadableError' || name === 'TrackStartError') {
    return '摄像头暂时无法读取，可能正被其他应用占用。请关闭占用摄像头的应用后重试。'
  }
  if (name === 'OverconstrainedError' || name === 'ConstraintNotSatisfiedError') {
    return '当前摄像头不满足拍照要求。请尝试切换设备或使用手机拍照链接。'
  }
  if (/getUserMedia support/i.test(message)) {
    return '当前浏览器不支持直接拍照。请使用支持摄像头权限的浏览器，或改用手机拍照链接。'
  }
  if (/video frame is not ready/i.test(message)) {
    return '摄像头画面还没有准备好，请等待画面出现后再拍摄。'
  }
  return message || fallback
}

export function stopCameraStream(stream: MediaStream | null | undefined): void {
  for (const track of stream?.getTracks() ?? []) {
    track.stop()
  }
}

export function captureFrameAsBase64(
  video: HTMLVideoElement,
  options: CameraCaptureOptions = {},
): CameraCaptureRequest {
  const width = video.videoWidth
  const height = video.videoHeight
  if (width <= 0 || height <= 0) {
    throw new Error('Camera video frame is not ready')
  }

  const contentType = options.contentType ?? DEFAULT_CAPTURE_CONTENT_TYPE
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext(CANVAS_CONTEXT_2D)
  if (!context) {
    throw new Error('Canvas 2D context unavailable')
  }
  context.drawImage(video, 0, 0, width, height)

  const dataURL = canvas.toDataURL(
    contentType,
    options.quality ?? DEFAULT_CAPTURE_QUALITY,
  )
  const imageBase64 = readImageBase64(dataURL)
  assertWithinMaxBytes(imageBase64, options.maxBytes)

  return {
    contentType,
    imageBase64,
    capturedAt: new Date().toISOString(),
  }
}

function readImageBase64(dataURL: string): string {
  const separatorIndex = dataURL.indexOf(',')
  if (separatorIndex < 0) {
    throw new Error('Camera capture did not produce a data URL')
  }
  return dataURL.slice(separatorIndex + 1)
}

function assertWithinMaxBytes(imageBase64: string, maxBytes?: number): void {
  if (maxBytes === undefined) return
  if (estimateBase64Bytes(imageBase64) > maxBytes) {
    throw new Error('Camera capture exceeds the admission material size limit')
  }
}

function estimateBase64Bytes(value: string): number {
  const padding = value.endsWith('==') ? 2 : value.endsWith('=') ? 1 : 0
  return (value.length / BASE64_CHARS_PER_CHUNK) * BASE64_BYTES_PER_CHUNK - padding
}
