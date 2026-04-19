import { Context } from 'koishi'

interface DisposableFork {
  dispose(): boolean
}

// 为 Koishi 插件测试统一管理 fork，避免 root.stop() 命中 plugin-mock 的清理缺陷。
export function createKoishiTestRuntime() {
  const root = new Context()
  const forks: DisposableFork[] = []

  function register(plugin: Parameters<Context['plugin']>[0], config?: Parameters<Context['plugin']>[1]) {
    const fork = root.plugin(plugin as never, config as never) as DisposableFork
    forks.unshift(fork)
    return fork
  }

  function dispose() {
    for (const fork of forks) {
      fork.dispose()
    }
  }

  return {
    root,
    register,
    dispose,
  }
}
