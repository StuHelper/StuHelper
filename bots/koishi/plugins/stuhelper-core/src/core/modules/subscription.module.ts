import type { Command, Context, Session } from 'koishi'

import type { DataManager } from '../data'
import type { Config, Subscription } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'
import { registerSubscriptionCommands } from './subscription-commands'
import {
  checkMuteExpires,
  setupMuteExpireCheck,
  type MuteExpireBot,
} from './subscription-mute-expiry'

interface SubscriptionKey {
  id: string
  type: 'group' | 'private'
}

export class SubscriptionModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'subscription',
    description: '订阅管理模块',
    version: '1.0.0',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null
  private checkInterval: ReturnType<typeof setInterval> | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager
  ) {}

  get config(): Config {
    return getRequiredPluginConfig(this.ctx)
  }

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      this.migrateData()
      registerSubscriptionCommands(this)
      setupMuteExpireCheck(this)
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this.clearCheckInterval()
    this._state = 'unloaded'
  }

  registerCommand(def: RuntimeCommandDef): Command {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  toggleSubscription(session: Session | undefined, feature: keyof Subscription['features']): string {
    const key = getSubscriptionKey(session)
    if (!key) return missingSubscriptionTarget(session)

    const subscriptions = this.data.subscriptions.getAll().list
    let sub = subscriptions.find(item => item.id === key.id && item.type === key.type)
    if (!sub) {
      sub = { type: key.type, id: key.id, features: {} }
      subscriptions.push(sub)
    }

    sub.features ||= {}
    sub.features[feature] = !sub.features[feature]
    this.data.subscriptions.flush()

    return sub.features[feature]
      ? `已订阅${getFeatureName(feature)}喵~`
      : `已取消订阅${getFeatureName(feature)}喵~`
  }

  updateAllSubscriptions(session: Session | undefined, enabled: boolean): string {
    const key = getSubscriptionKey(session)
    if (!key) return missingSubscriptionTarget(session)

    const subscriptions = this.data.subscriptions.getAll().list
    const index = subscriptions.findIndex(item => item.id === key.id && item.type === key.type)

    if (!enabled && index >= 0) {
      subscriptions.splice(index, 1)
      this.data.subscriptions.flush()
      return '已取消所有订阅喵~'
    }
    if (!enabled) return '无需操作喵~'

    const sub: Subscription = index >= 0 ? subscriptions[index] : { ...key, features: {} }
    if (index < 0) subscriptions.push(sub)
    sub.features = {
      log: true,
      memberChange: true,
      muteExpire: true,
      blacklist: true,
      warning: true,
    }
    this.data.subscriptions.flush()
    return '已订阅所有通知喵~'
  }

  showSubscriptionStatus(session: Session | undefined): string {
    const key = getSubscriptionKey(session)
    if (!key) return missingSubscriptionTarget(session)

    const sub = this.data.subscriptions.getAll().list.find(item => item.id === key.id && item.type === key.type)
    if (!sub?.features) return '当前没有任何订阅喵~'

    return [
      `当前订阅状态：`,
      `- 操作日志: ${sub.features.log ? '✅' : '❌'}`,
      `- 成员变动: ${sub.features.memberChange ? '✅' : '❌'}`,
      `- 禁言到期: ${sub.features.muteExpire ? '✅' : '❌'}`,
      `- 黑名单变更: ${sub.features.blacklist ? '✅' : '❌'}`,
      `- 警告通知: ${sub.features.warning ? '✅' : '❌'}`,
    ].join('\n')
  }

  async checkMuteExpires(bot: MuteExpireBot): Promise<void> {
    await checkMuteExpires(this, bot)
  }

  setCheckInterval(interval: ReturnType<typeof setInterval>): void {
    this.checkInterval = interval
  }

  clearCheckInterval(): void {
    if (!this.checkInterval) return
    clearInterval(this.checkInterval)
    this.checkInterval = null
  }

  private migrateData(): void {
    const data = this.data.subscriptions.getAll()
    if (!Array.isArray(data)) return

    this.ctx.logger('stuhelperGroupCenter').info('检测到旧格式的订阅数据 (Array)，正在迁移...')
    this.data.subscriptions.setAll({ list: data as Subscription[] })
    this.data.subscriptions.flush()
    this.ctx.logger('stuhelperGroupCenter').info('订阅数据已迁移到新格式')
  }
}

function getSubscriptionKey(session: Session | undefined): SubscriptionKey | null {
  if (!session) return null

  const id = session.guildId || session.userId
  if (!id) return null

  return {
    id,
    type: session.guildId ? 'group' : 'private',
  }
}

function missingSubscriptionTarget(session: Session | undefined): string {
  return session ? '无法获取订阅ID' : '无法获取会话信息'
}

function getFeatureName(feature: keyof Subscription['features']): string {
  const names: Partial<Record<keyof Subscription['features'], string>> = {
    log: '操作日志',
    memberChange: '成员变动',
    muteExpire: '禁言到期',
    blacklist: '黑名单变更',
    warning: '警告通知',
  }
  return names[feature] || feature
}

export const subscriptionRuntimeModule: RuntimeModule<SubscriptionModule> = {
  id: 'subscription',
  create(ctx, deps) {
    return new SubscriptionModule(ctx, deps.data)
  },
}
