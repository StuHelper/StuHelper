/**
 * 通用 JSON 数据存储类
 * 提供延迟保存、原子写入、备份等功能
 */
import * as fs from 'fs'
import * as path from 'path'

export interface JsonStoreOptions {
  /** 延迟保存时间（毫秒），默认 1000ms */
  saveDelay?: number
  /** 是否创建备份，默认 true */
  createBackup?: boolean
  /** 最大备份数量，默认 3 */
  maxBackups?: number
  /** 日志入口；运行时应传入 Koishi Logger */
  logger?: JsonDataStoreLogger
}

export interface JsonDataStoreLogger {
  info(message: string, ...args: unknown[]): void
  error(message: string, ...args: unknown[]): void
}

interface ResolvedJsonStoreOptions extends Required<Omit<JsonStoreOptions, 'logger'>> {
  logger: JsonDataStoreLogger | null
}

export class JsonDataStore<T extends Record<string, unknown> = Record<string, unknown>> {
  private data: T
  private saveTimer: NodeJS.Timeout | null = null
  private dirty = false
  private readonly options: ResolvedJsonStoreOptions

  constructor(
    private readonly filePath: string,
    private readonly defaultValue: T,
    options: JsonStoreOptions = {}
  ) {
    this.options = {
      saveDelay: options.saveDelay ?? 1000,
      createBackup: options.createBackup ?? true,
      maxBackups: options.maxBackups ?? 3,
      logger: options.logger ?? null,
    }
    this.data = this.load()
  }

  /**
   * 加载数据
   */
  private load(): T {
    const dir = path.dirname(this.filePath)
    try {
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true })
      }
    } catch (error) {
      this.options.logger?.error('[JsonDataStore] 加载数据失败: %s', this.filePath, error)
      throw error
    }

    if (!fs.existsSync(this.filePath)) {
      return this.cloneDefaultValue()
    }

    const content = fs.readFileSync(this.filePath, 'utf-8')
    try {
      return JSON.parse(content) as T
    } catch (parseError) {
      this.logJsonParseError(parseError as SyntaxError, content)
      throw parseError
    }
  }

  /**
   * 获取所有数据
   */
  getAll(): T {
    return this.data
  }

  /**
   * 重新从文件加载数据
   */
  reload(): void {
    this.data = this.load()
    this.options.logger?.info('[JsonDataStore] 重新加载: %s, 数据条目: %d', this.filePath, Object.keys(this.data).length)
  }

  /**
   * 设置所有数据
   */
  setAll(data: T): void {
    this.data = data
    this.markDirty()
  }

  /**
   * 获取指定键的数据（适用于对象类型）
   */
  get<K extends keyof T>(key: K): T[K] | undefined {
    return this.data[key]
  }

  /**
   * 设置指定键的数据（适用于对象类型）
   */
  set<K extends keyof T>(key: K, value: T[K]): void {
    this.data[key] = value
    this.markDirty()
  }

  /**
   * 删除指定键的数据（适用于对象类型）
   */
  delete<K extends keyof T>(key: K): boolean {
    if (key in this.data) {
      delete this.data[key]
      this.markDirty()
      return true
    }
    return false
  }

  /**
   * 检查键是否存在
   */
  has<K extends keyof T>(key: K): boolean {
    return key in this.data
  }

  /**
   * 更新数据（合并）
   */
  update(partial: Partial<T>): void {
    this.data = { ...this.data, ...partial }
    this.markDirty()
  }

  /**
   * 标记数据已修改，启动延迟保存
   */
  private markDirty(): void {
    this.dirty = true

    if (this.saveTimer) {
      clearTimeout(this.saveTimer)
    }

    this.saveTimer = setTimeout(() => {
      this.flush()
    }, this.options.saveDelay)
  }

  /**
   * 立即保存数据到文件
   */
  flush(): void {
    if (!this.dirty) return

    if (this.options.createBackup && fs.existsSync(this.filePath)) {
      this.createBackup()
    }

    const tempPath = `${this.filePath}.tmp`
    const content = JSON.stringify(this.data, null, 2)
    try {
      fs.writeFileSync(tempPath, content, 'utf-8')
      fs.renameSync(tempPath, this.filePath)
    } catch (error) {
      this.options.logger?.error('[JsonDataStore] 保存数据失败: %s', this.filePath, error)
      throw error
    }

    this.dirty = false
    if (this.saveTimer) {
      clearTimeout(this.saveTimer)
      this.saveTimer = null
    }
  }

  /**
   * 创建备份
   */
  private createBackup(): void {
    const dir = path.dirname(this.filePath)
    const basename = path.basename(this.filePath, '.json')
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-')
    const backupPath = path.join(dir, `${basename}.backup.${timestamp}.json`)

    fs.copyFileSync(this.filePath, backupPath)
    this.cleanOldBackups(dir, basename)
  }

  /**
   * 清理旧备份
   */
  private cleanOldBackups(dir: string, basename: string): void {
    const pattern = new RegExp(`^${escapeRegExpLiteral(basename)}\\.backup\\..+\\.json$`)
    const backups = fs.readdirSync(dir)
      .filter(file => pattern.test(file))
      .map(file => ({
        name: file,
        path: path.join(dir, file),
        time: fs.statSync(path.join(dir, file)).mtime.getTime()
      }))
      .sort((a, b) => b.time - a.time)

    for (let i = this.options.maxBackups; i < backups.length; i++) {
      fs.unlinkSync(backups[i].path)
    }
  }

  private cloneDefaultValue(): T {
    return JSON.parse(JSON.stringify(this.defaultValue))
  }

  private logJsonParseError(error: SyntaxError, content: string): void {
    this.options.logger?.error('[JsonDataStore] JSON 解析失败: %s', this.filePath)
    this.options.logger?.error('[JsonDataStore] 错误信息: %s', error.message)
    const match = error.message.match(/position (\d+)/)
    if (match) {
      const pos = parseInt(match[1])
      const before = content.substring(Math.max(0, pos - 50), pos)
      const after = content.substring(pos, pos + 50)
      this.options.logger?.error('[JsonDataStore] 错误位置附近: ...%s【错误在此】%s...', before, after)
    }
    if (content.includes(',]') || content.includes(',}')) {
      this.options.logger?.error('[JsonDataStore] 提示: 可能存在尾随逗号，JSON 不允许在数组/对象最后一个元素后加逗号')
    }
  }

  /**
   * 释放资源
   */
  dispose(): void {
    this.flush()
    if (this.saveTimer) {
      clearTimeout(this.saveTimer)
      this.saveTimer = null
    }
  }
}

function escapeRegExpLiteral(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
