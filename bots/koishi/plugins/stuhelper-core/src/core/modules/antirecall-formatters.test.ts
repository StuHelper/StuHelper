import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { formatStatusMessage, parseConfigUpdates } from './antirecall-formatters'

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('antirecall command helpers use typed options, status and error boundaries', () => {
  const formatterSource = readModuleFile('./antirecall-formatters.ts')
  const commandSource = readModuleFile('./antirecall-commands.ts')
  const moduleSource = readModuleFile('./antirecall.module.ts')

  for (const source of [formatterSource, commandSource, moduleSource]) {
    assert.doesNotMatch(source, /options: any/)
    assert.doesNotMatch(source, /updates: any/)
    assert.doesNotMatch(source, /status: any/)
    assert.doesNotMatch(source, /\$\{error\.message\}/)
    assert.doesNotMatch(source, /catch \(error: any\)/)
  }

  assert.match(formatterSource, /export interface AntiRecallConfigOptions/)
  assert.match(formatterSource, /export interface AntiRecallStatus/)
  assert.match(commandSource, /function errorMessage\(error: unknown\): string/)
  assert.match(moduleSource, /getStatus\(guildId: string\): AntiRecallStatus/)
})

test('parseConfigUpdates normalizes typed boolean and numeric options', () => {
  assert.deepEqual(parseConfigUpdates({ enabled: 'on', days: '14', max: 200 }), {
    updates: {
      enabled: true,
      retentionDays: 14,
      maxRecordsPerUser: 200,
    },
    messages: ['已启用防撤回', '保留天数设为 14 天', '最大记录数设为 200 条'],
  })

  assert.deepEqual(parseConfigUpdates({ enabled: 'off', days: 0, max: 'many' }), {
    updates: {
      enabled: false,
    },
    messages: ['已禁用防撤回', '保留天数无效 (需 1-365)', '最大记录数无效 (需 1-1000)'],
  })
})

test('formatStatusMessage renders typed antirecall status totals', () => {
  const message = formatStatusMessage({
    globalEnabled: true,
    groupSpecificEnabled: false,
    effectiveConfig: {
      enabled: false,
      retentionDays: 30,
      maxRecordsPerUser: 300,
      showOriginalTime: true,
    },
    statistics: {
      totalRecords: 12,
      totalUsers: 3,
      totalGuilds: 2,
    },
  })

  assert.match(message, /防撤回功能状态/)
  assert.match(message, /全局默认: 已启用/)
  assert.match(message, /本群设置: 已单独设置为: 已禁用/)
  assert.match(message, /总记录数: 12/)
  assert.match(message, /涉及用户数: 3/)
  assert.match(message, /涉及群组数: 2/)
})

function readModuleFile(relativePath: string): string {
  return readFileSync(resolve(modulesDir, relativePath), 'utf8')
}
