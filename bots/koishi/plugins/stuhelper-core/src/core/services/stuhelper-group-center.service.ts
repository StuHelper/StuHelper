/**
 * StuHelper 群管中心主服务
 * 使用 Koishi Service 模式注册为 ctx.stuhelperGroupCenter
 */
import { Context, Service } from 'koishi'
import { DataManager } from '../data'
import { SettingsManager, PluginSettings } from '../settings'
import { CacheService } from './cache.service'
import { AuthService } from './auth.service'
import type { Subscription } from '../../types'
import {
  formatShanghaiTimestamp,
  toErrorMessage,
} from './stuhelper-group-center.utils'
import { redactSensitiveText } from '../data/log-redaction'

export type PushMessageBot = {
  sendMessage(channelId: string, content: string): Promise<unknown>
  sendPrivateMessage(userId: string, content: string): Promise<unknown>
}

type CommandLogSession = {
  userId?: string
  username?: string
  guildId?: string
  bot: PushMessageBot
}

interface CoreCommandLogInput {
  readonly session: CommandLogSession
  readonly command: string
  readonly target: string
  readonly result: string
}

// 声明服务类型
declare module 'koishi' {
  interface Context {
    stuhelperGroupCenter: StuhelperGroupCenterService
  }
}

export class StuhelperGroupCenterService extends Service {
  static inject = ['database']

  /** 数据管理器 */
  private _data: DataManager
  /** 功能模块注册表 */
  /** 设置管理器 */
  private _settingsManager: SettingsManager
  /** 缓存服务 */
  private _cache: CacheService
  /** 权限服务 */
  private _auth: AuthService
  /** 日志 */
  private readonly serviceLogger
  /** 缓存预热定时器 */

  constructor(ctx: Context) {
    super(ctx, 'stuhelperGroupCenter')
    this.serviceLogger = ctx.logger('stuhelperGroupCenter')
    this._data = new DataManager(ctx)
    this._settingsManager = new SettingsManager(this._data.dataPath, this.serviceLogger)
    this._cache = new CacheService(ctx, this._data.dataPath)
    this._auth = new AuthService(ctx, this._data)
  }

  /** 获取数据管理器 */
  get data(): DataManager {
    return this._data
  }

  /** 获取设置管理器 */
  get settings(): SettingsManager {
    return this._settingsManager
  }

  /** 获取插件配置（兼容旧代码） */
  get pluginConfig(): PluginSettings {
    return this._settingsManager.settings
  }

  /** 获取缓存服务 */
  get cache(): CacheService {
    return this._cache
  }

  /** 获取权限服务 */
  get auth(): AuthService {
    return this._auth
  }

  /**
   * 获取订阅列表
   */
  getSubscriptions(): Subscription[] {
    return this._data.subscriptions.get('list') || []
  }

  /**
   * 添加订阅
   */
  addSubscription(subscription: Subscription): void {
    const list = this.getSubscriptions()
    const exists = list.find(
      s => s.type === subscription.type && s.id === subscription.id
    )
    if (!exists) {
      list.push(subscription)
      this._data.subscriptions.set('list', list)
    }
  }

  /**
   * 移除订阅
   */
  removeSubscription(type: string, id: string): boolean {
    const list = this.getSubscriptions()
    const index = list.findIndex(s => s.type === type && s.id === id)
    if (index !== -1) {
      list.splice(index, 1)
      this._data.subscriptions.set('list', list)
      return true
    }
    return false
  }

  /**
   * 向订阅者推送消息
   */
  async pushMessage(
    bot: PushMessageBot,
    message: string,
    feature: keyof Subscription['features']
  ): Promise<void> {
    const subscriptions = this.getSubscriptions()
    for (const sub of subscriptions) {
      try {
        if (!sub.features) continue
        if (sub.features[feature]) {
          if (sub.type === 'group') {
            await bot.sendMessage(sub.id, message)
          } else {
            await bot.sendPrivateMessage(sub.id, message)
          }
        }
      } catch (error) {
        this.serviceLogger.error('推送消息失败: %s', toErrorMessage(error))
      }
    }
  }

  /**
   * 记录命令日志
   */
  async logCommand(input: CoreCommandLogInput): Promise<void> {
    const { session, command, target, result } = input
    const user = session.userId || session.username
    const group = session.guildId || 'private'
    const time = formatShanghaiTimestamp(new Date())
    const message = redactSensitiveText(`[${command}] 用户(${user}) 群(${group}) 目标(${target}): ${result}`)
    this._data.writeLog(message)

    await this.pushMessage(session.bot, `[${time}] ${message}`, 'log')
  }

  /**
   * 服务停止时调用
   */
  protected stop(): void {
    this._data.dispose()
    this._settingsManager.dispose()
  }
}
