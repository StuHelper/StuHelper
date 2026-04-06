import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearLocalReviewDraft,
  getLocalReviewDraftClearedAt,
  loadLocalReviewDraft,
  saveLocalReviewDraft,
} from '../reviewDialogDraftState'

class MemoryStorage {
  private store = new Map<string, string>()

  get length() {
    return this.store.size
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null
  }

  getItem(key: string): string | null {
    return this.store.get(key) ?? null
  }

  setItem(key: string, value: string): void {
    this.store.set(key, value)
  }

  removeItem(key: string): void {
    this.store.delete(key)
  }

  clear(): void {
    this.store.clear()
  }
}

describe('reviewDialogDraftState', () => {
  const storage = new MemoryStorage()
  const validRatingKeys = new Set(['recommendation', 'content_quality'])

  beforeEach(() => {
    storage.clear()
    Object.defineProperty(globalThis, 'localStorage', {
      configurable: true,
      value: storage,
    })
  })

  it('stores drafts per course and preserves teacher IDs', () => {
    saveLocalReviewDraft({
      courseID: 1,
      courseName: 'Course A',
      teacherID: 11,
      teacherName: 'Teacher A',
      title: 'Title A',
      content: 'Content A',
      grade: 'A',
      termID: '2025-1',
      ratings: { recommendation: 5, content_quality: 4 },
      updatedAt: 100,
    })
    saveLocalReviewDraft({
      courseID: 2,
      courseName: 'Course B',
      teacherID: 22,
      teacherName: 'Teacher B',
      title: 'Title B',
      content: 'Content B',
      grade: 'B',
      termID: '2025-2',
      ratings: { recommendation: 3, content_quality: 2 },
      updatedAt: 200,
    })

    expect(loadLocalReviewDraft(validRatingKeys, 1)).toMatchObject({
      courseID: 1,
      teacherID: 11,
      teacherName: 'Teacher A',
    })
    expect(loadLocalReviewDraft(validRatingKeys, 2)).toMatchObject({
      courseID: 2,
      teacherID: 22,
      teacherName: 'Teacher B',
    })
    expect(loadLocalReviewDraft(validRatingKeys)).toMatchObject({
      courseID: 2,
      teacherID: 22,
      teacherName: 'Teacher B',
    })
  })

  it('clears drafts for a single course without affecting others', () => {
    saveLocalReviewDraft({
      courseID: 1,
      courseName: 'Course A',
      teacherID: 11,
      title: 'Title A',
      content: 'Content A',
      grade: 'A',
      termID: '2025-1',
      ratings: { recommendation: 5, content_quality: 4 },
      updatedAt: 100,
    })
    saveLocalReviewDraft({
      courseID: 2,
      courseName: 'Course B',
      teacherID: 22,
      title: 'Title B',
      content: 'Content B',
      grade: 'B',
      termID: '2025-2',
      ratings: { recommendation: 3, content_quality: 2 },
      updatedAt: 200,
    })

    clearLocalReviewDraft(2)

    expect(loadLocalReviewDraft(validRatingKeys, 2)).toBeNull()
    expect(loadLocalReviewDraft(validRatingKeys, 1)).toMatchObject({ courseID: 1 })
    expect(getLocalReviewDraftClearedAt(2)).toBeGreaterThan(0)
  })

  it('keeps clearing state scoped by course', () => {
    clearLocalReviewDraft(1)
    expect(getLocalReviewDraftClearedAt(1)).toBeGreaterThan(0)
    expect(getLocalReviewDraftClearedAt(2)).toBe(0)
  })
})
