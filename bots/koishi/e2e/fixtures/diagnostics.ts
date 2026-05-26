import { expect, type ConsoleMessage, type Page, type Request, type Response } from '@playwright/test'

interface ConsoleIssue {
  readonly type: 'error' | 'warning'
  readonly text: string
  readonly url: string
}

interface ResourceIssue {
  readonly method: string
  readonly resourceType: string
  readonly url: string
  readonly status?: number
  readonly statusText?: string
  readonly failure?: string
}

export interface Tracker extends AsyncDisposable {
  readonly issues: readonly ConsoleIssue[]
  readonly errors: readonly Error[]
  readonly resourceIssues: readonly ResourceIssue[]
  assertClean(): void
}

/**
 * Koishi Console 的 activity 路由由各插件异步 addRoute 注册。直连动态页面时，
 * Vue Router 会在 entry 注册完成前打印一次 "No match found"。这些路径随后由
 * RouterService 的 loader warm-up 正常解析；其他路径的同类 warning 仍然失败。
 */
const CONSOLE_ALLOWLIST: readonly RegExp[] = [
  /^\[Vue Router warn\]: No match found for location with path "\/login"$/,
  /^\[Vue Router warn\]: No match found for location with path "\/stuhelper(\?[^"]*)?"$/,
]

const CRITICAL_RESOURCE_TYPES = new Set(['document', 'font', 'image', 'script', 'stylesheet'])

export function createTracker(page: Page): Tracker {
  const issues: ConsoleIssue[] = []
  const errors: Error[] = []
  const resourceIssues: ResourceIssue[] = []

  const onConsole = (message: ConsoleMessage) => {
    const type = message.type()
    if (type !== 'error' && type !== 'warning') return
    const text = message.text()
    if (CONSOLE_ALLOWLIST.some((pattern) => pattern.test(text))) return
    issues.push({ type, text, url: message.location().url })
  }
  const onPageError = (error: Error) => {
    errors.push(error)
  }
  const onRequestFailed = (request: Request) => {
    if (!CRITICAL_RESOURCE_TYPES.has(request.resourceType())) return
    resourceIssues.push({
      method: request.method(),
      resourceType: request.resourceType(),
      url: request.url(),
      failure: request.failure()?.errorText ?? 'failed',
    })
  }
  const onResponse = (response: Response) => {
    const request = response.request()
    if (!CRITICAL_RESOURCE_TYPES.has(request.resourceType()) || response.status() < 400) {
      return
    }
    resourceIssues.push({
      method: request.method(),
      resourceType: request.resourceType(),
      url: response.url(),
      status: response.status(),
      statusText: response.statusText(),
    })
  }

  page.on('console', onConsole)
  page.on('pageerror', onPageError)
  page.on('requestfailed', onRequestFailed)
  page.on('response', onResponse)

  return {
    issues,
    errors,
    resourceIssues,
    assertClean() {
      expect(
        errors,
        `unexpected pageerror:\n${errors.map((error) => `  ${error.message}`).join('\n')}`,
      ).toHaveLength(0)
      expect(
        issues,
        `unexpected console output:\n${issues
          .map((issue) => `  [${issue.type}] ${issue.text} (${issue.url})`)
          .join('\n')}`,
      ).toHaveLength(0)
      expect(
        resourceIssues,
        `unexpected critical resource failures:\n${resourceIssues
          .map((issue) => {
            const status = issue.status ? ` HTTP ${issue.status} ${issue.statusText ?? ''}` : ''
            const failure = issue.failure ? ` ${issue.failure}` : ''
            return `  ${issue.resourceType} ${issue.method} ${issue.url}${status}${failure}`
          })
          .join('\n')}`,
      ).toHaveLength(0)
    },
    async [Symbol.asyncDispose]() {
      page.off('console', onConsole)
      page.off('pageerror', onPageError)
      page.off('requestfailed', onRequestFailed)
      page.off('response', onResponse)
    },
  }
}
