import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAuthorizedAppsController } from '../authorizedAppsController'
import type {
  OpenPlatformUserAuthorizedApp,
  OpenPlatformUserConsentAuditEvent,
  OpenPlatformUserConsentScope,
} from '@stuhelper/shared/api'

const mocks = vi.hoisted(() => ({
  listConsents: vi.fn(),
  listConsentAuditEvents: vi.fn(),
  revokeConsent: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  getErrorMessage: vi.fn((_error: unknown, fallback: string) => fallback),
}))

vi.mock('@/api', () => ({
  api: {
    openPlatform: {
      listConsents: mocks.listConsents,
      listConsentAuditEvents: mocks.listConsentAuditEvents,
      revokeConsent: mocks.revokeConsent,
    },
  },
}))

vi.mock('@/api/errors', () => ({
  getErrorMessage: mocks.getErrorMessage,
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    success: mocks.toastSuccess,
    error: mocks.toastError,
  }),
}))

const t = (key: string, params?: Record<string, unknown>) => {
  if (!params) return key
  const suffix = Object.values(params).join(' ')
  return suffix ? `${key} ${suffix}` : key
}

describe('useAuthorizedAppsController', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getErrorMessage.mockImplementation((_error: unknown, fallback: string) => fallback)
    mocks.listConsents.mockResolvedValue(consentsResponse([
      authorizedApp([
        consentScope('profile.basic.read', 'Basic profile'),
        consentScope('email.read', 'Email address'),
      ]),
    ]))
    mocks.listConsentAuditEvents.mockResolvedValue(auditResponse([
      auditEvent('open_platform.consent.granted', ['profile.basic.read', 'email.read']),
    ]))
    mocks.revokeConsent.mockResolvedValue({})
  })

  it('loads authorized apps and recent consent activity', async () => {
    const controller = useAuthorizedAppsController(t)

    await controller.loadConsents()
    await controller.loadActivities()

    expect(controller.loading.value).toBe(false)
    expect(controller.activityLoading.value).toBe(false)
    expect(controller.apps.value).toHaveLength(1)
    expect(controller.apps.value[0]?.app.displayName).toBe('Campus Planner')
    expect(controller.apps.value[0]?.scopes.map(scope => scope.scope)).toEqual([
      'profile.basic.read',
      'email.read',
    ])
    expect(controller.activities.value).toHaveLength(1)
    expect(mocks.listConsentAuditEvents).toHaveBeenCalledWith({ pageSize: 10 })
  })

  it('fails closed when authorized apps response is malformed', async () => {
    mocks.listConsents.mockResolvedValueOnce({
      data: {
        data: null,
      },
    })

    const controller = useAuthorizedAppsController(t)

    await controller.loadConsents()

    expect(controller.loading.value).toBe(false)
    expect(controller.errorMessage.value).toBe('common.loadFailed')
    expect(controller.apps.value).toEqual([])
  })

  it('fails closed when authorization activity response is malformed', async () => {
    mocks.listConsentAuditEvents.mockResolvedValueOnce({
      data: {
        data: null,
      },
    })

    const controller = useAuthorizedAppsController(t)

    await controller.loadActivities()

    expect(controller.activityLoading.value).toBe(false)
    expect(controller.activityErrorMessage.value).toBe(
      'user.authorizedApps.activityLoadFailed',
    )
    expect(controller.activities.value).toEqual([])
  })

  it('revokes one granted scope without removing the whole app', async () => {
    const controller = useAuthorizedAppsController(t)
    await controller.loadConsents()

    const app = controller.apps.value[0]!
    const scope = app.scopes.find(item => item.scope === 'email.read')!
    controller.openRevokeScopeDialog(app, scope)

    expect(controller.revokeDialogTitle.value).toBe(
      'user.authorizedApps.revokeScopeDialogTitle Email address',
    )

    await controller.submitRevokeDialog()

    expect(mocks.revokeConsent).toHaveBeenCalledWith({
      appID: 12,
      scopes: ['email.read'],
    })
    expect(controller.revokeDialog.open).toBe(false)
    expect(controller.apps.value).toHaveLength(1)
    expect(controller.apps.value[0]?.scopes.map(item => item.scope)).toEqual([
      'profile.basic.read',
    ])
    expect(mocks.listConsentAuditEvents).toHaveBeenCalledTimes(1)
    expect(mocks.toastSuccess).toHaveBeenCalledWith('user.authorizedApps.revokeSuccess')
  })

  it('revokes all consents for an app and removes it from the list', async () => {
    const controller = useAuthorizedAppsController(t)
    await controller.loadConsents()

    const app = controller.apps.value[0]!
    controller.openRevokeAppDialog(app)

    expect(controller.revokeDialogTitle.value).toBe('user.authorizedApps.revokeAppDialogTitle')

    await controller.submitRevokeDialog()

    expect(mocks.revokeConsent).toHaveBeenCalledWith({ appID: 12 })
    expect(controller.revokeDialog.open).toBe(false)
    expect(controller.apps.value).toEqual([])
    expect(mocks.listConsentAuditEvents).toHaveBeenCalledTimes(1)
  })

  it('keeps the confirmation dialog open when revoke fails', async () => {
    mocks.revokeConsent.mockRejectedValueOnce(new Error('network failed'))
    mocks.getErrorMessage.mockReturnValueOnce('mapped revoke failure')

    const controller = useAuthorizedAppsController(t)
    await controller.loadConsents()

    controller.openRevokeAppDialog(controller.apps.value[0]!)
    await controller.submitRevokeDialog()

    expect(controller.revokeDialog.open).toBe(true)
    expect(controller.revokeDialog.error).toBe('mapped revoke failure')
    expect(controller.apps.value).toHaveLength(1)
    expect(mocks.toastError).toHaveBeenCalledWith('mapped revoke failure')
  })
})

