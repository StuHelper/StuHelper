/**
 * 设置管理器
 * 管理插件的所有配置，从 settings.json 加载配置
 * 替代原来的 Koishi Schema 配置
 */
import { JsonDataStore } from '../data/json.store'
import * as path from 'path'
import * as fs from 'fs'

import { Config as PluginSettings } from '../../types'
import { CONTEXT_REPORT_PROMPT, DEFAULT_REPORT_PROMPT } from '../modules/report-prompts'
import { DEFAULT_TRANSLATE_PROMPT } from './default-prompts'
import { deepMerge, getDiff } from './settings-merge'

export type { PluginSettings }

interface SettingsLogger {
  info(message: string, ...args: unknown[]): void
  error(message: string, ...args: unknown[]): void
}

/** 默认配置 - 从原 config/index.ts 提取 */
export const DEFAULT_SETTINGS: PluginSettings = {
  keywords: [],
  warnLimit: 3,
  banTimes: {
    expression: '{t}^2h'
  },
  forbidden: {
    autoDelete: false,
    autoBan: false,
    autoKick: false,
    muteDuration: 600000,
    keywords: []
  },
  dice: {
    enabled: true,
    lengthLimit: 1000
  },
  banme: {
    enabled: true,
    baseMin: 1,
    baseMax: 30,
    growthRate: 30,
    autoBan: false,
    jackpot: {
      enabled: true,
      baseProb: 0.006,
      softPity: 73,
      hardPity: 89,
      upDuration: '24h',
      loseDuration: '12h'
    }
  },
  friendRequest: {
    enabled: false,
    keywords: [],
    rejectMessage: '请输入正确的验证信息'
  },
  guildRequest: {
    enabled: false,
    rejectMessage: '暂不接受入群邀请'
  },
  setTitle: {
    enabled: true,
    authority: 3,
    maxLength: 18
  },
  antiRepeat: {
    enabled: false,
    threshold: 3
  },
  openai: {
    enabled: false,
    chatEnabled: true,
    translateEnabled: true,
    apiKey: '',
    apiUrl: 'https://api.openai.com/v1',
    model: 'gpt-3.5-turbo',
    systemPrompt: '你是一个有帮助的AI助手，请简短、准确地回答问题。',
    translatePrompt: DEFAULT_TRANSLATE_PROMPT,
    maxTokens: 2048,
    temperature: 0.7,
    contextLimit: 10
  },
  report: {
    enabled: true,
    authority: 1,
    autoProcess: true,
    defaultPrompt: DEFAULT_REPORT_PROMPT,
    contextPrompt: CONTEXT_REPORT_PROMPT,
    maxReportTime: 30,
    guildConfigs: {},
    maxReportCooldown: 60,
    minAuthorityNoLimit: 2
  },
  antiRecall: {
    enabled: false,
    retentionDays: 7,
    maxRecordsPerUser: 50,
    showOriginalTime: true
  }
}

/**
 * 设置管理器类
 */
export class SettingsManager {
  private store: JsonDataStore<Partial<PluginSettings>>
  private _settings: PluginSettings
  private settingsPath: string
  private watcher: fs.FSWatcher | null = null
  private lastModified: number = 0
  private reloadTimeout: NodeJS.Timeout | null = null

  constructor(dataPath: string, private readonly logger?: SettingsLogger) {
    this.settingsPath = path.resolve(dataPath, 'settings.json')

    // 确保数据目录存在
    if (!fs.existsSync(dataPath)) {
      fs.mkdirSync(dataPath, { recursive: true })
    }

    this.store = new JsonDataStore(this.settingsPath, {}, { logger: this.logger })
    if (!fs.existsSync(this.settingsPath)) {
      fs.writeFileSync(this.settingsPath, '{}', 'utf8')
    }
    this._settings = this.loadSettings()

    // 启动文件监视器
    this.startWatcher()
  }

  /**
   * 启动文件监视器
   */
  private startWatcher(): void {
    try {
      // 记录初始修改时间
      if (fs.existsSync(this.settingsPath)) {
        this.lastModified = fs.statSync(this.settingsPath).mtimeMs
      }

      this.watcher = fs.watch(this.settingsPath, (eventType) => {
        if (eventType === 'change') {
          // 防抖：避免频繁重新加载
          if (this.reloadTimeout) {
            clearTimeout(this.reloadTimeout)
          }
          this.reloadTimeout = setTimeout(() => {
            this.checkAndReload()
          }, 100)
        }
      })
    } catch (e) {
      this.logger?.error('[SettingsManager] 启动文件监视器失败: %o', e)
    }
  }

  /**
   * 检查文件变化并重新加载
   */
  private checkAndReload(): void {
    try {
      if (!fs.existsSync(this.settingsPath)) return

      const stat = fs.statSync(this.settingsPath)
      // 只有当文件真正被修改时才重新加载
      if (stat.mtimeMs > this.lastModified) {
        this.lastModified = stat.mtimeMs
        // 重新加载 store 的数据
        this.store.reload()
        this._settings = this.loadSettings()
        this.logger?.info('[SettingsManager] 检测到配置文件变化，已重新加载')
      }
    } catch (e) {
      this.logger?.error('[SettingsManager] 重新加载配置失败: %o', e)
    }
  }

  /**
   * 加载设置，合并默认值
   */
  private loadSettings(): PluginSettings {
    const saved = this.store.getAll()
    return deepMerge(DEFAULT_SETTINGS, saved)
  }

  /**
   * 获取所有设置
   */
  get settings(): PluginSettings {
    return this._settings
  }

  /**
   * 获取特定设置项
   */
  get<K extends keyof PluginSettings>(key: K): PluginSettings[K] {
    return this._settings[key]
  }

  /**
   * 更新设置
   */
  async update(updates: Partial<PluginSettings>): Promise<void> {
    // 更新内存中的设置
    this._settings = deepMerge(this._settings, updates)

    // 保存到文件（只保存与默认值不同的部分）
    const toSave = getDiff(DEFAULT_SETTINGS, this._settings)

    // 获取当前 store 中的所有键
    const currentKeys = Object.keys(this.store.getAll())
    const newKeys = Object.keys(toSave)

    // 删除不再需要的键（值恢复为默认值的情况）
    for (const key of currentKeys) {
      if (!newKeys.includes(key)) {
        this.store.delete(key as keyof PluginSettings)
      }
    }

    // 设置新值
    for (const key of Object.keys(toSave) as Array<keyof PluginSettings>) {
      const value = toSave[key]
      if (value !== undefined) {
        this.store.set(key, value)
      }
    }

    await this.store.flush()
  }

  /**
   * 重置为默认设置
   */
  async reset(): Promise<void> {
    this._settings = { ...DEFAULT_SETTINGS }
    // 清空存储
    for (const key of Object.keys(this.store.getAll()) as Array<keyof PluginSettings>) {
      this.store.delete(key)
    }
    await this.store.flush()
  }

  /**
   * 释放资源
   */
  dispose(): void {
    // 关闭文件监视器
    if (this.watcher) {
      this.watcher.close()
      this.watcher = null
    }
    if (this.reloadTimeout) {
      clearTimeout(this.reloadTimeout)
      this.reloadTimeout = null
    }
    this.store.dispose()
  }
}
