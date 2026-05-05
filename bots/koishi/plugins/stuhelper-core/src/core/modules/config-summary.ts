import type { Session } from 'koishi'

import type {
  BanMeRecord,
  Config,
  GroupConfig,
  MuteRecord,
  WarnRecord,
} from '../../types'
import type { DataManager } from '../data'
import { formatDuration } from '../../utils'
import {
  listVisibleMemberBlacklists,
  type MemberBlacklistBackend,
} from './member-blacklist-backend'
import type { MemberBlacklistEntry } from '@stuhelper/koishi-shared'

const HOUR_MS = 3_600_000
const SOFT_PITY_STEP = 0.06

type ConfigSummaryHost = {
  readonly data: DataManager
  readonly config: Config
  readonly memberBlacklistBackend?: MemberBlacklistBackend
}

type ConfigSummaryModel = {
  readonly config: Config
  readonly groupConfig?: GroupConfig
  readonly guildWarns: WarnRecord
  readonly blacklist: readonly MemberBlacklistEntry[]
  readonly currentMutes: Record<string, MuteRecord>
  readonly currentBanMe: BanMeRecord
  readonly currentProb: number
  readonly maxDuration: string
  readonly currentGroupKeywords: readonly string[]
  readonly currentGroupApprovalKeywords: readonly string[]
  readonly currentGroupAuto: string
  readonly currentGroupReject: string
  readonly currentGroupAutoDelete: boolean
  readonly currentWelcome: string
}

export async function showAllConfig(host: ConfigSummaryHost, session: Session): Promise<string> {
  const model = await buildSummaryModel(host, session)
  return [
    formatWelcomeSection(model),
    formatApprovalSection(model),
    formatForbiddenSection(model),
    formatAutoMuteSection(model),
    formatAntiRepeatSection(model),
    formatAiSection(model),
    formatTitleSection(model),
    formatWarnSection(model),
    formatBlacklistSection(model),
    formatBanmeSection(model),
    formatMuteSection(model),
  ].join('\n\n')
}

async function buildSummaryModel(host: ConfigSummaryHost, session: Session): Promise<ConfigSummaryModel> {
  const guildId = session.guildId!
  const groupConfigs = host.data.groupConfig.getAll()
  const groupConfig = groupConfigs[guildId]
  const currentBanMe = getCurrentBanMeRecord(host.data.banmeRecords.getAll(), guildId)

  return {
    config: host.config,
    groupConfig,
    guildWarns: cleanupGuildWarns(host, guildId),
    blacklist: await loadVisibleBlacklist(host, session),
    currentMutes: host.data.mutes.getAll()[guildId] || {},
    currentBanMe,
    currentProb: calculateCurrentProbability(currentBanMe, host.config),
    maxDuration: calculateMaxDuration(currentBanMe, host.config),
    currentGroupKeywords: groupConfig?.keywords || [],
    currentGroupApprovalKeywords: groupConfig?.approvalKeywords || [],
    currentGroupAuto: groupConfig?.auto || 'false',
    currentGroupReject: groupConfig?.reject || '答案错误，请重新申请',
    currentGroupAutoDelete: groupConfig?.forbidden?.autoDelete || false,
    currentWelcome: groupConfig?.welcomeMsg || '未设置',
  }
}

function loadVisibleBlacklist(host: ConfigSummaryHost, session: Session) {
  if (!host.memberBlacklistBackend) {
    throw new Error('member blacklist backend client is required for config summary')
  }
  return listVisibleMemberBlacklists(host.memberBlacklistBackend, session.platform, session.guildId!)
}

function formatWelcomeSection(model: ConfigSummaryModel): string {
  return `=== 入群欢迎 ===
默认欢迎语：${model.config.defaultWelcome || '未设置'}
本群欢迎语：${model.currentWelcome || '未设置'}`
}

function formatApprovalSection(model: ConfigSummaryModel): string {
  return `=== 入群审核关键词 ===
全局关键词：${model.config.keywords.join('、') || '无'}
本群关键词：${model.currentGroupApprovalKeywords.join('、') || '无'}
自动拒绝：${model.currentGroupAuto === 'true' ? '已启用' : '未启用'}
拒绝词：${model.currentGroupReject}`
}

function formatForbiddenSection(model: ConfigSummaryModel): string {
  return `=== 禁言关键词 ===
全局关键词：${model.config.forbidden?.keywords?.join('、') || '无'}
本群关键词：${model.currentGroupKeywords.join('、') || '无'}
自动撤回：${model.currentGroupAutoDelete ? '已启用' : '未启用'}
自动禁言：${model.config.forbidden?.autoBan ? '已启用' : '未启用'}
禁言时长：${model.config.forbidden?.muteDuration || 0} 分钟`
}

function formatAutoMuteSection(model: ConfigSummaryModel): string {
  return `=== 自动禁言配置 ===
警告限制：${model.config.warnLimit} 次
禁言时长表达式：${model.config.banTimes.expression}
（{t}代表警告次数）`
}

