/**
 * 按 key 串行化异步任务的进程内互斥锁。
 *
 * minato 会把同一 tick 内对同一张表的并发 set 合并成一次更新（实测连
 * `$.add` 原子表达式也会被合并丢失），sqlite driver 的并发 transact 则直接
 * 抛 "cannot start a transaction within a transaction"。因此读-改-写计数器
 * 这类操作必须在进程内按 key 串行执行。
 */
export class KeyedMutex {
  private readonly tails = new Map<string, Promise<void>>()

  async run<T>(key: string, task: () => Promise<T>): Promise<T> {
    const previous = this.tails.get(key) ?? Promise.resolve()
    const result = previous.then(task)
    const tail = result.then(() => undefined, () => undefined)
    this.tails.set(key, tail)
    void tail.then(() => {
      if (this.tails.get(key) === tail) {
        this.tails.delete(key)
      }
    })
    return result
  }
}
