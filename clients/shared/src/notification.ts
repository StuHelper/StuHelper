import type { Notification, NotificationPayload, NotificationType } from './types/business/notification'

export type {
  Notification,
  NotificationPayload,
  NotificationType,
  UnreadCountResponse,
} from './types/business/notification'

export type NotificationVisualKind = 'reply' | 'like' | 'moderation' | 'success' | 'warning' | 'info'
export type NotificationTranslationValue = string | number | boolean | null | undefined
export type NotificationTranslationParams = Record<string, NotificationTranslationValue>
export type NotificationTranslator = (key: string, params?: NotificationTranslationParams) => string

export interface NotificationPresentation {
  title: string
  content?: string
  label: string
}

type NormalizedNotificationType = Exclude<NotificationType, 'vote'>

const TYPE_PREFIX = 'user.notification.items'

export function normalizeNotificationType(type: NotificationType): NormalizedNotificationType {
  return type === 'vote' ? 'like' : type
}

export function resolveNotificationPresentation(
  notification: Notification,
  translate: NotificationTranslator,
): NotificationPresentation {
  const type = normalizeNotificationType(notification.type)
  const prefix = `${TYPE_PREFIX}.${type}`
  const reason = readPayloadString(notification.payload, 'reason')
  const action = readPayloadString(notification.payload, 'action')

  let contentKey = `${prefix}.content`
  let contentParams: NotificationTranslationParams | undefined

  switch (type) {
    case 'review_hidden':
    case 'identity_rejected':
    case 'student_rejected':
      if (reason) {
        contentKey = `${prefix}.contentWithReason`
        contentParams = { reason }
      }
      break
    case 'report_resolved':
      if (action === 'reject') contentKey = `${prefix}.contentReject`
      if (action === 'hide') contentKey = `${prefix}.contentHide`
      if (action === 'delete') contentKey = `${prefix}.contentDelete`
      break
    default:
      break
  }

  const fallbackTitle = notification.title || notification.type
  const fallbackLabel = notification.type
  const title: string = translateWithFallback(translate, `${prefix}.title`, undefined, fallbackTitle) || fallbackTitle
  const label: string = translateWithFallback(translate, `${prefix}.label`, undefined, fallbackLabel) || fallbackLabel

  return {
    title,
    content: translateWithFallback(translate, contentKey, contentParams, notification.content),
    label,
  }
}

export function resolveNotificationHref(notification: Notification): string | undefined {
  if (typeof notification.sourceUrl === 'string') {
    const sourceUrl = notification.sourceUrl.trim()
    if (sourceUrl.length > 0) {
      return sourceUrl
    }
  }
  if (typeof notification.courseID === 'number' && notification.courseID > 0) {
    return `/courses/${notification.courseID}/reviews`
  }

  switch (normalizeNotificationType(notification.type)) {
    case 'identity_approved':
    case 'identity_rejected':
      return '/user/identity-verification'
    case 'student_approved':
    case 'student_rejected':
      return '/user/student-verification'
    default:
      return undefined
  }
}

export function resolveNotificationVisualKind(type: NotificationType): NotificationVisualKind {
  switch (normalizeNotificationType(type)) {
    case 'reply':
      return 'reply'
    case 'like':
      return 'like'
    case 'review_hidden':
    case 'report_resolved':
      return 'moderation'
    case 'review_restored':
    case 'identity_approved':
    case 'student_approved':
      return 'success'
    case 'identity_rejected':
    case 'student_rejected':
      return 'warning'
    default:
      return 'info'
  }
}

function translateWithFallback(
  translate: NotificationTranslator,
  key: string,
  params: NotificationTranslationParams | undefined,
  fallback?: string,
): string | undefined {
  const translated = translate(key, params)
  if (translated && translated !== key) {
    return translated
  }
  return fallback
}

function readPayloadString(payload: NotificationPayload | undefined, key: string): string | undefined {
  if (!payload || typeof payload !== 'object') return undefined
  const value = payload[key]
  return typeof value === 'string' && value.trim().length > 0 ? value : undefined
}
