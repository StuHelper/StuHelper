/**
 * 事件监听模块 - 处理各类事件和定时任务
 */

import type { Context } from 'koishi'
import { createPlatformClient, type PlatformClient } from '@stuhelper/koishi-shared'

import type { DataManager } from '../data'
import type { Config } from '../../types'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { getRequiredPluginConfig } from './module-config'
import {
  registerEventListeners,
  setupEventScheduledTasks,
} from './event-handlers'

export class EventModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'event',
    description: '事件监听模块 - 处理各类事件和定时任务',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(
    readonly ctx: Context,
    readonly data: DataManager,
    readonly admissionPlatform?: Pick<PlatformClient, 'getMemberBlacklistAccess' | 'recordJoinRequestEvent'>,
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
      registerEventListeners(this)
      setupEventScheduledTasks(this)
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this._state = 'unloaded'
  }
}

export const eventRuntimeModule: RuntimeModule<EventModule> = {
  id: 'event',
  create(ctx, deps) {
    const admissionPlatform = deps.coreConfig
      ? createPlatformClient(deps.coreConfig.platform)
      : undefined
    return new EventModule(ctx, deps.data, admissionPlatform)
  },
}
