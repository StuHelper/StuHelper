import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const postPageSource = readFileSync(
  resolve(__dirname, '../views/PostReviewPage.vue'),
  'utf-8',
)
const zhReviewSource = readFileSync(
  resolve(__dirname, '../../../i18n/locales/zh-CN/review.ts'),
  'utf-8',
)
const enReviewSource = readFileSync(
  resolve(__dirname, '../../../i18n/locales/en-US/review.ts'),
  'utf-8',
)

describe('review draft behavior', () => {
  it('asks users to discard or load an existing draft on entry', () => {
    expect(postPageSource).toContain(':danger-text="t(\'review.draft.discard\')"')
    expect(postPageSource).toContain('@discard="discardRestorePromptDraft"')
    expect(postPageSource).toContain('async function discardRestorePromptDraft()')
    expect(postPageSource).not.toContain(':cancel-text="t(\'review.draft.restoreSkip\')"')
    expect(postPageSource).not.toContain('@keep="restorePromptDraft = null"')

    expect(zhReviewSource).not.toContain('暂不载入')
    expect(zhReviewSource).not.toContain('保留草稿，不会删除')
    expect(zhReviewSource).not.toContain('restoreSkip')
    expect(enReviewSource).not.toContain('Not Now')
    expect(enReviewSource).not.toContain('Skipping keeps')
    expect(enReviewSource).not.toContain('restoreSkip')
  })

  it('preserves the draft teacher selection after loading the draft course', () => {
    expect(postPageSource).toContain('const draftTeacherIDToRestore = ref<number | null>(null)')
    expect(postPageSource).toContain('draftTeacherIDToRestore.value = draft.teacherID ?? null')
    expect(postPageSource).toContain('restoreDraftTeacherSelection(course)')
  })

  it('does not immediately recreate a draft after the user discards it', () => {
    expect(postPageSource).toContain('const discardedDraftSignature = ref<string | null>(null)')
    expect(postPageSource).toContain('if (currentDraftSignature() === discardedDraftSignature.value) return')
    expect(postPageSource).toContain('discardedDraftSignature.value = currentDraftSignature()')
    expect(postPageSource).toContain('if (autosaveTimer) clearTimeout(autosaveTimer)')
  })

  it('does not autosave the untouched form after discarding the restore prompt draft', () => {
    expect(postPageSource).toContain('const restorePromptDiscarded = ref(false)')
    expect(postPageSource).toContain('if (restorePromptDiscarded.value && !hasMeaningfulDraftInput.value) return')
    expect(postPageSource).toContain('restorePromptDiscarded.value = true')
  })
})
