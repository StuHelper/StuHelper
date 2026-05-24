import type { ApiClient } from './client'
import type { operations } from '../types/api.gen'

type ConsentPageResponse =
  operations['getOpenPlatformConsent']['responses'][200]['content']['application/json']['data']
type ProfileCompletionPageResponse =
  operations['getOpenPlatformProfileCompletion']['responses'][200]['content']['application/json']['data']
type AuthorizeResponse =
  operations['openPlatformAuthorize']['responses'][200]['content']['application/json']['data']
type RedirectResponse =
  operations['acceptOpenPlatformConsent']['responses'][200]['content']['application/json']['data']
type AppListResponse =
  operations['listOpenPlatformApps']['responses'][200]['content']['application/json']['data']
type AppListParams =
  NonNullable<operations['listOpenPlatformApps']['parameters']['query']>
type DeveloperAppAuditEventsResponse =
  operations['listOpenPlatformAppAuditEvents']['responses'][200]['content']['application/json']['data']
type DeveloperAppAuditEventsParams =
  NonNullable<operations['listOpenPlatformAppAuditEvents']['parameters']['query']>
type RegisterAppRequest =
  operations['registerOpenPlatformApp']['requestBody']['content']['application/json']
type UpdateAppProfileRequest =
  operations['updateOpenPlatformAppProfile']['requestBody']['content']['application/json']
type UpdateAppProfileResponse =
  operations['updateOpenPlatformAppProfile']['responses'][200]['content']['application/json']['data']
type ScopeChangeRequest =
  operations['requestOpenPlatformAppScopes']['requestBody']['content']['application/json']
type ScopeChangeResponse =
  operations['requestOpenPlatformAppScopes']['responses'][201]['content']['application/json']['data']
type ScopeWithdrawalResponse =
  operations['withdrawOpenPlatformScopeRequest']['responses'][200]['content']['application/json']['data']
type RedirectURIChangeRequest =
  operations['requestOpenPlatformRedirectURIs']['requestBody']['content']['application/json']
type RedirectURIRequestResponse =
  operations['requestOpenPlatformRedirectURIs']['responses'][201]['content']['application/json']['data']
type RedirectURIWithdrawalResponse =
  operations['withdrawOpenPlatformRedirectURIRequest']['responses'][200]['content']['application/json']['data']
type RotateAppSecretRequest =
  NonNullable<operations['rotateOpenPlatformAppSecret']['requestBody']>['content']['application/json']
type RotatedSecretResponse =
  operations['rotateOpenPlatformAppSecret']['responses'][200]['content']['application/json']['data']
type WithdrawalRequest =
  operations['withdrawOpenPlatformApp']['requestBody']['content']['application/json']
type AppLifecycleResponse =
  operations['withdrawOpenPlatformApp']['responses'][200]['content']['application/json']['data']
type UserConsentsResponse =
  operations['listOpenPlatformConsents']['responses'][200]['content']['application/json']['data']
type UserConsentAuditEventsResponse =
  operations['listOpenPlatformConsentAuditEvents']['responses'][200]['content']['application/json']['data']
type UserConsentAuditEventsParams =
  NonNullable<operations['listOpenPlatformConsentAuditEvents']['parameters']['query']>
type ResourceAccessCheckRequest =
  operations['checkOpenPlatformResourceAccess']['requestBody']['content']['application/json']
type ResourceAccessDecisionResponse =
  operations['checkOpenPlatformResourceAccess']['responses'][200]['content']['application/json']['data']
type AppWithScopes = AppListResponse['list'][number]
type AppRedirectURIRequest = AppWithScopes['redirectURIRequests'][number]
type AppScopeRequest = AppWithScopes['scopes'][number]
type DeveloperAppAuditEvent = DeveloperAppAuditEventsResponse['list'][number]
type UserAuthorizedApp = UserConsentsResponse['apps'][number]
type UserConsentAuditEvent = UserConsentAuditEventsResponse['list'][number]
type UserConsentScope = UserAuthorizedApp['scopes'][number]