function consentsResponse(apps: OpenPlatformUserAuthorizedApp[]) {
  return {
    data: {
      data: {
        apps,
      },
    },
  }
}

function auditResponse(list: OpenPlatformUserConsentAuditEvent[]) {
  return {
    data: {
      data: {
        list,
        total: list.length,
      },
    },
  }
}

function authorizedApp(scopes: OpenPlatformUserConsentScope[]): OpenPlatformUserAuthorizedApp {
  return {
    app: {
      id: 12,
      clientID: 'campus-planner',
      displayName: 'Campus Planner',
      description: 'Plan campus events and class reminders.',
      homepageURL: 'https://planner.example.com',
      privacyPolicyURL: 'https://planner.example.com/privacy',
      redirectURIs: ['https://planner.example.com/callback'],
      status: 'approved',
      createdAt: '2026-05-20T08:00:00Z',
      updatedAt: '2026-05-20T08:00:00Z',
    },
    scopes,
  }
}

function consentScope(
  scope: OpenPlatformUserConsentScope['scope'],
  displayName: string,
): OpenPlatformUserConsentScope {
  return {
    scope,
    displayName,
    sensitivity: scope === 'email.read' ? 'medium' : 'low',
    fields: scope === 'email.read' ? ['email'] : ['name', 'avatar'],
    grantedAt: '2026-05-21T09:00:00Z',
    lastUsedAt: '2026-05-22T09:00:00Z',
    grantSource: 'consent',
    reason: `${displayName} sync`,
  }
}

function auditEvent(
  eventType: string,
  scopes: OpenPlatformUserConsentAuditEvent['scopes'],
): OpenPlatformUserConsentAuditEvent {
  return {
    id: 100,
    appID: 12,
    appDisplayName: 'Campus Planner',
    clientID: 'campus-planner',
    eventType,
    scopes,
    endpoint: '/oidc/userinfo',
    result: 'granted',
    requestID: 'req-100',
    details: {},
    createdAt: '2026-05-23T09:00:00Z',
  }
}
