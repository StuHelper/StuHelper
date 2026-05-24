import type { Page, Request, Response, Route } from '@playwright/test'

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

export const test = base.extend<{ page: Page }>({
  page: async ({ page }, use) => {
    const pageErrors: string[] = []
    const failedRequests: string[] = []

    page.on('pageerror', (error) => {
      pageErrors.push(describePageError(error))
    })
    page.on('requestfailed', (request) => {
      if (criticalResourceTypes.has(request.resourceType())) {
        failedRequests.push(describeFailedRequest(request))
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
    })

    await use(page)

    expect(pageErrors, 'unexpected browser page errors').toEqual([])
    expect(
      failedRequests,
      'critical browser resources should load with successful HTTP status',
    ).toEqual([])
  },
})

export { expect }
export type { Page, Route }
