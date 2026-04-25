import assert from 'node:assert/strict'
import test from 'node:test'

import { buildInspectorSections, resolveNoticeMessage } from './ui-state'

test('resolveNoticeMessage falls back when task returns empty content', () => {
  assert.equal(resolveNoticeMessage(''), '操作已提交并刷新。')
  assert.equal(resolveNoticeMessage(null, '已完成'), '已完成')
  assert.equal(resolveNoticeMessage('保存成功'), '保存成功')
})

test('buildInspectorSections keeps drawer sections in stable order', () => {
  const sections = buildInspectorSections('review', {
    memberId: 'member_1',
    actionType: 'kick',
    status: 'pending',
    createdAt: '2026-04-20T10:00:00.000Z',
    resolutionNote: '待人工确认',
    reason: '重复刷屏',
  })

  assert.deepEqual(
    sections.map((section) => section.title),
    ['基本信息', '上下文记录', '风险与说明'],
  )
  assert.deepEqual(
    sections[0]?.items.map((item) => item.label),
    ['成员', '动作', '状态'],
  )
  assert.equal(sections[0]?.items[1]?.value, '踢出成员')
  assert.equal(sections[0]?.items[2]?.value, '待复核')
  assert.equal(sections[2]?.items[0]?.value, '重复刷屏')
})

test('buildInspectorSections includes report context and translates review fields', () => {
  const reportSections = buildInspectorSections('report', {
    guildId: 'guild_1',
    channelId: 'channel_1',
    platform: 'mock',
    reporterMemberId: 'reporter_1',
    targetMemberId: 'target_1',
    aiStatus: 'completed',
    aiSeverity: 'high',
    createdAt: '2026-04-20T10:00:00.000Z',
    aiSummary: '高风险',
    reason: '恶意引流',
  })

  assert.equal(reportSections[1]?.items[0]?.label, '群号')
  assert.equal(reportSections[1]?.items[0]?.value, 'guild_1')
  assert.equal(reportSections[1]?.items[1]?.label, '频道')
  assert.equal(reportSections[1]?.items[2]?.label, '平台')
})
