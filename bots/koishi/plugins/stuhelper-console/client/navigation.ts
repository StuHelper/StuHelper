import {
  DEFAULT_CONSOLE_SECTION,
  isConsoleSectionId,
  type ConsoleSectionId,
} from './sections'

export interface ConsoleSearchState {
  section: ConsoleSectionId
  queue: string | null
  id: string
  source: 'dashboard' | 'direct' | 'nav'
}

const DEFAULT_SOURCE: ConsoleSearchState['source'] = 'direct'

export function parseConsoleSearch(params: URLSearchParams): ConsoleSearchState {
  const rawSection = params.get('section') ?? ''
  const source = params.get('source')
  return {
    section: isConsoleSectionId(rawSection)
      ? rawSection
      : DEFAULT_CONSOLE_SECTION,
    queue: params.get('queue'),
    id: params.get('id') ?? '',
    source:
      source === 'dashboard' || source === 'nav' || source === 'direct'
        ? source
        : DEFAULT_SOURCE,
  }
}

export function createConsoleSearch(state: ConsoleSearchState) {
  const params = new URLSearchParams()
  params.set('section', state.section)
  if (state.queue) params.set('queue', state.queue)
  if (state.id) params.set('id', state.id)
  params.set('source', state.source)
  return params
}
