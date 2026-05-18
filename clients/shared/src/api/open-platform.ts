import type { ApiClient } from './client'
import type { operations } from '../types/api.gen'

type ConsentPageResponse =
  operations['getOpenPlatformConsent']['responses'][200]['content']['application/json']['data']
type AuthorizeResponse =
  operations['openPlatformAuthorize']['responses'][200]['content']['application/json']['data']
type RedirectResponse =
  operations['acceptOpenPlatformConsent']['responses'][200]['content']['application/json']['data']

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
})

export type {
  AuthorizeResponse as OpenPlatformAuthorizeResponse,
  ConsentPageResponse as OpenPlatformConsentPageResponse,
  RedirectResponse as OpenPlatformRedirectResponse,
}
