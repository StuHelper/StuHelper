import type { Context } from 'koishi'
import type { Client } from '@koishijs/plugin-console'
import type {} from '@koishijs/plugin-auth'

import { ModerationActionService, ModerationStore } from '@stuhelper/koishi-moderation-core'

import {
  handleWorkItemAction,
  type WorkItemActionActor,
  type ReviewActionInput,
  type WorkItemActionInput,
} from './review-action-handler'

export { handleWorkItemAction } from './review-action-handler'

export function registerReviewActionAPI(ctx: Context) {
  if (!ctx.console) {
    return
  }

  const moderationStore = new ModerationStore(ctx)
  const actions = new ModerationActionService(moderationStore)
  const deps = { ctx, moderationStore, actions }

  ctx.console.addListener('stuhelperGroupCenter/action/review', async function (input) {
    return handleWorkItemAction(deps, normalizeLegacyReviewAction(input), resolveActionActor(this))
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/action/work-item', async function (input) {
    return handleWorkItemAction(deps, parseWorkItemActionInput(input), resolveActionActor(this))
  }, { authority: 4 })
}

export function normalizeLegacyReviewAction(input: unknown): ReviewActionInput {
  const record = requireRecord(input, 'review action')
  return {
    kind: 'review',
    itemId: readString(record.reviewId, 'reviewId'),
    action: readEnum(record.action, ['execute', 'reject'], 'action'),
    note: readOptionalString(record.note, 'note'),
  }
}

export function parseWorkItemActionInput(input: unknown): WorkItemActionInput {
  const record = requireRecord(input, 'work item action')
  const kind = readEnum(record.kind, ['review', 'admission', 'report'], 'kind')
  const itemId = readString(record.itemId, 'itemId')
  const note = readOptionalString(record.note, 'note')

  if (kind === 'review') {
    return { kind, itemId, action: readEnum(record.action, ['execute', 'reject'], 'action'), note }
  }
  if (kind === 'admission') {
    return { kind, itemId, action: readEnum(record.action, ['approve', 'deny', 'defer'], 'action'), note }
  }
  return { kind, itemId, action: readEnum(record.action, ['dismiss', 'escalate', 'create-review'], 'action'), note }
}

function requireRecord(input: unknown, label: string) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as Record<string, unknown>
}

function readString(value: unknown, field: string) {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${field} must be a non-empty string`)
  }
  return value.trim()
}

function readOptionalString(value: unknown, field: string) {
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string`)
  }
  return value.trim() || undefined
}

function readEnum<T extends string>(value: unknown, candidates: readonly T[], field: string): T {
  if (typeof value !== 'string' || !candidates.includes(value as T)) {
    throw new Error(`${field} must be one of: ${candidates.join(', ')}`)
  }
  return value as T
}

function resolveActionActor(client: Client): WorkItemActionActor {
  if (!client.auth) {
    throw new Error('console auth is required')
  }

  return {
    memberId: String(client.auth.id),
    displayName: client.auth.name || null,
  }
}
