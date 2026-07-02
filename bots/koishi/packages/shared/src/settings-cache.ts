/**
 * 进程内 settings 单例记录读缓存。
 *
 * 同一张 settings 表会被多个 store 实例消费（插件运行时实例 + WebUI console
 * API 实例各自构造），因此缓存不属于单个实例：通过 settingsReadCacheFor 按
 * (database 实例, 表名) 共享，任何实例 save/reset 都会回填同一份缓存，其他
 * 实例立即可见（F049）。适用前提是 settings 只由当前进程经 store 写入
 * （Koishi 单进程）。
 *
 * 缓存的是 Promise：并发的首次读取共享同一次数据库加载；加载失败时清空缓存，
 * 下一次读取重新加载。缓存值会被深冻结，防止调用方原地修改污染后续读取。
 */
export class SettingsReadCache<T extends object> {
  private pending: Promise<T> | undefined

  read(load: () => Promise<T>): Promise<T> {
    if (!this.pending) {
      this.pending = load().then(deepFreeze, (error) => {
        this.pending = undefined
        throw error
      })
    }
    return this.pending
  }

  write(next: T): T {
    const frozen = deepFreeze(next)
    this.pending = Promise.resolve(frozen)
    return frozen
  }

  invalidate(): void {
    this.pending = undefined
  }
}

const cachesByDatabase = new WeakMap<object, Map<string, SettingsReadCache<object>>>()

/**
 * 按 (database 实例, 表名) 取共享缓存。以 database 身份为键隔离不同 Koishi
 * 应用实例（测试里每个 Context 持有独立 sqlite），避免跨库串数据。
 */
export function settingsReadCacheFor<T extends object>(
  database: object,
  table: string,
): SettingsReadCache<T> {
  const identity = resolveDatabaseIdentity(database)
  let tableCaches = cachesByDatabase.get(identity)
  if (!tableCaches) {
    tableCaches = new Map()
    cachesByDatabase.set(identity, tableCaches)
  }
  let cache = tableCaches.get(table)
  if (!cache) {
    cache = new SettingsReadCache<object>()
    tableCaches.set(table, cache)
  }
  return cache as SettingsReadCache<T>
}

/**
 * cordis 服务代理在每次 `ctx.database` 访问时返回新的 Proxy，对象身份不稳定；
 * minato Database 实例上的 `tables` 是不被代理重绑的普通属性，跨 Proxy 指向
 * 同一对象，用它作为应用级身份锚点。测试 fake 没有 tables 时退回 fake 对象
 * 本身（每个 fake 即一个独立应用）。
 */
function resolveDatabaseIdentity(database: object): object {
  const tables = (database as { tables?: unknown }).tables
  return typeof tables === 'object' && tables !== null ? tables : database
}

function deepFreeze<T>(value: T): T {
  if (typeof value !== 'object' || value === null || Object.isFrozen(value)) {
    return value
  }
  for (const key of Object.keys(value)) {
    deepFreeze((value as Record<string, unknown>)[key])
  }
  return Object.freeze(value)
}
