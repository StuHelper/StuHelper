/**
 * 模块基类
 * 所有功能模块都应继承此类
 */
import { Command, Context, Session } from 'koishi'
import type { DataManager } from '../data'
import type { Config } from '../../types'
import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeCommandDef,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'

/** 模块元信息 */
export type ModuleMeta = RuntimeModuleMeta

/** 命令定义选项 */
export type CommandDef = RuntimeCommandDef

/** 模块状态 */
export type ModuleState = RuntimeModuleState

export abstract class BaseModule {
  /** 模块元信息 */
  abstract readonly meta: ModuleMeta
  /** 模块状态 */
  protected _state: ModuleState = 'unloaded'
  /** 错误信息 */
  protected _error: Error | null = null

  constructor(
    protected ctx: Context,
    protected data: DataManager,
    private _config: Config  // 改为私有，通过 getter 访问
  ) {}

  /**
   * 获取配置（动态获取，支持实时更新）
   * 模块应通过此 getter 访问配置，而非直接使用 _config
   */
  protected get config(): Config {
    // 优先从 stuhelperGroupCenter 服务获取最新配置
    // 这样当 UI 保存设置后，模块可以立即看到更新
    try {
      if (this.ctx.stuhelperGroupCenter?.pluginConfig) {
        return this.ctx.stuhelperGroupCenter.pluginConfig
      }
    } catch {
      // 服务未就绪时使用构造函数传入的配置
    }
    return this._config
  }

  /** 获取模块状态 */
  get state(): ModuleState {
    return this._state
  }

  /** 获取错误信息 */
  get error(): Error | null {
    return this._error
  }

  /**
   * 初始化模块
   * 子类应重写此方法来注册命令、中间件等
   */
  async init(): Promise<void> {
    this._state = 'loading'
    try {
      await this.onInit()
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      console.error(`[${this.meta.name}] 初始化失败:`, error)
      throw error
    }
  }

  /**
   * 子类实现的初始化逻辑
   */
  protected abstract onInit(): Promise<void>

  /**
   * 销毁模块
   */
  async dispose(): Promise<void> {
    try {
      await this.onDispose()
    } catch (error) {
      console.error(`[${this.meta.name}] 销毁失败:`, error)
    }
    this._state = 'unloaded'
  }

  /**
   * 子类实现的销毁逻辑
   */
  protected async onDispose(): Promise<void> {
    // 默认空实现，子类可重写
  }

  /**
   * 获取群配置
   */
  protected getGroupConfig(guildId: string) {
    return this.data.groupConfig.get(guildId)
  }

  /**
   * 记录日志并推送订阅
   */
  protected async log(session: Session, command: string, target: string, result: string, success?:boolean): Promise<void> {
    if(success === false){
      session['_commandFailed'] = true
    }
    await this.ctx.stuhelperGroupCenter.logCommand(session, command, target, result)
  }

  /**
   * 注册命令并自动绑定权限节点
   * @param def 命令定义
   * @returns Koishi Command 对象，可继续链式调用
   *
   * 权限节点命名规则：{模块名}.{命令名}
   * 例如：warn 模块的 add 命令 → warn.add
   */
  protected registerCommand(def: CommandDef): Command {
    return registerRuntimeCommand(this.ctx, this.meta, def)
  }

  /**
   * 注册权限节点（不绑定命令）
   * 用于注册非命令类权限，如 WebUI 操作权限
   */
  protected registerPermission(id: string, name: string, description: string): void {
    const permId = `${this.meta.name}.${id}`
    this.ctx.stuhelperGroupCenter.auth.registerPermission(
      permId,
      name,
      description,
      this.meta.description
    )
  }

  protected logCommand(session: any, command: string, target: string, result: string, success?: boolean): void {
    // 使用 BaseModule 的 log 方法
    this.log(session, command, target, result, success)
  }
  
}
