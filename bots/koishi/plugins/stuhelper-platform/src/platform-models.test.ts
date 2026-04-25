import assert from 'node:assert/strict'
import test from 'node:test'

import {
  AUDIT_EVENT_TABLE,
  GUILD_POLICY_TABLE,
  MODULE_CONFIG_TABLE,
  MODULE_STATE_TABLE,
  PERMISSION_POLICY_TABLE,
} from './constants'
import { registerPlatformModels } from './platform-models'

type Row = Record<string, unknown>

interface RegisteredModel {
  readonly table: string
  readonly fields: Row
  readonly config: Row
}

test('registerPlatformModels registers exactly platform tables', () => {
  const registeredModels: RegisteredModel[] = []

  registerPlatformModels({
    model: {
      extend(table: string, fields: Row, config: Row): void {
        registeredModels.push({ table, fields, config })
      },
    },
  } as never)

  assert.deepEqual(getRegisteredTables(registeredModels), [
    MODULE_STATE_TABLE,
    MODULE_CONFIG_TABLE,
    PERMISSION_POLICY_TABLE,
    GUILD_POLICY_TABLE,
    AUDIT_EVENT_TABLE,
  ])
  assert.equal(registeredModels.length, 5)
  assertPrimaryKeys(registeredModels)
  assert.equal(registeredModels[1].fields.config.type, 'json')
  assert.equal(registeredModels[3].fields.config.type, 'json')
})

function getRegisteredTables(models: readonly RegisteredModel[]): string[] {
  return models.map((model) => model.table)
}

function assertPrimaryKeys(models: readonly RegisteredModel[]): void {
  for (const model of models) {
    assert.equal(model.config.primary, 'id')
  }
}
