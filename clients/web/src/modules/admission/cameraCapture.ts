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
