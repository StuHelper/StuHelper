import assert from 'node:assert/strict'
import test from 'node:test'

import { MemberGuardService } from './member-guard'

test('member guard forwards freshman material to every management group before marking forwarded', async () => {
  const sends: Array<{ guildID: string, content: string }> = []
  const calls: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [freshmanForwardItem()]
      },
      async markFreshmanForwarded(applicationID: string) {
        calls.push(`mark:${applicationID}`)
      },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
    },
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await service.scanPendingMembers([{
    platform: 'mock',
    selfId: '514',
    sid: 'mock:514',
    sendMessage: async (guildID: string, content: string) => {
      sends.push({ guildID, content })
      calls.push(`send:${guildID}`)
      return ['message-1']
    },
  } as any])

  assert.deepEqual(calls, ['send:9001', 'send:9002', 'mark:A123'])
  assert.deepEqual(sends.map((item) => item.guildID), ['9001', '9002'])
  assert.match(sends[0].content, /https:\/\/cdn\.example\.edu\/notice\.jpg/)
  assert.match(sends[0].content, /A123/)
  assert.match(sends[0].content, /张\*/)
  assert.match(sends[0].content, /北京航空航天大学/)
  assert.match(sends[0].content, /计算机科学与技术/)
  assert.match(sends[0].content, /10001/)
  assert.match(sends[0].content, /新生审核通过 A123/)
  assert.match(sends[0].content, /新生审核驳回 A123 <原因>/)
})

test('member guard attempts every freshman forward target and does not mark when any send fails', async () => {
  const marks: string[] = []
  const attemptedGuilds: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [freshmanForwardItem(['9001', '9002', '9003'])]
      },
      async markFreshmanForwarded(applicationID: string) {
        marks.push(applicationID)
      },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
    },
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: { error() {}, warn() {} },
  } as any)

  await assert.rejects(
    service.scanPendingMembers([{
      platform: 'mock',
      selfId: '514',
      sid: 'mock:514',
      sendMessage: async (guildID: string) => {
        attemptedGuilds.push(guildID)
        if (guildID === '9002') throw new Error('send failed')
        return ['message-1']
      },
    } as any]),
    /freshman forward A123 failed for guilds: 9002/,
  )
  assert.deepEqual(attemptedGuilds, ['9001', '9002', '9003'])
  assert.deepEqual(marks, [])
})

function freshmanForwardItem(managementGuildIDs = ['9001', '9002']) {
  return {
    application: {
      id: 'A123',
      userID: 42,
      schoolID: 1,
      applicantNameMasked: '张*',
      departmentOrMajor: '计算机科学与技术',
      materialType: 'admission_notice',
      status: 'pending',
      provisionalExpiresAt: '2026-10-01T12:00:00+08:00',
      createdAt: '2026-05-03T12:00:00+08:00',
    },
    materialURL: 'https://cdn.example.edu/notice.jpg',
    managementGuildIDs,
    schoolName: '北京航空航天大学',
    qqID: '10001',
  }
}
