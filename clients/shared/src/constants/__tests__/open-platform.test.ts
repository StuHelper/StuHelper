import { describe, expect, it } from 'vitest'

import {
  OPEN_PLATFORM_AUDIT_EVENT_TYPES,
  type OpenPlatformAuditEventType,
} from '../open-platform'

describe('Open Platform audit event constants', () => {
  it('tracks lifecycle and resource access audit events used by UI filters', () => {
    const eventTypes: OpenPlatformAuditEventType[] = [
      'open_platform.app.resumed',
      'open_platform.app.approved_app_ensured',
      'open_platform.resource_access.checked',
      'open_platform.resource_access.granted',
      'open_platform.resource_access.revoked',
    ]

    expect(OPEN_PLATFORM_AUDIT_EVENT_TYPES).toEqual(
      expect.arrayContaining(eventTypes),
    )
  })
})
