import { describe, expect, it, vi } from 'vitest'
import { loadCourseCatalog } from '../courseCatalogLoader'

describe('loadCourseCatalog', () => {
  it('loads all courses even when backend clamps requested page size', async () => {
    const fetchPage = vi.fn(async (page: number, pageSize: number) => {
      expect(pageSize).toBe(200)

      if (page === 1) {
        return {
          list: Array.from({ length: 100 }, (_, index) => ({ id: index + 1 })),
          total: 150
        }
      }

      if (page === 2) {
        return {
          list: Array.from({ length: 50 }, (_, index) => ({ id: index + 101 })),
          total: 150
        }
      }

      return {
        list: [],
        total: 150
      }
    })

    const list = await loadCourseCatalog(fetchPage, 200)

    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(fetchPage).toHaveBeenNthCalledWith(1, 1, 200)
    expect(fetchPage).toHaveBeenNthCalledWith(2, 2, 200)
    expect(list).toHaveLength(150)
    expect(list[0]).toEqual({ id: 1 })
    expect(list[149]).toEqual({ id: 150 })
  })

  it('stops after first page when all courses are already returned', async () => {
    const fetchPage = vi.fn(async () => ({
      list: Array.from({ length: 80 }, (_, index) => ({ id: index + 1 })),
      total: 80
    }))

    const list = await loadCourseCatalog(fetchPage, 200)

    expect(fetchPage).toHaveBeenCalledTimes(1)
    expect(list).toHaveLength(80)
  })
})