export const createOpenPlatformApi = (client: ApiClient) => ({
  authorize: (input: {
    clientID: string
    redirectURI: string
    scope: string
    state?: string
  }) => {
    const query: NonNullable<operations['openPlatformAuthorize']['parameters']['query']> = {
      client_id: input.clientID,
      redirect_uri: input.redirectURI,
      scope: input.scope,
    }
    if (input.state) query.state = input.state
    return client.GET('/api/v1/open-platform/authorize', {
      params: { query },
    })
  },

  getConsent: (token: string) =>
    client.GET('/api/v1/open-platform/consent', {
      params: { query: { token } },
    }),

  acceptConsent: (token: string) =>
    client.POST('/api/v1/open-platform/consent/accept', {
      body: { token },
    }),

  denyConsent: (token: string) =>
    client.POST('/api/v1/open-platform/consent/deny', {
      body: { token },
    }),

  getProfileCompletion: (token: string) =>
    client.GET('/api/v1/open-platform/profile-completion', {
      params: { query: { token } },
    }),

  continueProfileCompletion: (token: string) =>
    client.POST('/api/v1/open-platform/profile-completion/continue', {
      body: { token },
    }),

  listApps: (params?: AppListParams) =>
    client.GET('/api/v1/open-platform/apps', {
      params: { query: params },
    }),

  listAppAuditEvents: (input: {
    appID: number
    params?: DeveloperAppAuditEventsParams
  }) =>
    client.GET('/api/v1/open-platform/apps/{appID}/audit-events', {
      params: {
        path: { appID: input.appID },
        query: input.params,
      },
    }),

  registerApp: (data: RegisterAppRequest) =>
    client.POST('/api/v1/open-platform/apps', {
      body: data,
    }),

  updateAppProfile: (input: {
    appID: number
    profile: UpdateAppProfileRequest
  }) =>
    client.PATCH('/api/v1/open-platform/apps/{appID}', {
      body: input.profile,
      params: { path: { appID: input.appID } },
    }),

  requestScopeChange: (input: {
    appID: number
    scopes: ScopeChangeRequest['scopes']
  }) =>
    client.POST('/api/v1/open-platform/apps/{appID}/scopes', {
      body: { scopes: input.scopes },
      params: { path: { appID: input.appID } },
    }),

  withdrawScopeRequest: (input: {
    appID: number
    scope: AppScopeRequest['scope']
    reason: WithdrawalRequest['reason']
  }) =>
    client.POST('/api/v1/open-platform/apps/{appID}/scopes/{scope}/withdraw', {
      body: { reason: input.reason },
      params: { path: { appID: input.appID, scope: input.scope } },
    }),

  requestRedirectURIChange: (input: {
    appID: number
    redirectURIs: RedirectURIChangeRequest['redirectURIs']
    reason: RedirectURIChangeRequest['reason']
  }) =>
    client.POST('/api/v1/open-platform/apps/{appID}/redirect-uris', {
      body: {
        redirectURIs: input.redirectURIs,
        reason: input.reason,
      },
      params: { path: { appID: input.appID } },
    }),

  withdrawRedirectURIRequest: (input: {
    appID: number
    requestID: number
    reason: WithdrawalRequest['reason']
  }) =>
    client.POST('/api/v1/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/withdraw', {
      body: { reason: input.reason },
      params: { path: { appID: input.appID, requestID: input.requestID } },
    }),

  withdrawApp: (input: {
    appID: number
    reason: WithdrawalRequest['reason']
  }) =>
    client.POST('/api/v1/open-platform/apps/{appID}/withdraw', {
      body: { reason: input.reason },
      params: { path: { appID: input.appID } },
    }),

  rotateAppSecret: (input: {
    appID: number
    reason?: string
  }) => {
    const body: RotateAppSecretRequest | undefined = input.reason
      ? { reason: input.reason }
      : undefined
    return client.POST('/api/v1/open-platform/apps/{appID}/secret/rotate', {
      body,
      params: { path: { appID: input.appID } },
    })
  },

  listConsents: () =>
    client.GET('/api/v1/open-platform/consents'),

  listConsentAuditEvents: (params?: UserConsentAuditEventsParams) =>
    client.GET('/api/v1/open-platform/consents/audit-events', {
      params: { query: params },
    }),

  revokeConsent: (input: {
    appID: number
    scopes?: UserConsentScope['scope'][]
  }) => {
    const query: NonNullable<operations['revokeOpenPlatformConsent']['parameters']['query']> = {}
    if (input.scopes?.length) query.scope = input.scopes
    return client.DELETE('/api/v1/open-platform/consents/{appID}', {
      params: {
        path: { appID: input.appID },
        query,
      },
    })
  },

  checkResourceAccess: (data: ResourceAccessCheckRequest, options?: { accessToken?: string }) => {
    const request = {
      body: data,
    }
    if (options?.accessToken) {
      return client.POST('/api/v1/open-platform/resources/access/check', {
        ...request,
        headers: { Authorization: `Bearer ${options.accessToken}` },
      })
    }
    return client.POST('/api/v1/open-platform/resources/access/check', request)
  },
})

