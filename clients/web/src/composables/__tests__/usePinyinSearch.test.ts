import { describe, expect, it } from 'vitest'
import { effectScope, ref } from 'vue'

import { usePinyinSearch, type PinyinSearchItem } from '../usePinyinSearch'

interface CourseSearchItem extends PinyinSearchItem {
  id: number
  name: string
  reviewCount: number
}

function createSearch(items: CourseSearchItem[]) {
  const scope = effectScope()
  const state = scope.run(() =>
    usePinyinSearch<CourseSearchItem>({
      items: ref(items),
      maxResults: 10,
      sortBy: (a, b) => b.reviewCount - a.reviewCount,
    }),
  )
  if (!state) {
    throw new Error('failed to create pinyin search state')
  }
  return { scope, state }
}

describe('usePinyinSearch', () => {
  it('matches course names by pinyin initials', () => {
    const { scope, state } = createSearch([
      { id: 1, name: '高等数学A', reviewCount: 2 },
      { id: 2, name: '数据结构', reviewCount: 5 },
    ])

    state.query.value = 'gdsx'

    expect(state.results.value.map(item => item.name)).toEqual(['高等数学A'])
    scope.stop()
  })

  it('matches common Chinese abbreviated queries as subsequences', () => {
    const { scope, state } = createSearch([
      { id: 1, name: '高等数学A', reviewCount: 2 },
      { id: 2, name: '数据结构', reviewCount: 5 },
    ])

    state.query.value = '高数'

    expect(state.results.value.map(item => item.name)).toEqual(['高等数学A'])
    scope.stop()
  })
})
