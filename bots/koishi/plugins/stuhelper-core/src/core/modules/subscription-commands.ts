import type { Session } from 'koishi'

import type { Subscription } from '../../types'
import type { SubscriptionModule } from './subscription.module'

type SubscriptionFeature = keyof Subscription['features']

interface FeatureCommand {
  name: string
  desc: string
  permNode: string
  permDesc: string
  usage: string
  feature: SubscriptionFeature
}

const HELP_TEXT = `使用以下命令管理订阅：
sub log - 操作日志订阅
sub member - 成员变动通知
sub mute - 禁言到期通知
sub blacklist - 黑名单变更通知
sub warning - 警告通知
sub all - 订阅所有通知
sub none - 取消所有订阅
sub status - 查看订阅状态`

const FEATURE_COMMANDS: readonly FeatureCommand[] = [
  {
    name: 'sub.log',
    desc: '订阅操作日志',
    permNode: 'sub.log',
    permDesc: '订阅操作日志',
    usage: '开启/关闭操作日志推送',
    feature: 'log',
  },
  {
    name: 'sub.member',
    desc: '订阅成员变动',
    permNode: 'sub.member',
    permDesc: '订阅成员变动',
    usage: '开启/关闭成员加入退出通知',
    feature: 'memberChange',
  },
  {
    name: 'sub.mute',
    desc: '订阅禁言到期通知',
    permNode: 'sub.mute',
    permDesc: '订阅禁言到期通知',
    usage: '开启/关闭禁言到期提醒',
    feature: 'muteExpire',
  },
  {
    name: 'sub.blacklist',
    desc: '订阅黑名单变更',
    permNode: 'sub.blacklist',
    permDesc: '订阅黑名单变更',
    usage: '开启/关闭黑名单变更通知',
    feature: 'blacklist',
  },
  {
    name: 'sub.warning',
    desc: '订阅警告通知',
    permNode: 'sub.warning',
    permDesc: '订阅警告通知',
    usage: '开启/关闭警告处理通知',
    feature: 'warning',
  },
]

export function registerSubscriptionCommands(host: SubscriptionModule): void {
  registerHelpCommand(host)
  FEATURE_COMMANDS.forEach(command => registerFeatureCommand(host, command))
  registerAllCommand(host)
  registerNoneCommand(host)
  registerStatusCommand(host)
}

function registerHelpCommand(host: SubscriptionModule): void {
  host.registerCommand({
    name: 'sub',
    desc: '订阅管理',
    permNode: 'sub',
    permDesc: '订阅管理帮助',
    usage: '管理各类通知订阅，使用子命令操作',
  }).action(async () => HELP_TEXT)
}

function registerFeatureCommand(host: SubscriptionModule, command: FeatureCommand): void {
  host.registerCommand(command).action(async ({ session }) => {
    return host.toggleSubscription(session as Session | undefined, command.feature)
  })
}

function registerAllCommand(host: SubscriptionModule): void {
  host.registerCommand({
    name: 'sub.all',
    desc: '订阅所有通知',
    permNode: 'sub.all',
    permDesc: '订阅所有通知',
    usage: '一键开启所有类型的通知订阅',
  }).action(async ({ session }) => {
    return host.updateAllSubscriptions(session as Session | undefined, true)
  })
}

function registerNoneCommand(host: SubscriptionModule): void {
  host.registerCommand({
    name: 'sub.none',
    desc: '取消所有订阅',
    permNode: 'sub.none',
    permDesc: '取消所有订阅',
    usage: '一键关闭所有类型的通知订阅',
  }).action(async ({ session }) => {
    return host.updateAllSubscriptions(session as Session | undefined, false)
  })
}

function registerStatusCommand(host: SubscriptionModule): void {
  host.registerCommand({
    name: 'sub.status',
    desc: '查看订阅状态',
    permNode: 'sub.status',
    permDesc: '查看订阅状态',
    usage: '查看当前群/私聊的订阅状态',
  }).action(async ({ session }) => {
    return host.showSubscriptionStatus(session as Session | undefined)
  })
}
