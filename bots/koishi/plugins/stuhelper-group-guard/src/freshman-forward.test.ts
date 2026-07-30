import assert from 'node:assert/strict'
import test from 'node:test'

import { MemberGuardService } from './member-guard'

const MATERIAL_HOSTS_ENV = 'STUHELPER_FRESHMAN_MATERIAL_HOSTS'

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

  await withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async (guildID: string, content: string) => {
      sends.push({ guildID, content })
      calls.push(`send:${guildID}`)
      return ['message-1']
    },
  } as any]))

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
    withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
      platform: 'qq',
      selfId: '514',
      sid: 'qq:514',
      sendMessage: async (guildID: string) => {
        attemptedGuilds.push(guildID)
        if (guildID === '9002') throw new Error('send failed')
        return ['message-1']
      },
    } as any])),
    /freshman forward A123 failed for guilds: 9002/,
  )
  assert.deepEqual(attemptedGuilds, ['9001', '9002', '9003'])
  assert.deepEqual(marks, [])
})

test('member guard continues after a poison freshman forward and marks later healthy items', async () => {
  const attemptedGuilds: string[] = []
  const marks: string[] = []
  const warnings: Array<{ message: string, details: unknown }> = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [
          freshmanForwardItem(['9001'], 'https://cdn.example.edu/poison.jpg', 'A-poison'),
          freshmanForwardItem(['9002'], 'https://cdn.example.edu/healthy.jpg', 'A-healthy'),
        ]
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
    logger: {
      error() {},
      warn(message: string, details: unknown) {
        warnings.push({ message, details })
      },
    },
  } as any)

  await assert.rejects(
    withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
      platform: 'qq',
      selfId: '514',
      sid: 'qq:514',
      sendMessage: async (guildID: string) => {
        attemptedGuilds.push(guildID)
        if (guildID === '9001') throw new Error('bot removed from management guild')
        return ['message-1']
      },
    } as any])),
    /freshman forward A-poison failed for guilds: 9001/,
  )

  assert.deepEqual(attemptedGuilds, ['9001', '9002'])
  assert.deepEqual(marks, ['A-healthy'])
  assert.deepEqual(warnings, [{
    message: 'group guard freshman forward failed',
    details: {
      applicationID: 'A-poison',
      phase: 'delivery',
      error: 'freshman forward A-poison failed for guilds: 9001',
    },
  }])
})

test('member guard stops later deliveries when a freshman forward acknowledgement fails', async () => {
  const attemptedGuilds: string[] = []
  const markAttempts: string[] = []
  const warnings: Array<{ message: string, details: unknown }> = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [
          freshmanForwardItem(['9001'], 'https://cdn.example.edu/first.jpg', 'A-first'),
          freshmanForwardItem(['9002'], 'https://cdn.example.edu/second.jpg', 'A-second'),
        ]
      },
      async markFreshmanForwarded(applicationID: string) {
        markAttempts.push(applicationID)
        throw new Error('forward acknowledgement unavailable')
      },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
    },
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: {
      error() {},
      warn(message: string, details: unknown) {
        warnings.push({ message, details })
      },
    },
  } as any)

  await assert.rejects(
    withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
      platform: 'qq',
      selfId: '514',
      sid: 'qq:514',
      sendMessage: async (guildID: string) => {
        attemptedGuilds.push(guildID)
        return ['message-1']
      },
    } as any])),
    /forward acknowledgement unavailable/,
  )

  assert.deepEqual(attemptedGuilds, ['9001'])
  assert.deepEqual(markAttempts, ['A-first'])
  assert.deepEqual(warnings, [{
    message: 'group guard freshman forward failed',
    details: {
      applicationID: 'A-first',
      phase: 'ack',
      error: 'forward acknowledgement unavailable',
    },
  }])
})

test('member guard rejects freshman material URLs outside the explicit host allowlist', async () => {
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [freshmanForwardItem(['9001'], 'https://127.0.0.1/material.jpg')]
      },
      async markFreshmanForwarded() {
        throw new Error('forward should not be marked')
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
    withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
      platform: 'qq',
      selfId: '514',
      sid: 'qq:514',
      sendMessage: async () => { throw new Error('send should not be called') },
    } as any])),
    /freshman material URL host is not public: 127\.0\.0\.1/,
  )
})

async function withMaterialHosts<T>(hosts: string, action: () => Promise<T>) {
  const previous = process.env[MATERIAL_HOSTS_ENV]
  process.env[MATERIAL_HOSTS_ENV] = hosts
  try {
    return await action()
  } finally {
    if (typeof previous === 'undefined') {
      delete process.env[MATERIAL_HOSTS_ENV]
    } else {
      process.env[MATERIAL_HOSTS_ENV] = previous
    }
  }
}

function freshmanForwardItem(
  managementGuildIDs = ['9001', '9002'],
  materialURL = 'https://cdn.example.edu/notice.jpg',
  applicationID = 'A123',
) {
  return {
    application: {
      id: applicationID,
      userID: 42,
      schoolID: 4111010006,
      applicantNameMasked: '张*',
      departmentOrMajor: '计算机科学与技术',
      materialType: 'admission_notice',
      status: 'pending',
      provisionalExpiresAt: '2026-10-01T12:00:00+08:00',
      createdAt: '2026-05-03T12:00:00+08:00',
    },
    materialURL,
    managementGuildIDs,
    schoolName: '北京航空航天大学',
    qqID: '10001',
  }
}
