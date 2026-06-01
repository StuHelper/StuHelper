const QQ_ADMISSION_PLATFORM = 'qq'
const SUPPORTED_QQ_RUNTIME_PLATFORMS = new Set(['qq', 'onebot'])

export type AdmissionSubjectPlatform = typeof QQ_ADMISSION_PLATFORM

// The backend admission schema stores the subject account platform (`qq` today),
// while Koishi stores the bot adapter platform (`onebot` in NapCat production).
// Keep this mapping explicit so a future official QQ bot adapter can be handled
// deliberately instead of leaking adapter names into admission records.
export function resolveAdmissionSubjectPlatform(
  runtimePlatform: string | undefined,
): AdmissionSubjectPlatform | undefined {
  if (!runtimePlatform || !SUPPORTED_QQ_RUNTIME_PLATFORMS.has(runtimePlatform)) {
    return undefined
  }
  return QQ_ADMISSION_PLATFORM
}

export function requireAdmissionSubjectPlatform(runtimePlatform: string | undefined) {
  const platform = resolveAdmissionSubjectPlatform(runtimePlatform)
  if (!platform) {
    throw new Error(`admission worker requires qq-compatible platform, got ${runtimePlatform || 'unknown'}`)
  }
  return platform
}
