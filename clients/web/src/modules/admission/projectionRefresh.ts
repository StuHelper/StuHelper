export const ADMISSION_PROJECTED_CAPABILITY = 'review:create'
export const ADMISSION_PROJECTION_RETRY_DELAYS_MS = [
  1000,
  2000,
  4000,
  8000,
  16_000,
] as const

export type ProjectionUser = {
  capabilities?: readonly string[]
  globalCapabilities?: readonly string[]
}

type ProjectionRefreshOptions<TUser extends ProjectionUser> = {
  signal?: AbortSignal
  hasProjectedCapability?: (user: TUser) => boolean
  refreshAuth: () => Promise<TUser>
  retryDelays?: readonly number[]
  wait?: (delayMs: number, signal?: AbortSignal) => Promise<void>
}

export async function waitForAdmissionProjection<TUser extends ProjectionUser>({
  signal,
  hasProjectedCapability = userHasAdmissionProjection,
  refreshAuth,
  retryDelays = ADMISSION_PROJECTION_RETRY_DELAYS_MS,
  wait = waitDelay,
}: ProjectionRefreshOptions<TUser>): Promise<boolean> {
  for (const delay of retryDelays) {
    throwIfAborted(signal)
    await wait(delay, signal)
    throwIfAborted(signal)
    const user = await refreshAuth()
    if (hasProjectedCapability(user)) return true
  }

  return false
}

export function userHasAdmissionProjection(user: ProjectionUser): boolean {
  return user.capabilities?.includes(ADMISSION_PROJECTED_CAPABILITY) === true ||
    user.globalCapabilities?.includes(ADMISSION_PROJECTED_CAPABILITY) === true
}

function waitDelay(delayMs: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(admissionProjectionAbortError())
      return
    }
    const timeout = setTimeout(() => {
      signal?.removeEventListener('abort', abort)
      resolve()
    }, delayMs)
    const abort = () => {
      clearTimeout(timeout)
      reject(admissionProjectionAbortError())
    }
    signal?.addEventListener('abort', abort, { once: true })
  })
}

function throwIfAborted(signal?: AbortSignal): void {
  if (!signal?.aborted) return
  throw admissionProjectionAbortError()
}

function admissionProjectionAbortError(): DOMException {
  return new DOMException('Admission projection refresh aborted', 'AbortError')
}