function formatAntiRepeatSection(model: ConfigSummaryModel): string {
  const antiRepeatConfig = model.groupConfig?.antiRepeat
  return `=== 复读管理 ===
全局状态：${model.config.antiRepeat.enabled ? '已启用' : '未启用'}
全局阈值：${model.config.antiRepeat.threshold} 条
本群状态：${antiRepeatConfig?.enabled ? '已启用' : '未启用'}
本群阈值：${antiRepeatConfig?.threshold || '未设置'} 条`
}

function formatAiSection(model: ConfigSummaryModel): string {
  const groupAiEnabled = model.groupConfig?.openai?.enabled
  return `=== AI功能 ===
全局状态：${model.config.openai?.enabled ? '已启用' : '未启用'}
使用模型：${model.config.openai?.model || 'gpt-3.5-turbo'}
API地址：${model.config.openai?.apiUrl || 'https://api.openai.com/v1'}
本群状态：${groupAiEnabled === undefined ? '跟随全局' : groupAiEnabled ? '已启用' : '已禁用'}`
}

function formatTitleSection(model: ConfigSummaryModel): string {
  return `=== 头衔管理 ===
状态：${model.config.setTitle.enabled ? '已启用' : '未启用'}
权限要求：${model.config.setTitle.authority} 级
最大长度：${model.config.setTitle.maxLength} 字节`
}

function formatWarnSection(model: ConfigSummaryModel): string {
  return `=== 警告记录 ===
${formatWarns(model.guildWarns) || '无记录'}`
}

function formatBlacklistSection(model: ConfigSummaryModel): string {
  return `=== 黑名单 ===
${formatBlacklist(model.blacklist) || '无记录'}`
}

function formatBanmeSection(model: ConfigSummaryModel): string {
  return `=== Banme 统计 ===
状态：${model.config.banme.enabled ? '已启用' : '未启用'}
本群1小时内使用：${model.currentBanMe.count} 次
当前抽数：${model.currentBanMe.pity}
当前概率：${(model.currentProb * 100).toFixed(2)}%
状态：${model.currentBanMe.guaranteed ? '大保底' : '普通'}
当前最大禁言：${model.maxDuration}
自动禁言：${model.config.banme.autoBan ? '已启用' : '未启用'}`
}

function formatMuteSection(model: ConfigSummaryModel): string {
  return `=== 当前禁言 ===
${formatMutes(model.currentMutes) || '无记录'}`
}

function cleanupGuildWarns(host: ConfigSummaryHost, guildId: string): WarnRecord {
  const guildWarns = host.data.warns.get(guildId) || {}
  let hasChanges = false

  for (const userId in guildWarns) {
    if (guildWarns[userId].count <= 0) {
      delete guildWarns[userId]
      hasChanges = true
    }
  }

  if (hasChanges) {
    host.data.warns.set(guildId, guildWarns)
    host.data.warns.flush()
  }
  return guildWarns
}

function getCurrentBanMeRecord(
  banMeRecords: Record<string, BanMeRecord>,
  guildId: string,
): BanMeRecord {
  const record = banMeRecords[guildId] || {
    count: 0,
    lastResetTime: Date.now(),
    pity: 0,
    guaranteed: false,
  }

  if (Date.now() - record.lastResetTime > HOUR_MS) {
    record.count = 0
  }
  return record
}

function calculateMaxDuration(record: BanMeRecord, config: Config): string {
  const baseMaxMillis = config.banme.baseMax * 60 * 1000
  const additionalMinutes = Math.floor(Math.pow(record.count, 1 / 3) * config.banme.growthRate)
  return formatDuration(baseMaxMillis + (additionalMinutes * 60 * 1000))
}

function calculateCurrentProbability(record: BanMeRecord, config: Config): number {
  let currentProb = config.banme.jackpot.baseProb
  if (record.pity >= config.banme.jackpot.softPity) {
    currentProb = config.banme.jackpot.baseProb +
      (record.pity - config.banme.jackpot.softPity + 1) * SOFT_PITY_STEP
  }
  return Math.min(currentProb, 1)
}

function formatBlacklist(entries: readonly MemberBlacklistEntry[]): string {
  return entries
    .map((entry) => `用户 ${entry.subjectID}：${formatBlacklistScope(entry)} / ${formatShanghaiTime(Date.parse(entry.createdAt))}`)
    .join('\n')
}

function formatBlacklistScope(entry: MemberBlacklistEntry): string {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID}`
}

function formatWarns(guildWarns: WarnRecord): string {
  return Object.entries(guildWarns)
    .filter(([, data]) => data.count > 0)
    .map(([userId, data]) => `用户 ${userId}：${data.count} 次 (${formatShanghaiTime(data.timestamp)})`)
    .filter(Boolean)
    .join('\n')
}

function formatMutes(currentMutes: Record<string, MuteRecord>): string {
  return Object.entries(currentMutes)
    .filter(([, data]) => !data.leftGroup && Date.now() - data.startTime < data.duration)
    .map(([userId, data]) => {
      const remainingTime = data.duration - (Date.now() - data.startTime)
      return `用户 ${userId}：剩余 ${formatDuration(remainingTime)}`
    })
    .join('\n')
}

function formatShanghaiTime(timestamp: number): string {
  return new Date(timestamp).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' })
}
