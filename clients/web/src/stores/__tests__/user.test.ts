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

const favoriteCourse = {
  id: 101,
  name: '数据结构与算法',
  code: 'CS201',
  credits: 3,
  departmentID: 1,
  departmentName: '计算机科学与技术学院',
  reviewCount: 23,
  favoritedAt: '2026-03-25T10:00:00Z',
}

describe('useUserStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads favorites and marks listed courses as favorited', async () => {
    mockGetMyFavorites.mockResolvedValue({
      data: { data: { list: [favoriteCourse], total: 1 } },
    })

    const store = useUserStore()
    await store.fetchMyFavorites()

    expect(store.myFavorites).toEqual([favoriteCourse])
    expect(store.myFavoritesTotal).toBe(1)
    expect(store.myFavoritesLoading).toBe(false)
    expect(store.myFavoritesError).toBeNull()
    expect(store.favoriteStatus[101]).toBe(true)
  })

  it('preserves unknown department and credit metadata in favorites', async () => {
    const nullableFavorite = {
      ...favoriteCourse,
      departmentID: null,
      departmentName: undefined,
      code: undefined,
      credits: null,
    }
    mockGetMyFavorites.mockResolvedValue({
      data: { data: { list: [nullableFavorite], total: 1 } },
    })

    const store = useUserStore()
    await store.fetchMyFavorites()

    expect(store.myFavorites).toEqual([nullableFavorite])
    expect(store.myFavorites[0]?.departmentID).toBeNull()
    expect(store.myFavorites[0]?.credits).toBeNull()
  })

  it('fails closed when favorites response is missing page data', async () => {
    mockGetMyFavorites
      .mockResolvedValueOnce({
        data: { data: { list: [favoriteCourse], total: 1 } },
      })
      .mockResolvedValueOnce({
        data: { data: null },
      })

    const store = useUserStore()
    await store.fetchMyFavorites()

    await expect(store.fetchMyFavorites(2)).rejects.toThrow(
      'Invalid paginated response',
    )
    expect(store.myFavorites).toEqual([favoriteCourse])
    expect(store.myFavoritesTotal).toBe(1)
    expect(store.myFavoritesLoading).toBe(false)
    expect(store.myFavoritesError).toBe('Invalid paginated response')
  })

  it('fails closed when favorites response has malformed course fields', async () => {
    mockGetMyFavorites.mockResolvedValue({
      data: {
        data: {
          list: [
            {
              ...favoriteCourse,
              credits: '3',
            },
          ],
          total: 1,
        },
      },
    })

    const store = useUserStore()

    await expect(store.fetchMyFavorites()).rejects.toThrow(
      'Invalid favorite course response',
    )
    expect(store.myFavorites).toEqual([])
    expect(store.myFavoritesTotal).toBe(0)
    expect(store.myFavoritesLoading).toBe(false)
    expect(store.myFavoritesError).toBe('Invalid favorite course response')
    expect(store.favoriteStatus[101]).toBeUndefined()
  })

  it('keeps favorite status unknown when status response is malformed', async () => {
    mockGetFavoriteStatus.mockResolvedValue({ data: { data: null } })

    const store = useUserStore()
    await store.ensureFavoriteStatus(101)

    expect(store.favoriteStatus[101]).toBeUndefined()
  })
})
