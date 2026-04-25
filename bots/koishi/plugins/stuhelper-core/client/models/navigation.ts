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
    view: isConsoleViewId(rawView) ? rawView : DEFAULT_CONSOLE_VIEW,
    workspace: readNullableQuery(params, 'workspace'),
    guildId: readNullableQuery(params, 'guildId'),
    memberId: readNullableQuery(params, 'memberId'),
    itemId: readNullableQuery(params, 'itemId'),
    tab: readNullableQuery(params, 'tab'),
    keyword: params.get('keyword') ?? '',
  }
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

export function mergeConsoleQuery(currentUrl: URL, state: ConsoleNavigationState) {
  const nextUrl = new URL(currentUrl.href)
  const merged = new URLSearchParams()

  currentUrl.searchParams.forEach((value, key) => {
    if (QUERY_KEYS.includes(key as (typeof QUERY_KEYS)[number])) {
      return
    }
    merged.append(key, value)
  })

  createConsoleQuery(state).forEach((value, key) => {
    merged.append(key, value)
  })

  nextUrl.search = merged.toString()
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
