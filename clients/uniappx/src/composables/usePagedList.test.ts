import { describe, expect, it, vi } from 'vitest'

import { usePagedList, type PageResult } from './usePagedList'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

describe('usePagedList', () => {
  it('commits the next page only after a successful response', async () => {
    const onError = vi.fn()
    const fetchPage = vi.fn(async (page: number): Promise<PageResult<number>> => {
      if (page === 2 && fetchPage.mock.calls.length === 2) {
        throw new Error('temporary failure')
      }
      return page === 1
        ? { list: [1, 2], total: 3 }
        : { list: [3], total: 3 }
    })
    const pager = usePagedList({ fetchPage, onError, pageSize: 2 })

    expect(await pager.refresh()).toBe(true)
    expect(pager.page.value).toBe(1)

    expect(await pager.loadMore()).toBe(false)
    expect(pager.page.value).toBe(1)
    expect(pager.items.value).toEqual([1, 2])
    expect(onError).toHaveBeenCalledWith(expect.any(Error), 'more')

    expect(await pager.loadMore()).toBe(true)
    expect(pager.page.value).toBe(2)
    expect(pager.items.value).toEqual([1, 2, 3])
    expect(fetchPage.mock.calls.map(([page]) => page)).toEqual([1, 2, 2])
  })

  it('uses total instead of page fullness to decide whether more data exists', async () => {
    const pager = usePagedList({
      fetchPage: async () => ({ list: [1, 2], total: 2 }),
      onError: vi.fn(),
      pageSize: 2,
    })

    await pager.refresh()

    expect(pager.hasMore.value).toBe(false)
    expect(await pager.loadMore()).toBe(false)
  })

  it('ignores a slower stale refresh response', async () => {
    const first = deferred<PageResult<string>>()
    const second = deferred<PageResult<string>>()
    const fetchPage = vi.fn()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise)
    const pager = usePagedList<string>({
      fetchPage,
      onError: vi.fn(),
      pageSize: 20,
    })

    const firstRefresh = pager.refresh()
    const secondRefresh = pager.refresh()
    second.resolve({ list: ['current'], total: 1 })
    await secondRefresh
    first.resolve({ list: ['stale'], total: 1 })
    await firstRefresh

    expect(pager.items.value).toEqual(['current'])
    expect(pager.loading.value).toBe(false)
  })
})
