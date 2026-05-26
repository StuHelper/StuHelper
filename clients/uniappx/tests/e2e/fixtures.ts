import type {
  ConsoleMessage,
  Page,
  Request,
  Response,
  Route,
} from '@playwright/test'

import { test as base, expect } from '@playwright/test'

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

function describeUnsuccessfulResponse(response: Response) {
  const request = response.request()
  return `${request.resourceType()} ${request.method()} ${response.url()} HTTP ${response.status()} ${
    response.statusText() || 'failed'
  }`
}

function isApiRequest(request: Request) {
  const resourceType = request.resourceType()
  if (resourceType !== 'fetch' && resourceType !== 'xhr') {
    return false
  }
  return new URL(request.url()).pathname.startsWith('/api/v1/')
}

function isExpectedApiErrorResponse(response: Response) {
  const request = response.request()
  const method = request.method().toUpperCase()
  const pathname = new URL(response.url()).pathname
  const status = response.status()

  if (method === 'GET' && pathname === '/api/v1/auth/me' && status === 401) {
    return true
  }
  if (method === 'POST' && pathname === '/api/v1/auth/refresh' && status === 401) {
    return true
  }
  return false
}

function isBrowserNetworkStatusConsoleError(text: string) {
  return /^Failed to load resource: the server responded with a status of [45]\d\d \([^)]+\)$/.test(
    text,
  )
}

function isExpectedAbortedTabBarImage(request: Request) {
  if (request.resourceType() !== 'image') return false
  if (request.failure()?.errorText !== 'net::ERR_ABORTED') return false
  return new URL(request.url()).pathname.startsWith('/static/tabbar/')
}

function isExpectedAbortedViteDevtoolsScript(request: Request) {
  if (request.resourceType() !== 'script') return false
  if (request.failure()?.errorText !== 'net::ERR_ABORTED') return false
  const pathname = new URL(request.url()).pathname
  return pathname.startsWith('/@fs/') && pathname.includes('/@vue/devtools-api/')
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
        !isBrowserNetworkStatusConsoleError(message.text())
      ) {
        consoleErrors.push(describeConsoleMessage(message))
      }
    })
    page.on('pageerror', (error) => {
      pageErrors.push(describePageError(error))
    })
    page.on('requestfailed', (request) => {
      if (
        isExpectedAbortedTabBarImage(request) ||
        isExpectedAbortedViteDevtoolsScript(request)
      ) {
        return
      }
      if (criticalResourceTypes.has(request.resourceType())) {
        failedRequests.push(describeFailedRequest(request))
      }
      if (isApiRequest(request)) {
        apiFailures.push(describeFailedRequest(request))
      }
    })
    page.on('response', (response) => {
      const request = response.request()
      if (
        criticalResourceTypes.has(request.resourceType()) &&
        response.status() >= 400
      ) {
        failedRequests.push(describeUnsuccessfulResponse(response))
      }
      if (isApiRequest(request) && response.status() >= 400) {
        if (isExpectedApiErrorResponse(response)) {
          return
        }
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
export type { Page, Route }
