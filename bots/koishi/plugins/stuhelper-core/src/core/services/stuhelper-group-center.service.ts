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
import type { RuntimeModuleInstance } from '../../runtime/types'
import {
  collectWarmCacheTargets,
  formatShanghaiTimestamp,
  toErrorMessage,
} from './stuhelper-group-center.utils'
import { redactSensitiveText } from '../modules/log-redaction'
import { resolveRequiredConsoleGuildScope } from '../api/console-guild-scope'

const CACHE_WARM_DELAY_MS = 2_000

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

  private readonly context: Context
  /** 数据管理器 */
  private _data: DataManager
  /** 功能模块注册表 */
  private _modules: Map<string, RuntimeModuleInstance> = new Map()
  /** 设置管理器 */
  private _settingsManager: SettingsManager
  /** 缓存服务 */
  private _cache: CacheService
  /** 权限服务 */
  private _auth: AuthService
  /** 日志 */
  private readonly serviceLogger
  /** 缓存预热定时器 */
  private warmCacheTimer: NodeJS.Timeout | null = null

  constructor(ctx: Context) {
    super(ctx, 'stuhelperGroupCenter')
    this.context = ctx
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

  resolveConsoleGuildScope(client: unknown) {
    return resolveRequiredConsoleGuildScope(client, {
      roles: this._auth.getRoles(),
      getUserRoleIds: (userId: string) => this._auth.getUserRoleIds(userId),
      listBindingsByAuthId: (authId: number) => this.context.database.get('binding', { aid: authId }),
    })
  }

  /**
   * 注册模块
   */
  registerModule(module: RuntimeModuleInstance): void {
    const name = module.meta.name
    if (this._modules.has(name)) {
      this.serviceLogger.warn('模块 %s 已存在，将被覆盖', name)
    }
    this._modules.set(name, module)
    this.serviceLogger.info('注册模块: %s', name)
  }

  /**
   * 获取模块
   */
  getModule<T extends RuntimeModuleInstance>(name: string): T | undefined {
    return this._modules.get(name) as T | undefined
  }

  /**
   * 获取所有模块
   */
  getAllModules(): RuntimeModuleInstance[] {
    return Array.from(this._modules.values())
  }

  /**
   * 初始化所有模块
   */
  async initModules(): Promise<void> {
    for (const [name, module] of this._modules) {
      try {
        await module.init()
        this.serviceLogger.info('模块 %s 初始化完成', name)
      } catch (error) {
        this.serviceLogger.error('模块 %s 初始化失败: %s', name, toErrorMessage(error))
        throw error
      }
    }

    this.warmCacheAsync()
  }

  /**
   * 异步预热缓存（不阻塞启动）
   */
  private warmCacheAsync(): void {
    if (this.warmCacheTimer) {
      clearTimeout(this.warmCacheTimer)
    }

    this.warmCacheTimer = setTimeout(() => {
      void this.runWarmCache()
    }, CACHE_WARM_DELAY_MS)
  }

  private async runWarmCache(): Promise<void> {
    try {
      const targets = collectWarmCacheTargets(this._data, this.getSubscriptions())
      await this._cache.warmCache(targets.guildIds, targets.userIds, targets.memberPairs)
    } catch (error) {
      this.serviceLogger.error('缓存预热失败: %s', toErrorMessage(error))
    } finally {
      this.warmCacheTimer = null
    }
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
    if (this.warmCacheTimer) {
      clearTimeout(this.warmCacheTimer)
      this.warmCacheTimer = null
    }

    for (const [name, module] of this._modules) {
      try {
        module.dispose()
        this.serviceLogger.info('模块 %s 已销毁', name)
      } catch (error) {
        this.serviceLogger.error('模块 %s 销毁失败: %s', name, toErrorMessage(error))
      }
    }
    this._modules.clear()

    this._data.dispose()
    this._settingsManager.dispose()
  }
}
