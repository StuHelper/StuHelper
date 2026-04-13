import { describe, expect, it } from 'vitest'
import { resolveNotificationHref, resolveNotificationPresentation, resolveNotificationVisualKind } from '../notification'
import type { Notification } from '../types/business/notification'

const t = (key: string, params?: Record<string, unknown>) => {
  if (key === 'user.notification.items.review_hidden.title') return 'Review hidden'
  if (key === 'user.notification.items.review_hidden.contentWithReason') return `Review hidden: ${params?.reason as string}`
  if (key === 'user.notification.items.review_hidden.label') return 'Moderation'
  if (key === 'user.notification.items.like.title') return 'Liked'
  if (key === 'user.notification.items.like.content') return 'Someone liked your review'
  if (key === 'user.notification.items.like.label') return 'Like'
  return key
}

describe('notification helpers', () => {
  it('renders payload-aware localized content', () => {
    const notification: Notification = {
      id: '1',
      type: 'review_hidden',
      title: 'fallback-title',
      content: 'fallback-content',
      payload: { reason: '违规内容' },
      isRead: false,
      createdAt: '2026-04-08T00:00:00Z',
    }

    expect(resolveNotificationPresentation(notification, t)).toEqual({
      title: 'Review hidden',
      content: 'Review hidden: 违规内容',
      label: 'Moderation',
    })
  })

  it('normalizes legacy vote to like visuals', () => {
    expect(resolveNotificationVisualKind('vote')).toBe('like')
  })

  it('prefers sourceUrl over derived course path', () => {
    const notification: Notification = {
      id: '1',
      type: 'like',
      title: 'x',
      sourceUrl: '/user/student-verification',
      courseID: 42,
      isRead: false,
      createdAt: '2026-04-08T00:00:00Z',
    }

    expect(resolveNotificationHref(notification)).toBe('/user/student-verification')
  })

  it('falls back to legacy related review route when no source fields are present', () => {
    const notification: Notification = {
      id: '1',
      type: 'vote',
      title: 'x',
      relatedType: 'review',
      relatedID: 'review-1',
      isRead: false,
      createdAt: '2026-04-08T00:00:00Z',
    }

    expect(resolveNotificationHref(notification)).toBe('/user/reviews')
  })
})
