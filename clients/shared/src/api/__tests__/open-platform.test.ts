import { describe, expect, it, vi } from 'vitest'

import { createOpenPlatformApi } from '../open-platform'
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

describe('createOpenPlatformApi', () => {
  it('lists current user apps with lifecycle filters', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.listApps({ page: 2, pageSize: 10, status: 'pending' })

    expect(client.GET).toHaveBeenCalledWith('/api/v1/open-platform/apps', {
      params: { query: { page: 2, pageSize: 10, status: 'pending' } },
    })
  })

  it('lists developer app audit events with app-scoped filters', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.listAppAuditEvents({
      appID: 42,
      params: {
        eventType: 'open_platform.app.secret_rotated',
        scope: 'email.read',
        page: 2,
        pageSize: 10,
      },
    })

    expect(client.GET).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}/audit-events', {
      params: {
        path: { appID: 42 },
        query: {
          eventType: 'open_platform.app.secret_rotated',
          scope: 'email.read',
          page: 2,
          pageSize: 10,
        },
      },
    })
  })

  it('registers developer apps', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)
    const payload = {
      displayName: 'Campus Tools',
      homepageURL: 'https://tools.example.com',
      privacyPolicyURL: 'https://tools.example.com/privacy',
      redirectURIs: ['https://tools.example.com/callback'],
      scopes: [{ scope: 'profile.basic.read' as const, reason: 'login' }],
    }

    api.registerApp(payload)

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/apps', {
      body: payload,
    })
  })

  it('updates developer app profile metadata', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)
    const profile = {
      displayName: 'Campus Tools v2',
      description: 'Updated app listing.',
      homepageURL: 'https://tools-v2.example.com',
      privacyPolicyURL: 'https://tools-v2.example.com/privacy',
      reason: 'public listing update',
    }

    api.updateAppProfile({ appID: 42, profile })

    expect(client.PATCH).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}', {
      body: profile,
      params: { path: { appID: 42 } },
    })
  })

  it('submits developer scope change requests', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.requestScopeChange({
      appID: 42,
      scopes: [{ scope: 'email.read', reason: 'send account security notices' }],
    })

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}/scopes', {
      body: {
        scopes: [{ scope: 'email.read', reason: 'send account security notices' }],
      },
      params: { path: { appID: 42 } },
    })
  })

  it('withdraws developer scope requests', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.withdrawScopeRequest({
      appID: 42,
      scope: 'email.read',
      reason: 'need to narrow the requested purpose',
    })

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/open-platform/apps/{appID}/scopes/{scope}/withdraw',
      {
        body: { reason: 'need to narrow the requested purpose' },
        params: { path: { appID: 42, scope: 'email.read' } },
      },
    )
  })

  it('rotates developer app secrets', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.rotateAppSecret({ appID: 42, reason: 'scheduled rotation' })

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}/secret/rotate', {
      body: { reason: 'scheduled rotation' },
      params: { path: { appID: 42 } },
    })
  })

  it('submits developer redirect URI change requests', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.requestRedirectURIChange({
      appID: 42,
      redirectURIs: ['https://app.example.com/oauth/callback'],
      reason: 'move callback host',
    })

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}/redirect-uris', {
      body: {
        redirectURIs: ['https://app.example.com/oauth/callback'],
        reason: 'move callback host',
      },
      params: { path: { appID: 42 } },
    })
  })

  it('withdraws developer redirect URI change requests', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.withdrawRedirectURIRequest({
      appID: 42,
      requestID: 9,
      reason: 'domain verification is not ready',
    })

    expect(client.POST).toHaveBeenCalledWith(
      '/api/v1/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/withdraw',
      {
        body: { reason: 'domain verification is not ready' },
        params: { path: { appID: 42, requestID: 9 } },
      },
    )
  })

  it('withdraws pending developer apps', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.withdrawApp({
      appID: 42,
      reason: 'submitted by mistake',
    })

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/apps/{appID}/withdraw', {
      body: { reason: 'submitted by mistake' },
      params: { path: { appID: 42 } },
    })
  })

  it('lists current user consents', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.listConsents()

    expect(client.GET).toHaveBeenCalledWith('/api/v1/open-platform/consents')
  })

  it('lists current user consent audit events with filters', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.listConsentAuditEvents({
      appID: 12,
      eventType: 'open_platform.consent.revoked',
      scope: 'email.read',
      page: 2,
      pageSize: 10,
    })

    expect(client.GET).toHaveBeenCalledWith('/api/v1/open-platform/consents/audit-events', {
      params: {
        query: {
          appID: 12,
          eventType: 'open_platform.consent.revoked',
          scope: 'email.read',
          page: 2,
          pageSize: 10,
        },
      },
    })
  })

  it('revokes all consents for an app when scopes are absent', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.revokeConsent({ appID: 12 })

    expect(client.DELETE).toHaveBeenCalledWith('/api/v1/open-platform/consents/{appID}', {
      params: {
        path: { appID: 12 },
        query: {},
      },
    })
  })

  it('passes repeated scope query values for scoped revocation', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.revokeConsent({
      appID: 12,
      scopes: ['profile.basic.read', 'email.read'],
    })

    expect(client.DELETE).toHaveBeenCalledWith('/api/v1/open-platform/consents/{appID}', {
      params: {
        path: { appID: 12 },
        query: { scope: ['profile.basic.read', 'email.read'] },
      },
    })
  })

  it('checks app resource access with client credentials', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.checkResourceAccess({
      clientID: 'app-client',
      clientSecret: 'secret',
      resourceType: 'resource_item',
      resourceID: 'resource-42',
      action: 'read',
    })

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/resources/access/check', {
      body: {
        clientID: 'app-client',
        clientSecret: 'secret',
        resourceType: 'resource_item',
        resourceID: 'resource-42',
        action: 'read',
      },
    })
  })

  it('checks app resource access with an app-only bearer token', () => {
    const client = createMockClient()
    const api = createOpenPlatformApi(client)

    api.checkResourceAccess(
      {
        resourceType: 'resource_item',
        resourceID: 'resource-42',
        action: 'read',
      },
      { accessToken: 'resource-access-token' },
    )

    expect(client.POST).toHaveBeenCalledWith('/api/v1/open-platform/resources/access/check', {
      body: {
        resourceType: 'resource_item',
        resourceID: 'resource-42',
        action: 'read',
      },
      headers: { Authorization: 'Bearer resource-access-token' },
    })
  })
})
