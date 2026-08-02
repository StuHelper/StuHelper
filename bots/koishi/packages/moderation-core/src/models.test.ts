import assert from 'node:assert/strict'
import test from 'node:test'

import { MODERATION_EVENT_TABLE, MODERATION_MESSAGE_LEDGER_TABLE } from './constants'
import { registerModerationModels } from './models'

test('message ledger model declares the repeat-query index', () => {
  const extensions = new Map<string, {
    fields: Record<string, unknown>
    config: Record<string, unknown>
  }>()

  registerModerationModels({
    model: {
      extend(
        table: string,
        fields: Record<string, unknown>,
        config: Record<string, unknown>,
      ) {
        extensions.set(table, { fields, config })
      },
    },
  } as never)

  const ledger = extensions.get(MODERATION_MESSAGE_LEDGER_TABLE)
  assert.ok(ledger)
  assert.deepEqual(ledger.config, {
    primary: 'messageId',
    indexes: [{
      keys: {
        guildId: 'asc',
        createdAt: 'desc',
      },
    }],
  })

  const events = extensions.get(MODERATION_EVENT_TABLE)
  assert.ok(events)
  assert.deepEqual(events.config, {
    primary: 'id',
    indexes: [{ keys: { createdAt: 'desc' } }],
  })
})
