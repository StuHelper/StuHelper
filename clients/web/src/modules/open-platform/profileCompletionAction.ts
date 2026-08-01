import type { OpenPlatformProfileCompletionPageResponse } from '@stuhelper/shared/api'
import { accountCenterURLForHref } from '@/utils/redirect'

type ProfileCompletionField =
  OpenPlatformProfileCompletionPageResponse['missingFields'][number]

const IDENTITY_PROVIDER_FIELDS = new Set([
  'profile.username',
  'profile.email',
  'profile.avatar',
])

export function resolveProfileCompletionActionURL(
  field: Pick<ProfileCompletionField, 'key' | 'actionURL'>,
  accountSettingsURL?: string,
): string {
  if (accountSettingsURL && IDENTITY_PROVIDER_FIELDS.has(field.key)) {
    return accountSettingsURL
  }
  return accountCenterURLForHref(field.actionURL) ?? field.actionURL
}
