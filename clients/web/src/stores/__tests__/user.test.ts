import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockGetMyFavorites = vi.fn()
const mockGetFavoriteStatus = vi.fn()
const mockAddFavorite = vi.fn()
const mockRemoveFavorite = vi.fn()

vi.mock('@/api', () => ({
  api: {
    user: {
      getMyFavorites: mockGetMyFavorites,
      getFavoriteStatus: mockGetFavoriteStatus,
      addFavorite: mockAddFavorite,
      removeFavorite: mockRemoveFavorite,
    },
  },
}))

const { useUserStore } = await import('@/stores/user')

describe('useUserStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads favorites and marks listed courses as favorited', async () => {
    const favorite = {
      id: 101,
      name: '数据结构与算法',
      code: 'CS201',
      departmentID: 1,
      departmentName: '计算机科学与技术学院',
      reviewCount: 23,
      favoritedAt: '2026-03-25T10:00:00Z',
    }
    mockGetMyFavorites.mockResolvedValue({
      data: { data: { list: [favorite], total: 1 } },
    })

    const store = useUserStore()
    await store.fetchMyFavorites()

    expect(store.myFavorites).toEqual([favorite])
    expect(store.myFavoritesTotal).toBe(1)
    expect(store.myFavoritesLoading).toBe(false)
    expect(store.myFavoritesError).toBeNull()
    expect(store.favoriteStatus[101]).toBe(true)
  })

  it('fails closed when favorites response is missing page data', async () => {
    const favorite = {
      id: 101,
      name: '数据结构与算法',
      code: 'CS201',
      departmentID: 1,
      departmentName: '计算机科学与技术学院',
      reviewCount: 23,
      favoritedAt: '2026-03-25T10:00:00Z',
    }
    mockGetMyFavorites
      .mockResolvedValueOnce({
        data: { data: { list: [favorite], total: 1 } },
      })
      .mockResolvedValueOnce({
        data: { data: null },
      })

    const store = useUserStore()
    await store.fetchMyFavorites()

    await expect(store.fetchMyFavorites(2)).rejects.toThrow(
      'Invalid paginated response',
    )
    expect(store.myFavorites).toEqual([favorite])
    expect(store.myFavoritesTotal).toBe(1)
    expect(store.myFavoritesLoading).toBe(false)
    expect(store.myFavoritesError).toBe('Invalid paginated response')
  })

  it('keeps favorite status unknown when status response is malformed', async () => {
    mockGetFavoriteStatus.mockResolvedValue({ data: { data: null } })

    const store = useUserStore()
    await store.ensureFavoriteStatus(101)

    expect(store.favoriteStatus[101]).toBeUndefined()
  })
})