export type {
  AppListParams as OpenPlatformAppListParams,
  AppListResponse as OpenPlatformAppListResponse,
  AppLifecycleResponse as OpenPlatformAppLifecycleResponse,
  AppRedirectURIRequest as OpenPlatformAppRedirectURIRequest,
  AppScopeRequest as OpenPlatformAppScopeRequest,
  AppWithScopes as OpenPlatformAppWithScopes,
  AuthorizeResponse as OpenPlatformAuthorizeResponse,
  ConsentPageResponse as OpenPlatformConsentPageResponse,
  DeveloperAppAuditEvent as OpenPlatformDeveloperAppAuditEvent,
  DeveloperAppAuditEventsParams as OpenPlatformDeveloperAppAuditEventsParams,
  DeveloperAppAuditEventsResponse as OpenPlatformDeveloperAppAuditEventsResponse,
  ProfileCompletionPageResponse as OpenPlatformProfileCompletionPageResponse,
  ResourceAccessCheckRequest as OpenPlatformResourceAccessCheckRequest,
  ResourceAccessDecisionResponse as OpenPlatformResourceAccessDecisionResponse,
  RedirectURIChangeRequest as OpenPlatformRedirectURIChangeRequest,
  RedirectURIRequestResponse as OpenPlatformRedirectURIRequestResponse,
  RedirectURIWithdrawalResponse as OpenPlatformRedirectURIWithdrawalResponse,
  RegisterAppRequest as OpenPlatformRegisterAppRequest,
  RedirectResponse as OpenPlatformRedirectResponse,
  RotatedSecretResponse as OpenPlatformRotatedSecretResponse,
  ScopeChangeRequest as OpenPlatformScopeChangeRequest,
  ScopeChangeResponse as OpenPlatformScopeChangeResponse,
  ScopeWithdrawalResponse as OpenPlatformScopeWithdrawalResponse,
  UpdateAppProfileRequest as OpenPlatformUpdateAppProfileRequest,
  UpdateAppProfileResponse as OpenPlatformUpdateAppProfileResponse,
  UserConsentAuditEvent as OpenPlatformUserConsentAuditEvent,
  UserConsentAuditEventsParams as OpenPlatformUserConsentAuditEventsParams,
  UserConsentAuditEventsResponse as OpenPlatformUserConsentAuditEventsResponse,
  UserAuthorizedApp as OpenPlatformUserAuthorizedApp,
  UserConsentScope as OpenPlatformUserConsentScope,
  UserConsentsResponse as OpenPlatformUserConsentsResponse,
}
