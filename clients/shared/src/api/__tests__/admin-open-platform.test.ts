import { describe, expect, it, vi } from 'vitest'

import { createAdminApi } from '../admin'
import type { ApiClient } from '../client'

function createMockClient(): ApiClient {
  return {
    DELETE: vi.fn(),
    GET: vi.fn(),
    PATCH: vi.fn(),
    POST: vi.fn(),
    PUT: vi.fn(),
  } as unknown as ApiClient
}

describe('createAdminApi open platform lifecycle methods', () => {
  it('imports legacy Casdoor applications', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.importOpenPlatformCasdoorApp({
      casdoorApplicationName: 'legacy-app',
      privacyPolicyURL: 'https://legacy.example.com/privacy',
      redirectURIs: ['https://legacy.example.com/callback'],
      scopes: [{ scope: 'profile.basic.read', reason: 'legacy login' }],
    })

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/apps/import-casdoor',
      {
        body: {
          casdoorApplicationName: 'legacy-app',
          privacyPolicyURL: 'https://legacy.example.com/privacy',
          redirectURIs: ['https://legacy.example.com/callback'],
          scopes: [{ scope: 'profile.basic.read', reason: 'legacy login' }],
        },
      },
    )
  })

  it('lists audit events with filters', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.getOpenPlatformAuditEvents({
      appID: 12,
      eventType: 'open_platform.consent.granted',
      page: 2,
      pageSize: 50,
      scope: 'profile.basic.read',
      userID: 34,
    })

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/audit-events',
      {
        params: {
          query: {
            appID: 12,
            eventType: 'open_platform.consent.granted',
            page: 2,
            pageSize: 50,
            scope: 'profile.basic.read',
            userID: 34,
          },
        },
      },
    )
  })

  it('lists and revokes active user consents through admin endpoints', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.getOpenPlatformConsents({
      appID: 12,
      page: 2,
      pageSize: 50,
      userID: 34,
    })
    api.revokeOpenPlatformConsent(12, {
      reason: 'privacy incident response',
      scopes: ['email.read'],
      userID: 34,
    })

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/consents',
      {
        params: {
          query: {
            appID: 12,
            page: 2,
            pageSize: 50,
            userID: 34,
          },
        },
      },
    )
    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/apps/{appID}/consents/revoke',
      {
        body: {
          reason: 'privacy incident response',
          scopes: ['email.read'],
          userID: 34,
        },
        params: { path: { appID: 12 } },
      },
    )
  })

  it('lists token probe evidence with filters', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.getOpenPlatformTokenProbeEvidence({
      appID: 12,
      clientID: 'client-12',
      page: 2,
      pageSize: 50,
      result: 'failed',
      reviewerUserID: 34,
    })

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/token-probe-evidence',
      {
        params: {
          query: {
            appID: 12,
            clientID: 'client-12',
            page: 2,
            pageSize: 50,
            result: 'failed',
            reviewerUserID: 34,
          },
        },
      },
    )
  })

  it('loads the disclosure operations report', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.getOpenPlatformDisclosureReport({ windowHours: 48 })

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/disclosure-report',
      {
        params: {
          query: {
            windowHours: 48,
          },
        },
      },
    )
  })

  it('lists OpenFGA resource grants for an app and resource type', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.getOpenPlatformResourceGrants(12, 'resource_item')

    expect(client.GET).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/apps/{appID}/resource-grants',
      {
        params: {
          path: { appID: 12 },
          query: { resourceType: 'resource_item' },
        },
      },
    )
  })

  it('grants and revokes app resource access through the admin endpoints', () => {
    const client = createMockClient()
    const api = createAdminApi(client)
    const payload = {
      resourceType: 'resource_item' as const,
      resourceID: 'resource-42',
      actions: ['read' as const, 'write' as const],
      reason: 'dataset sharing approved',
    }

    api.grantOpenPlatformResourceAccess(12, payload)
    api.revokeOpenPlatformResourceAccess(12, payload)

    expect(client.POST).toHaveBeenNthCalledWith(
      1,
      '/api/v1/admin/open-platform/apps/{appID}/resource-grants',
      {
        body: payload,
        params: { path: { appID: 12 } },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/admin/open-platform/apps/{appID}/resource-grants/revoke',
      {
        body: payload,
        params: { path: { appID: 12 } },
      },
    )
  })

  it('reviews redirect URI change requests', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.approveOpenPlatformRedirectURIRequest(12, 3, { decisionNote: 'domain verified' })
    api.rejectOpenPlatformRedirectURIRequest(12, 4, { decisionNote: 'domain mismatch' })

    expect(client.POST).toHaveBeenNthCalledWith(
      1,
      '/api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/approve',
      {
        body: { decisionNote: 'domain verified' },
        params: { path: { appID: 12, requestID: 3 } },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/reject',
      {
        body: { decisionNote: 'domain mismatch' },
        params: { path: { appID: 12, requestID: 4 } },
      },
    )
  })

  it('reviews scope requests', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.approveOpenPlatformScope(12, 'profile.basic.read', { decisionNote: 'baseline profile is allowed' })
    api.rejectOpenPlatformScope(12, 'email.read', { decisionNote: 'purpose too broad' })

    expect(client.POST).toHaveBeenNthCalledWith(
      1,
      '/api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/approve',
      {
        body: { decisionNote: 'baseline profile is allowed' },
        params: { path: { appID: 12, scope: 'profile.basic.read' } },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/reject',
      {
        body: { decisionNote: 'purpose too broad' },
        params: { path: { appID: 12, scope: 'email.read' } },
      },
    )
  })

  it('rotates app secrets through the admin endpoint', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.rotateOpenPlatformAppSecret(12, { reason: 'leak response' })

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/admin/open-platform/apps/{appID}/secret/rotate',
      {
        body: { reason: 'leak response' },
        params: { path: { appID: 12 } },
      },
    )
  })

  it('suspends, resumes, and revokes apps with audit reasons', () => {
    const client = createMockClient()
    const api = createAdminApi(client)

    api.suspendOpenPlatformApp(12, { reason: 'risk review' })
    api.resumeOpenPlatformApp(12, { reason: 'risk cleared' })
    api.revokeOpenPlatformApp(13, { reason: 'owner request' })

    expect(client.POST).toHaveBeenNthCalledWith(
      1,
      '/api/v1/admin/open-platform/apps/{appID}/suspend',
      {
        body: { reason: 'risk review' },
        params: { path: { appID: 12 } },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      2,
      '/api/v1/admin/open-platform/apps/{appID}/resume',
      {
        body: { reason: 'risk cleared' },
        params: { path: { appID: 12 } },
      },
    )
    expect(client.POST).toHaveBeenNthCalledWith(
      3,
      '/api/v1/admin/open-platform/apps/{appID}/revoke',
      {
        body: { reason: 'owner request' },
        params: { path: { appID: 13 } },
      },
    )
  })
})
