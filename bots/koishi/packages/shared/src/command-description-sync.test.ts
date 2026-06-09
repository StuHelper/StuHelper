import assert from 'node:assert/strict'
import test from 'node:test'

import { Context } from 'koishi'

import {
  syncAdminCommandDescriptions,
  syncGroupGuardCommandDescriptions,
} from './command-description-sync'

test('syncAdminCommandDescriptions updates Koishi command description i18n entries', () => {
  const ctx = new Context()
  ctx.command('群审状态 [guildId:text]', '旧群审状态说明')
  ctx.command('新生审核通过 <payload:text>', '旧新生审核通过说明')

  syncAdminCommandDescriptions(ctx, {
    guardStatusCommandDescription: '自定义群审状态说明',
    freshmanApproveCommandDescription: '自定义新生审核通过说明',
  })

  assert.equal(commandDescription(ctx, '群审状态'), '自定义群审状态说明')
  assert.equal(commandDescription(ctx, '新生审核通过'), '自定义新生审核通过说明')
})

test('syncGroupGuardCommandDescriptions updates public and admission command descriptions', () => {
  const ctx = new Context()
  ctx.command('举报 <targetMemberId> <reason:text>', '旧举报说明')
  ctx.command('跳过入群认证 <qqID>', '旧跳过说明')

  syncGroupGuardCommandDescriptions(ctx, {
    publicReportCommandDescription: '自定义举报说明',
    admissionSkipCommandDescription: '自定义跳过说明',
  })

  assert.equal(commandDescription(ctx, '举报'), '自定义举报说明')
  assert.equal(commandDescription(ctx, '跳过入群认证'), '自定义跳过说明')
})

function commandDescription(ctx: Context, commandName: string) {
  return ctx.i18n.get(`commands.${commandName}.description`)['']
}
