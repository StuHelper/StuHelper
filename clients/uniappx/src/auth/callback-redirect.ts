import { consumeStoredSSORedirect } from '@/auth/sso-state'

export const NATIVE_SSO_REDIRECT_DELAY_MS = 500

type Relaunch = (options: { url: string }) => void
type Schedule = (callback: () => void, delay: number) => unknown

export function scheduleNativeSSORedirect(
  relaunch: Relaunch,
  schedule: Schedule = setTimeout,
): void {
  const target = consumeStoredSSORedirect()
  schedule(() => relaunch({ url: target }), NATIVE_SSO_REDIRECT_DELAY_MS)
}
