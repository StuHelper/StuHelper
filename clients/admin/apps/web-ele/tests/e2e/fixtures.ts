import type { Page, Request, Route } from '@playwright/test';

import { test as base, expect } from '@playwright/test';

const criticalResourceTypes = new Set([
  'document',
  'font',
  'image',
  'script',
  'stylesheet',
]);

function describePageError(error: Error) {
  return error.stack || `${error.name}: ${error.message}`;
}

function describeFailedRequest(request: Request) {
  const failure = request.failure();
  return `${request.resourceType()} ${request.method()} ${request.url()} ${
    failure?.errorText ?? 'failed'
  }`;
}

export const test = base.extend<{ page: Page }>({
  page: async ({ page }, use) => {
    const pageErrors: string[] = [];
    const failedRequests: string[] = [];

    page.on('pageerror', (error) => {
      pageErrors.push(describePageError(error));
    });
    page.on('requestfailed', (request) => {
      if (criticalResourceTypes.has(request.resourceType())) {
        failedRequests.push(describeFailedRequest(request));
      }
    });

    await use(page);

    expect(pageErrors, 'unexpected browser page errors').toEqual([]);
    expect(failedRequests, 'critical browser resources should load').toEqual(
      [],
    );
  },
});

export { expect };
export type { Page, Route };
