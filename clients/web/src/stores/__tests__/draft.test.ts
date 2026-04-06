import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { SaveDraftParams } from '@/types/draft'

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

vi.mock('@/types/course', () => ({
  isValidRating: (v: unknown) => typeof v === 'number' && v >= 1 && v <= 5,
}))

const { useDraftStore } = await import('@/stores/draft')

describe('useDraftStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('saveDraft', () => {
    it('saves a draft and updates local cache', async () => {
      const draftData = {
        id: 'draft-1',
        courseID: 100,
        title: 'Test Draft',
        content: 'Some content',
        updatedAt: '2026-04-05T00:00:00Z',
      }
      mockSaveDraft.mockResolvedValue({ data: { data: draftData } })

      const store = useDraftStore()
      const result = await store.saveDraft({ courseID: 100, title: 'Test Draft', content: 'Some content' })

      expect(result).toBeDefined()
      expect(result?.title).toBe('Test Draft')
      expect(store.hasDraft(100)).toBe(true)
      expect(store.saving).toBe(false)
      expect(store.lastSavedAt).toBeTruthy()
    })

    it('rejects invalid params (no courseID)', async () => {
      const store = useDraftStore()
      const invalidDraft: SaveDraftParams = { courseID: 0, title: 'x' }
      const result = await store.saveDraft(invalidDraft)

      expect(result).toBeUndefined()
      expect(mockSaveDraft).not.toHaveBeenCalled()
    })

    it('rejects NaN courseID', async () => {
      const store = useDraftStore()
      const invalidDraft: SaveDraftParams = { courseID: Number.NaN, title: 'x' }
      const result = await store.saveDraft(invalidDraft)

      expect(result).toBeUndefined()
      expect(mockSaveDraft).not.toHaveBeenCalled()
    })
  })

  describe('loadDraft', () => {
    it('loads draft from API when not cached', async () => {
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
      const result = await store.loadDraft(100)

      expect(result).toBeDefined()
      expect(result?.title).toBe('Cached')
      expect(mockGetDraft).toHaveBeenCalledWith(100)
    })

    it('returns cached draft without API call', async () => {
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            courseID: 50,
            title: 'First',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.loadDraft(50)

      mockGetDraft.mockClear()
      const cached = await store.loadDraft(50)

      expect(mockGetDraft).not.toHaveBeenCalled()
      expect(cached?.title).toBe('First')
    })

    it('returns null for invalid courseID', async () => {
      const store = useDraftStore()
      const result = await store.loadDraft(0)
      expect(result).toBeNull()
      expect(mockGetDraft).not.toHaveBeenCalled()
    })

    it('returns null silently on 404', async () => {
      mockGetDraft.mockRejectedValue({ status: 404 })

      const store = useDraftStore()
      const result = await store.loadDraft(999)
      expect(result).toBeNull()
    })

    it('re-throws non-404 errors', async () => {
      mockGetDraft.mockRejectedValue(new Error('server error'))

      const store = useDraftStore()
      await expect(store.loadDraft(999)).rejects.toThrow('server error')
    })
  })

  describe('deleteDraft', () => {
    it('removes draft from local cache', async () => {
      mockDeleteDraft.mockResolvedValue({})
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            courseID: 100,
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.loadDraft(100)
      expect(store.hasDraft(100)).toBe(true)

      await store.deleteDraft(100)
      expect(store.hasDraft(100)).toBe(false)
    })
  })

  describe('getCachedDraft', () => {
    it('returns undefined when no draft cached', () => {
      const store = useDraftStore()
      expect(store.getCachedDraft(999)).toBeUndefined()
    })
  })

  describe('reset', () => {
    it('clears all drafts', async () => {
      mockGetDraft.mockResolvedValue({
        data: {
          data: {
            id: 'd1',
            courseID: 100,
            title: 'x',
            updatedAt: '2026-04-05T00:00:00Z',
          },
        },
      })

      const store = useDraftStore()
      await store.loadDraft(100)

      store.reset()

      expect(store.hasDraft(100)).toBe(false)
      expect(store.saving).toBe(false)
      expect(store.lastSavedAt).toBeNull()
    })
  })
})
