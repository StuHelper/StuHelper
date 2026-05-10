import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'

import enUS from '../locales/en-US'
import zhCN from '../locales/zh-CN'

function createTestI18n(locale: 'en-US' | 'zh-CN') {
  return createI18n({
    legacy: false,
    locale,
    messages: {
      'en-US': enUS,
      'zh-CN': zhCN,
    },
  })
}

describe('review locale messages', () => {
  it('renders literal at signs without vue-i18n compile errors', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
    try {
      const en = createTestI18n('en-US').global.t
      const zh = createTestI18n('zh-CN').global.t

      expect(en('review.about.title')).toBe('About Course Review@BUAA')
      expect(en('review.about.contactEmail')).toBe('stuhelper@protonmail.com')
      expect(zh('review.about.title')).toBe('关于评课社区@BUAA')
      expect(zh('review.about.contactEmail')).toBe('stuhelper@protonmail.com')
      expect(errorSpy).not.toHaveBeenCalled()
    } finally {
      errorSpy.mockRestore()
    }
  })

  it('defines review rating card icon messages for known dimensions', () => {
    const t = createTestI18n('zh-CN').global.t

    expect(t('review.ratingEmoji.icon.difficulty')).not.toBe('review.ratingEmoji.icon.difficulty')
    expect(t('review.ratingEmoji.icon.grading')).not.toBe('review.ratingEmoji.icon.grading')
    expect(t('review.ratingEmoji.icon.teaching')).not.toBe('review.ratingEmoji.icon.teaching')
    expect(t('review.ratingEmoji.icon.usefulness')).not.toBe('review.ratingEmoji.icon.usefulness')
    expect(t('review.ratingEmoji.icon.workload')).not.toBe('review.ratingEmoji.icon.workload')
  })
})
