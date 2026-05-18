import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { SaveDraftParams } from '@stuhelper/shared/draft'

const mockSaveDraft = vi.fn()
const mockGetDraft = vi.fn()
const mockDeleteDraft = vi.fn()

vi.mock('@/api', () => ({
  api: {
    draft: {
      saveDraft: mockSaveDraft,
      getDraft: mockGetDraft,
      deleteDraft: mockDeleteDraft,
    },
  },
}))

vi.mock('@stuhelper/shared/course', () => ({
  isValidRating: (v: unknown) => typeof v === 'number' && v >= 1 && v <= 5,
}))

const { useDraftStore } = await import('@/stores/draft')

describe('useDraftStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('saveDraft', () => {
    it('saves a user-level draft without requiring a course', async () => {
      const draftData = {
        id: 'draft-1',
        title: 'Test Draft',
        content: 'Some content',
        updatedAt: '2026-04-05T00:00:00Z',
      }
      mockSaveDraft.mockResolvedValue({ data: { data: draftData } })

      const store = useDraftStore()
      const result = await store.saveDraft({ title: 'Test Draft', content: 'Some content' })

      expect(mockSaveDraft).toHaveBeenCalledWith({ title: 'Test Draft', content: 'Some content' })
      expect(result?.title).toBe('Test Draft')
      expect(store.hasDraft).toBe(true)
      expect(store.draft?.courseID).toBeUndefined()
      expect(store.saving).toBe(false)
      expect(store.lastSavedAt).toBeTruthy()
    })

    it('keeps only one cached draft when course changes', async () => {
      mockSaveDraft
        .mockResolvedValueOnce({
          data: {
            data: {
              id: 'draft-1',
              courseID: 100,
              title: 'Course A',
              updatedAt: '2026-04-05T00:00:00Z',
            },
          },
        })
        .mockResolvedValueOnce({
          data: {
            data: {
              id: 'draft-1',
              courseID: 200,
              title: 'Course B',
              updatedAt: '2026-04-05T00:01:00Z',
            },
          },
        })

      const store = useDraftStore()
      await store.saveDraft({ courseID: 100, title: 'Course A' })
      await store.saveDraft({ courseID: 200, title: 'Course B' })

      expect(store.hasDraft).toBe(true)
      expect(store.draft?.courseID).toBe(200)
      expect(store.draft?.title).toBe('Course B')
    })

    it('throws on invalid optional courseID', async () => {
      const store = useDraftStore()
      const invalidDraft: SaveDraftParams = { courseID: Number.NaN, title: 'x' }

      await expect(store.saveDraft(invalidDraft)).rejects.toThrow('invalid draft parameters')
      expect(mockSaveDraft).not.toHaveBeenCalled()
    })

    it('throws when the API response is malformed', async () => {
      mockSaveDraft.mockResolvedValue({ data: {} })

      const store = useDraftStore()
      await expect(store.saveDraft({ title: 'x' })).rejects.toThrow('invalid draft response')
      expect(store.hasDraft).toBe(false)
    })
  })

  describe('loadDraft', () => {
    it('loads the current user draft from API when not cached', async () => {
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'draft-1',
            courseID: 100,
            title: 'Cached',
            content: 'body',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      const result = await store.loadDraft()

      expect(result?.title).toBe('Cached')
      expect(result?.courseID).toBe(100)
      expect(mockGetDraft).toHaveBeenCalledTimes(1)
    })

    it('returns cached user draft without API call', async () => {
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            title: 'First',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.loadDraft()

      mockGetDraft.mockClear()
      const cached = await store.loadDraft()

      expect(mockGetDraft).not.toHaveBeenCalled()
      expect(cached?.title).toBe('First')
    })

    it('returns null and clears cached draft on 404', async () => {
      mockSaveDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })
      mockGetDraft.mockRejectedValue({ status: 404 })

      const store = useDraftStore()
      await store.saveDraft({ title: 'x' })
      const result = await store.loadDraft(true)

      expect(result).toBeNull()
      expect(store.hasDraft).toBe(false)
    })

    it('re-throws non-404 errors', async () => {
      mockGetDraft.mockRejectedValue(new Error('server error'))

      const store = useDraftStore()
      await expect(store.loadDraft()).rejects.toThrow('server error')
    })

    it('throws when the loaded draft contains invalid ratings', async () => {
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            ratings: { teaching: 8 },
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await expect(store.loadDraft()).rejects.toThrow('invalid draft response')
    })
  })

  describe('deleteDraft', () => {
    it('removes the single cached draft', async () => {
      mockDeleteDraft.mockResolvedValue({})
      mockSaveDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.saveDraft({ title: 'x' })
      expect(store.hasDraft).toBe(true)

      await store.deleteDraft()
      expect(mockDeleteDraft).toHaveBeenCalledTimes(1)
      expect(store.hasDraft).toBe(false)
      expect(store.lastSavedAt).toBeNull()
    })

    it('waits for in-flight saves before deleting the draft', async () => {
      const order: string[] = []
      let resolveSave: (value: unknown) => void = () => {}

      mockSaveDraft.mockImplementation(() => new Promise((resolve) => {
        resolveSave = (value: unknown) => {
          order.push('save')
          resolve(value)
        }
      }))
      mockDeleteDraft.mockImplementation(async () => {
        order.push('delete')
        return {}
      })

      const store = useDraftStore()
      const save = store.saveDraft({ title: 'x' })
      const deletion = store.deleteDraft()

      await Promise.resolve()
      expect(mockDeleteDraft).not.toHaveBeenCalled()

      resolveSave({
        data: {
          data: {
            id: 'd1',
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      await Promise.all([save, deletion])
      expect(order).toEqual(['save', 'delete'])
      expect(store.hasDraft).toBe(false)
    })
  })

  describe('reset', () => {
    it('clears the cached draft', async () => {
      mockSaveDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.saveDraft({ title: 'x' })

      store.reset()

      expect(store.hasDraft).toBe(false)
      expect(store.saving).toBe(false)
      expect(store.lastSavedAt).toBeNull()
    })
  })
})
