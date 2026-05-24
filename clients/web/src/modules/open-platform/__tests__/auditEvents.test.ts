import { OPEN_PLATFORM_AUDIT_EVENT_TYPES } from '@stuhelper/shared/constants'
import { describe, expect, it } from 'vitest'

import enUSDeveloper from '@/i18n/locales/en-US/developer'
import zhCNDeveloper from '@/i18n/locales/zh-CN/developer'
import {
  developerOpenPlatformAuditTypeKey,
  developerOpenPlatformAuditTypeKeys,
} from '../auditEvents'

describe('developer Open Platform audit event labels', () => {
  it('provides a translation key for every shared Open Platform audit event', () => {
    for (const eventType of OPEN_PLATFORM_AUDIT_EVENT_TYPES) {
      expect(developerOpenPlatformAuditTypeKeys[eventType], eventType).toMatch(
        /^developer\.apps\.auditTypes\./,
      )
    }
  })

  it('keeps developer audit event labels present in Chinese and English locales', () => {
    for (const key of Object.values(developerOpenPlatformAuditTypeKeys)) {
      const auditTypeKey = key.replace('developer.apps.auditTypes.', '')
      expect(
        zhCNDeveloper.apps.auditTypes[
          auditTypeKey as keyof typeof zhCNDeveloper.apps.auditTypes
        ],
        `zh-CN ${key}`,
      ).toEqual(expect.any(String))
      expect(
        enUSDeveloper.apps.auditTypes[
          auditTypeKey as keyof typeof enUSDeveloper.apps.auditTypes
        ],
        `en-US ${key}`,
      ).toEqual(expect.any(String))
    }
  })

  it('labels resumed app and resource access audit events', () => {
    expect(developerOpenPlatformAuditTypeKey('open_platform.app.resumed')).toBe(
      'developer.apps.auditTypes.appResumed',
    )
    expect(
      developerOpenPlatformAuditTypeKey('open_platform.app.approved_app_ensured'),
    ).toBe('developer.apps.auditTypes.approvedAppEnsured')
    expect(
      developerOpenPlatformAuditTypeKey(
        'open_platform.app.identity_public_smoke_bootstrapped',
      ),
    ).toBe('developer.apps.auditTypes.identityPublicSmokeBootstrapped')
    expect(
      developerOpenPlatformAuditTypeKey('open_platform.resource_access.granted'),
    ).toBe('developer.apps.auditTypes.resourceGranted')
    expect(
      developerOpenPlatformAuditTypeKey('open_platform.resource_access.revoked'),
    ).toBe('developer.apps.auditTypes.resourceRevoked')
  })

  it('returns null for unknown future audit event types', () => {
    expect(developerOpenPlatformAuditTypeKey('open_platform.future.event')).toBe(
      null,
    )
  })
})
