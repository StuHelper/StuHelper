import type {
  ConsoleMessage,
  Page,
  Request,
  Response,
  Route,
} from '@playwright/test'

import { test as base, expect } from '@playwright/test'

const expectedCriticalResourceFailures = new WeakMap<Page, RegExp[]>()
const expectedConsoleErrors = new WeakMap<Page, RegExp[]>()

export function allowExpectedCriticalResourceFailure(
  page: Page,
  urlPattern: RegExp,
) {
  const patterns = expectedCriticalResourceFailures.get(page) ?? []
  patterns.push(urlPattern)
  expectedCriticalResourceFailures.set(page, patterns)
}

export function allowExpectedConsoleError(page: Page, textPattern: RegExp) {
  const patterns = expectedConsoleErrors.get(page) ?? []
  patterns.push(textPattern)
  expectedConsoleErrors.set(page, patterns)
}

const criticalResourceTypes = new Set([
  'document',
  'font',
  'image',
  'script',
  'stylesheet',
])

function describePageError(error: Error) {
  return error.stack || `${error.name}: ${error.message}`
}

function describeConsoleMessage(message: ConsoleMessage) {
  const location = message.location()
  const locationText =
    location.url && location.lineNumber > 0
      ? ` (${location.url}:${location.lineNumber}:${location.columnNumber})`
      : ''
  return `${message.text()}${locationText}`
}

function describeFailedRequest(request: Request) {
  const failure = request.failure()
  return `${request.resourceType()} ${request.method()} ${request.url()} ${
    failure?.errorText ?? 'failed'
  }`
}

function isExpectedCriticalResourceFailure(page: Page, url: string) {
  return (
    expectedCriticalResourceFailures
      .get(page)
      ?.some((pattern) => pattern.test(url)) ?? false
  )
}

function isExpectedConsoleError(page: Page, text: string) {
  return (
    expectedConsoleErrors
      .get(page)
      ?.some((pattern) => pattern.test(text)) ?? false
  )
}

function describeUnsuccessfulResponse(response: Response) {
  const request = response.request()
  return `${request.resourceType()} ${request.method()} ${response.url()} HTTP ${response.status()} ${
    response.statusText() || 'failed'
  }`
}

function isApiRequest(request: Request) {
  const resourceType = request.resourceType()
  if (
    resourceType !== 'fetch' &&
    resourceType !== 'xhr' &&
    resourceType !== 'eventsource'
  ) {
    return false
  }
  return new URL(request.url()).pathname.startsWith('/api/v1/')
}

function isExpectedApiRequestFailure(request: Request) {
  return request.failure()?.errorText === 'net::ERR_ABORTED'
}

async function mockNotificationStream(page: Page) {
  await page.route(
    '**/api/v1/course/review/user/notifications/stream',
    (route) =>
      route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        headers: { 'Cache-Control': 'no-cache' },
        body: 'event: unread_count\ndata: {"count":0}\n\n',
      }),
  )
}

function isBrowserNetworkStatusConsoleError(text: string) {
  return /^Failed to load resource: the server responded with a status of [45]\d\d \([^)]+\)$/.test(
    text,
  )
}

function isExpectedApiErrorResponse(response: Response) {
  const request = response.request()
  const method = request.method().toUpperCase()
  const pathname = new URL(response.url()).pathname
  const status = response.status()

  if (method === 'GET' && pathname === '/api/v1/auth/me' && status === 401) {
    return true
  }
  if (
    method === 'GET' &&
    /^\/api\/v1\/admission\/sessions\/[^/]+$/.test(pathname) &&
    (status === 400 || status === 404 || status === 409 || status === 410)
  ) {
    return true
  }
  if (
    method === 'GET' &&
    /^\/api\/v1\/course\/review\/courses\/[^/]+\/favorites$/.test(pathname) &&
    status === 401
  ) {
    return true
  }
  if (
    method === 'GET' &&
    pathname === '/api/v1/course/review/drafts' &&
    status === 404
  ) {
    return true
  }

  return false
}

export const test = base.extend<{ page: Page }>({
  page: async ({ page }, use) => {
    const pageErrors: string[] = []
    const consoleErrors: string[] = []
    const failedRequests: string[] = []
    const apiFailures: string[] = []

    page.on('console', (message) => {
      if (
        message.type() === 'error' &&
        !isBrowserNetworkStatusConsoleError(message.text()) &&
        !isExpectedConsoleError(page, message.text())
      ) {
        consoleErrors.push(describeConsoleMessage(message))
      }
    })
    page.on('pageerror', (error) => {
      pageErrors.push(describePageError(error))
    })
    page.on('requestfailed', (request) => {
      if (
        criticalResourceTypes.has(request.resourceType()) &&
        !isExpectedCriticalResourceFailure(page, request.url())
      ) {
        failedRequests.push(describeFailedRequest(request))
      }
      if (isApiRequest(request)) {
        if (!isExpectedApiRequestFailure(request)) {
          apiFailures.push(describeFailedRequest(request))
        }
      }
    })
    page.on('response', (response) => {
      const request = response.request()
      if (
        criticalResourceTypes.has(request.resourceType()) &&
        response.status() >= 400 &&
        !isExpectedCriticalResourceFailure(page, response.url())
      ) {
        failedRequests.push(describeUnsuccessfulResponse(response))
      }
      if (
        isApiRequest(request) &&
        response.status() >= 400 &&
        !isExpectedApiErrorResponse(response)
      ) {
        apiFailures.push(describeUnsuccessfulResponse(response))
      }
    })

    await use(page)

    expect(pageErrors, 'unexpected browser page errors').toEqual([])
    expect(consoleErrors, 'unexpected browser console errors').toEqual([])
    expect(
      failedRequests,
      'critical browser resources should load with successful HTTP status',
    ).toEqual([])
    expect(
      apiFailures,
      'API requests should not fail with unexpected network errors or HTTP 4xx/5xx',
    ).toEqual([])
  },
})

export { expect }
export { mockNotificationStream }
export type { Page, Route }

interface CurrentAccountProjectionOptions {
  capabilities?: string[]
  displayName?: string
  phoneBound?: boolean
  studentVerified?: boolean
}

/**
 * Installs the three independent account projections consumed by AppShell.
 * Student eligibility, phone verification and QQ binding deliberately remain
 * separate responses so E2E tests cannot revive the deleted profile fallback.
 */
export async function mockCurrentAccountProjections(
  page: Page,
  options: CurrentAccountProjectionOptions = {},
) {
  const studentVerified = options.studentVerified ?? true
  const phoneBound = options.phoneBound ?? false
  await page.route('**/api/v1/user/me', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: {
          displayName: options.displayName ?? 'Test User',
          phone: phoneBound ? '138****8000' : null,
          studentVerificationStatus: studentVerified ? 'approved' : 'none',
          phoneBound,
          capabilities: options.capabilities ?? [],
        },
      }),
    }),
  )
  await page.route('**/api/v1/account/phone', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        success: true,
        data: phoneBound
          ? {
              state: 'verified',
              maskedPhone: '138****8000',
              method: 'sms_possession',
              verifiedAt: '2026-08-05T08:00:00Z',
              expiresAt: null,
              publishingRequirementSatisfied: true,
              revision: 1,
            }
          : {
              state: 'unbound',
              maskedPhone: null,
              method: null,
              verifiedAt: null,
              expiresAt: null,
              publishingRequirementSatisfied: false,
              revision: 1,
            },
      }),
    }),
  )
  await page.route('**/api/v1/user/qq-binding', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ success: true, data: null }),
    }),
  )
}
