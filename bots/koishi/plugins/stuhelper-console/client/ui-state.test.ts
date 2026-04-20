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
  assert.equal(sections[2]?.items[0]?.value, '重复刷屏')
})
