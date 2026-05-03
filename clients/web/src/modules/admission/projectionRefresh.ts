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
  hasProjectedCapability?: (user: TUser) => boolean
  refreshAuth: () => Promise<TUser>
  retryDelays?: readonly number[]
  wait?: (delayMs: number) => Promise<void>
}

export async function waitForAdmissionProjection<TUser extends ProjectionUser>({
  hasProjectedCapability = userHasAdmissionProjection,
  refreshAuth,
  retryDelays = ADMISSION_PROJECTION_RETRY_DELAYS_MS,
  wait = waitDelay,
}: ProjectionRefreshOptions<TUser>): Promise<boolean> {
  for (const delay of retryDelays) {
    await wait(delay)
    const user = await refreshAuth()
    if (hasProjectedCapability(user)) return true
  }

  return false
}

export function userHasAdmissionProjection(user: ProjectionUser): boolean {
  return user.capabilities?.includes(ADMISSION_PROJECTED_CAPABILITY) === true ||
    user.globalCapabilities?.includes(ADMISSION_PROJECTED_CAPABILITY) === true
}

function waitDelay(delayMs: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, delayMs))
}
