const DEFAULT_CAPTURE_CONTENT_TYPE = 'image/jpeg' as const
const DEFAULT_CAPTURE_QUALITY = 0.92
const BASE64_CHARS_PER_CHUNK = 4
const BASE64_BYTES_PER_CHUNK = 3

type CameraNavigator = {
  readonly mediaDevices?: {
    readonly getUserMedia?: MediaDevices['getUserMedia']
  }
}

export interface CameraFrame {
  contentType: 'image/jpeg' | 'image/png' | 'image/webp'
  imageBase64: string
}

export interface CameraCaptureOptions {
  readonly contentType?: CameraFrame['contentType']
  readonly maxBytes?: number
  readonly quality?: number
}

export function supportsCameraCapture(source: CameraNavigator = navigator): boolean {
  return typeof source.mediaDevices?.getUserMedia === 'function'
}

export function buildCameraConstraints(): MediaStreamConstraints {
  return {
    audio: false,
    video: { facingMode: { ideal: 'environment' } },
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
  for (const track of stream?.getTracks() ?? []) track.stop()
}

export function captureFrameAsBase64(
  video: HTMLVideoElement,
  options: CameraCaptureOptions = {},
): CameraFrame {
  const width = video.videoWidth
  const height = video.videoHeight
  if (width <= 0 || height <= 0) throw new Error('Camera video frame is not ready')

  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const context = canvas.getContext('2d')
  if (!context) throw new Error('Canvas 2D context unavailable')
  context.drawImage(video, 0, 0, width, height)

  const contentType = options.contentType ?? DEFAULT_CAPTURE_CONTENT_TYPE
  const dataURL = canvas.toDataURL(contentType, options.quality ?? DEFAULT_CAPTURE_QUALITY)
  const separatorIndex = dataURL.indexOf(',')
  if (separatorIndex < 0) throw new Error('Camera capture did not produce a data URL')
  const imageBase64 = dataURL.slice(separatorIndex + 1)
  if (options.maxBytes !== undefined && estimateBase64Bytes(imageBase64) > options.maxBytes) {
    throw new Error('Camera capture exceeds the material size limit')
  }
  return { contentType, imageBase64 }
}

export function describeCameraCaptureError(error: unknown, fallback: string): string {
  const message = error instanceof Error ? error.message : ''
  const name = error instanceof Error ? error.name : ''
  if (name === 'NotAllowedError' || /permission|denied/i.test(message)) {
    return '摄像头权限被浏览器拒绝。请允许本站使用摄像头，或改用手机扫码拍摄。'
  }
  if (name === 'NotFoundError' || name === 'DevicesNotFoundError') {
    return '当前设备没有可用摄像头，请改用手机扫码拍摄。'
  }
  if (name === 'NotReadableError' || name === 'TrackStartError') {
    return '摄像头暂时无法读取。请关闭占用摄像头的应用后重试。'
  }
  if (/getUserMedia support/i.test(message)) {
    return '当前浏览器不支持直接拍照，请使用手机扫码拍摄。'
  }
  if (/video frame is not ready/i.test(message)) {
    return '摄像头画面尚未准备好，请稍候再拍。'
  }
  if (/material size limit/i.test(message)) {
    return '照片超过材料大小限制，请调整距离后重新拍摄。'
  }
  return message || fallback
}

function estimateBase64Bytes(value: string): number {
  const padding = value.endsWith('==') ? 2 : value.endsWith('=') ? 1 : 0
  return (value.length / BASE64_CHARS_PER_CHUNK) * BASE64_BYTES_PER_CHUNK - padding
}
