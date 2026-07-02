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

test('member guard attempts every freshman forward target and still marks forwarded on partial failure', async () => {
  const marks: string[] = []
  const attemptedGuilds: string[] = []
  const warns: Array<{ message: string, context: Record<string, unknown> }> = []
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
    logger: {
      error() {},
      warn(message: string, context: Record<string, unknown>) {
        warns.push({ message, context })
      },
    },
  } as any)

  await withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async (guildID: string) => {
      attemptedGuilds.push(guildID)
      if (guildID === '9002') throw new Error('send failed')
      return ['message-1']
    },
  } as any]))

  assert.deepEqual(attemptedGuilds, ['9001', '9002', '9003'])
  // 后端 forwarded 回执无按群粒度：部分成功也回执，避免下一轮向已送达群重复刷屏
  assert.deepEqual(marks, ['A123'])
  const partialWarn = warns.find((entry) => entry.message === 'freshman forward partially delivered')
  assert.ok(partialWarn)
  assert.equal(partialWarn?.context.applicationID, 'A123')
  assert.deepEqual(partialWarn?.context.deliveredGuildIDs, ['9001', '9003'])
  assert.match(String(partialWarn?.context.failures), /guild 9002: send failed/)
})

test('member guard retries next round without marking when all freshman forward targets fail', async () => {
  const marks: string[] = []
  const warns: string[] = []
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [freshmanForwardItem(['9001', '9002'])]
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
      warn(message: string) {
        warns.push(message)
      },
    },
  } as any)

  await withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => { throw new Error('send failed') },
  } as any]))

  assert.deepEqual(marks, [])
  assert.ok(warns.includes('freshman forward delivery failed for all guilds'))
})

test('member guard skips poisoned freshman forward item and keeps processing the queue', async () => {
  const marks: string[] = []
  const warns: Array<{ message: string, context: Record<string, unknown> }> = []
  const badItem = { ...freshmanForwardItem(['9001']), materialURL: '' }
  const goodItem = {
    ...freshmanForwardItem(['9001']),
    application: { ...freshmanForwardItem().application, id: 'B456' },
  }
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [badItem, goodItem]
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
      warn(message: string, context: Record<string, unknown>) {
        warns.push({ message, context })
      },
    },
  } as any)

  await withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => ['message-1'],
  } as any]))

  // 坏项被跳过并记日志，后续项继续转发
  assert.deepEqual(marks, ['B456'])
  const itemWarn = warns.find((entry) => entry.message === 'freshman forward item failed')
  assert.equal(itemWarn?.context.applicationID, 'A123')
  assert.match(String(itemWarn?.context.error), /missing materialURL/)
})

test('member guard rejects freshman material URLs outside the explicit host allowlist', async () => {
  const warns: Array<{ message: string, context: Record<string, unknown> }> = []
  let marked = false
  const service = new MemberGuardService({
    platform: {
      async listPendingAdmissionActions() { return [] },
      async listPendingFreshmanForwards() {
        return [freshmanForwardItem(['9001'], 'https://127.0.0.1/material.jpg')]
      },
      async markFreshmanForwarded() {
        marked = true
      },
    },
    guardStore: {
      async listBackendSyncPending() { return [] },
    },
    policyStore: {},
    moderationStore: { async appendEvent() {} },
    logger: {
      error() {},
      warn(message: string, context: Record<string, unknown>) {
        warns.push({ message, context })
      },
    },
  } as any)

  await withMaterialHosts('cdn.example.edu', () => service.scanPendingMembers([{
    platform: 'qq',
    selfId: '514',
    sid: 'qq:514',
    sendMessage: async () => { throw new Error('send should not be called') },
  } as any]))

  assert.equal(marked, false)
  const itemWarn = warns.find((entry) => entry.message === 'freshman forward item failed')
  assert.match(String(itemWarn?.context.error), /freshman material URL host is not public: 127\.0\.0\.1/)
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
) {
  return {
    application: {
      id: 'A123',
      userID: '42',
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
