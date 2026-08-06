import { describe, expect, it, vi } from 'vitest'

import type { ApiClient } from '../client'
import {
  createStudentVerificationAdminApi,
  createStudentVerificationApi,
} from '../student-verification'

function createMockClient(): ApiClient {
  return {
    DELETE: vi.fn(),
    GET: vi.fn(),
    PATCH: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  } as unknown as ApiClient
}

describe('createStudentVerificationApi', () => {
  it('keeps application lifecycle independent from admission sessions', () => {
    const client = createMockClient()
    const api = createStudentVerificationApi(client)

    api.createApplication({ schoolCode: '4111010006' })
    api.cancelApplication('11111111-1111-4111-8111-111111111111')

    expect(client.POST).toHaveBeenCalledWith('/api/v1/student-verification/applications', {
      body: { schoolCode: '4111010006' },
    })
    expect(client.DELETE).toHaveBeenCalledWith(
      '/api/v1/student-verification/applications/{applicationID}',
      { params: { path: { applicationID: '11111111-1111-4111-8111-111111111111' } } },
    )
  })

  it('submits sensitive evidence only in the method request body', () => {
    const client = createMockClient()
    const api = createStudentVerificationApi(client)
    const applicationID = '11111111-1111-4111-8111-111111111111'
    const body = {
      studentID: '20990001',
      name: '测试用户',
      documentNumber: '11010519491231002X',
      privacyNoticeVersion: '2026-08-05',
      sensitiveDataConsent: true as const,
    }

    api.verifyRealName(applicationID, body)

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/student-verification/applications/{applicationID}/real-name/verify',
      { params: { path: { applicationID } }, body },
    )
  })

  it('keeps phone operations explicit and user initiated', () => {
    const client = createMockClient()
    const api = createStudentVerificationApi(client)

    api.createPhoneOperation({ phone: '13800138000' })
    api.verifyPhoneSMS('22222222-2222-4222-8222-222222222222', { code: '123456' })

    expect(client.POST).toHaveBeenNthCalledWith(1, '/api/v1/account/phone/operations', {
      body: { phone: '13800138000' },
    })
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/account/phone/operations/{operationID}/sms/verify',
      {
        params: { path: { operationID: '22222222-2222-4222-8222-222222222222' } },
        body: { code: '123456' },
      },
    )
  })
})

describe('createStudentVerificationAdminApi', () => {
  it('creates a disabled school profile separately from the school directory', () => {
    const client = createMockClient()
    const api = createStudentVerificationAdminApi(client)
    const body = {
      schoolCode: '4111019999',
      adapterID: 'declarative',
      adapterVersion: '1',
      emailDomains: ['example.edu.cn'],
      studentIDPolicy: { strategy: 'regex', pattern: '^[0-9]{8}$' },
      nameMatchPolicy: { strategy: 'exact_trimmed' },
      enrollmentPolicy: {
        rosterKnownEligibilityCodes: ['eligible', 'ineligible'],
        rosterEligibleCodes: ['eligible'],
        rosterMinimumRows: 1,
        rosterMaximumRowDeltaRatio: 0.25,
      },
      manualFormSchema: {},
      snapshotAutoActivate: false,
      snapshotSyncIntervalSeconds: 21_600,
      snapshotWarningAfterSeconds: 43_200,
      snapshotHardExpirySeconds: 172_800,
      snapshotGraceSeconds: 0,
      reason: '新增学校认证配置',
    }

    api.createSchool(body)

    expect(client.POST).toHaveBeenCalledWith('/api/v1/admin/student-verification/schools', {
      body,
    })
  })

  it('requires explicit revisions for credential revocation and snapshot activation', () => {
    const client = createMockClient()
    const api = createStudentVerificationAdminApi(client)

    api.revokeCredential('33333333-3333-4333-8333-333333333333', {
      expectedRevision: 3,
      reason: '主体冲突复核后撤销',
    })
    api.activateRosterSnapshot(
      '4111010006',
      '44444444-4444-4444-8444-444444444444',
      { allowSourceRegression: false, reason: '质量门禁通过并批准激活' },
    )

    expect(client.POST).toHaveBeenNthCalledWith(
      1,
      '/api/v1/admin/student-verification/credentials/{credentialID}/revoke',
      {
        params: {
          path: { credentialID: '33333333-3333-4333-8333-333333333333' },
        },
        body: { expectedRevision: 3, reason: '主体冲突复核后撤销' },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-snapshots/{snapshotID}/activate',
      {
        params: {
          path: {
            schoolCode: '4111010006',
            snapshotID: '44444444-4444-4444-8444-444444444444',
          },
        },
        body: { allowSourceRegression: false, reason: '质量门禁通过并批准激活' },
      },
    )
  })
})
