import {
  DEFAULT_CONSOLE_VIEW,
  isConsoleViewId,
  type ConsoleViewId,
} from './views'

export interface ConsoleNavigationState {
  view: ConsoleViewId
  workspace: string | null
  guildId: string | null
  memberId: string | null
  itemId: string | null
  tab: string | null
  keyword: string
}

const QUERY_KEYS = [
  'view',
  'workspace',
  'guildId',
  'memberId',
  'itemId',
  'tab',
  'keyword',
] as const

export function parseConsoleQuery(params: URLSearchParams): ConsoleNavigationState {
  const rawView = params.get('view')
  return {
    view: parseConsoleView(rawView),
    workspace: readNullableQuery(params, 'workspace'),
    guildId: readNullableQuery(params, 'guildId'),
    memberId: readNullableQuery(params, 'memberId'),
    itemId: readNullableQuery(params, 'itemId'),
    tab: readNullableQuery(params, 'tab'),
    keyword: params.get('keyword') ?? '',
  }
}

export function parseConsoleHash(hash: string): ConsoleNavigationState | null {
  const fragment = hash.startsWith('#') ? hash.slice(1) : hash
  if (!fragment) return null
  if (fragment.startsWith('?') || fragment.startsWith('view=')) {
    return parseConsoleQuery(new URLSearchParams(fragment.replace(/^\?/, '')))
  }

  const separatorIndex = fragment.indexOf('?')
  const rawView = separatorIndex >= 0 ? fragment.slice(0, separatorIndex) : fragment
  const view = rawView.replace(/^\/+/, '')
  if (!isConsoleViewId(view)) {
    throw new Error(`unknown console view: ${view}`)
  }

  const params = new URLSearchParams(separatorIndex >= 0 ? fragment.slice(separatorIndex + 1) : '')
  params.set('view', view)
  return parseConsoleQuery(params)
}

export function parseConsoleLocation(url: URL): ConsoleNavigationState {
  return parseConsoleHash(url.hash) ?? parseConsoleQuery(url.searchParams)
}

export function createConsoleQuery(state: ConsoleNavigationState) {
  const params = new URLSearchParams()
  params.set('view', state.view)
  appendNullableQuery(params, 'workspace', state.workspace)
  appendNullableQuery(params, 'guildId', state.guildId)
  appendNullableQuery(params, 'memberId', state.memberId)
  appendNullableQuery(params, 'itemId', state.itemId)
  appendNullableQuery(params, 'tab', state.tab)
  if (state.keyword) {
    params.set('keyword', state.keyword)
  }
  return params
}

export function createConsoleHash(state: ConsoleNavigationState) {
  const params = new URLSearchParams()
  appendNullableQuery(params, 'workspace', state.workspace)
  appendNullableQuery(params, 'guildId', state.guildId)
  appendNullableQuery(params, 'memberId', state.memberId)
  appendNullableQuery(params, 'itemId', state.itemId)
  appendNullableQuery(params, 'tab', state.tab)
  if (state.keyword) {
    params.set('keyword', state.keyword)
  }

  const query = params.toString()
  return query ? `#${state.view}?${query}` : `#${state.view}`
}

export function mergeConsoleLocation(currentUrl: URL, state: ConsoleNavigationState) {
  const nextUrl = new URL(currentUrl.href)
  const merged = new URLSearchParams()

  currentUrl.searchParams.forEach((value, key) => {
    if (QUERY_KEYS.includes(key as (typeof QUERY_KEYS)[number])) {
      return
    }
    merged.append(key, value)
  })

  nextUrl.search = merged.toString()
  nextUrl.hash = createConsoleHash(state)
  return nextUrl
}

function appendNullableQuery(
  params: URLSearchParams,
  key: Exclude<keyof ConsoleNavigationState, 'view' | 'keyword'>,
  value: string | null,
) {
  if (value) {
    params.set(key, value)
  }
}

function readNullableQuery(
  params: URLSearchParams,
  key: Exclude<keyof ConsoleNavigationState, 'view' | 'keyword'>,
) {
  const value = params.get(key)
  return value ? value : null
}

function parseConsoleView(value: string | null) {
  if (!value) {
    return DEFAULT_CONSOLE_VIEW
  }
  if (!isConsoleViewId(value)) {
    throw new Error(`unknown console view: ${value}`)
  }
  return value
}
