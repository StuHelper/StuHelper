import { ref, computed, watch, type Ref } from 'vue'
import { pinyin } from 'pinyin-pro'

export interface PinyinSearchItem {
  id: number | string
  name: string
  _pinyin?: string
  _firstLetters?: string
  [key: string]: unknown
}

export interface UsePinyinSearchOptions<T extends PinyinSearchItem> {
  items: Ref<T[]>
  maxResults?: number
  sortBy?: (a: T, b: T) => number
}

interface SearchTokens {
  pinyin: string
  firstLetters: string
}

const SEARCH_TOKEN_CACHE_LIMIT = 500
const searchTokenCache = new Map<string, SearchTokens>()
const cjkPattern = /[\u3400-\u9fff]/

function isSubsequence(query: string, target: string): boolean {
  let queryIndex = 0
  for (const char of target) {
    if (char === query[queryIndex]) {
      queryIndex += 1
      if (queryIndex === query.length) return true
    }
  }
  return false
}

function getSearchTokens(name: string): SearchTokens {
  const cacheKey = name.trim().toLowerCase()
  const cached = searchTokenCache.get(cacheKey)
  if (cached) {
    searchTokenCache.delete(cacheKey)
    searchTokenCache.set(cacheKey, cached)
    return cached
  }

  const tokens: SearchTokens = {
    pinyin: pinyin(name, { toneType: 'none', type: 'array' }).join('').toLowerCase(),
    firstLetters: pinyin(name, { pattern: 'first', toneType: 'none', type: 'array' }).join('').toLowerCase(),
  }
  searchTokenCache.set(cacheKey, tokens)
  if (searchTokenCache.size > SEARCH_TOKEN_CACHE_LIMIT) {
    const oldestKey = searchTokenCache.keys().next().value
    if (oldestKey) {
      searchTokenCache.delete(oldestKey)
    }
  }
  return tokens
}

export function usePinyinSearch<T extends PinyinSearchItem>(options: UsePinyinSearchOptions<T>) {
  const { items, maxResults = 20 } = options

  const query = ref('')
  const selectedIndex = ref(0)

  const indexedItems = computed(() =>
    items.value.map((item) => {
      const tokens = getSearchTokens(item.name)
      return {
        ...item,
        _pinyin: tokens.pinyin,
        _firstLetters: tokens.firstLetters,
      }
    }),
  )

  const results = computed(() => {
    const q = query.value.toLowerCase().trim()
    if (!q) {
      const sorted = options.sortBy
        ? [...indexedItems.value].sort(options.sortBy)
        : indexedItems.value
      return sorted.slice(0, maxResults)
    }

    const filtered = indexedItems.value.filter((item) => {
      const normalizedName = item.name.toLowerCase()
      if (normalizedName.includes(q)) return true
      if (q.length >= 2 && cjkPattern.test(q) && isSubsequence(q, normalizedName)) return true
      if (item._pinyin?.includes(q)) return true
      if (item._firstLetters?.includes(q)) return true
      return false
    })

    const sorted = options.sortBy ? filtered.sort(options.sortBy) : filtered
    return sorted.slice(0, maxResults)
  })

  watch(results, () => {
    selectedIndex.value = 0
  })

  /** Handle keyboard navigation. Returns selected item on Enter, null otherwise. */
  function handleKeyDown(e: KeyboardEvent): T | null {
    if (!results.value.length) return null

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        selectedIndex.value = Math.min(selectedIndex.value + 1, results.value.length - 1)
        break
      case 'ArrowUp':
        e.preventDefault()
        selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
        break
      case 'Enter':
        e.preventDefault()
        return (results.value[selectedIndex.value] as T) ?? null
    }
    return null
  }

  function clear() {
    query.value = ''
    selectedIndex.value = 0
  }

  return {
    query,
    results,
    selectedIndex,
    handleKeyDown,
    clear,
  }
}
