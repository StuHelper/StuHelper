import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { createServer } from 'node:net'
import { setTimeout as sleep } from 'node:timers/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { load, dump } from 'js-yaml'

const STARTUP_TIMEOUT_MS = 10000
const SMOKE_PORT = await findAvailablePort()
const COREPACK_BIN = process.platform === 'win32' ? 'corepack.cmd' : 'corepack'
const cwd = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tempConfigDir = await createTempConfigDir()
const tempConfigPath = await writeSmokeConfig(tempConfigDir)

try {
  const startup = startupInvocation(tempConfigPath)
  const child = spawn(startup.command, startup.args, {
    cwd,
    env: {
      ...process.env,
      NODE_ENV: 'production',
      KOISHI_CONFIG_FILE: '',
      STUHELPER_CONSOLE_ADMIN_PASSWORD: 'startup-smoke-password',
      STUHELPER_PLATFORM_BASE_URL: process.env.STUHELPER_PLATFORM_BASE_URL ?? 'http://127.0.0.1:8080',
      STUHELPER_PLATFORM_SERVICE_TOKEN: process.env.STUHELPER_PLATFORM_SERVICE_TOKEN ?? 'startup-smoke-service-token',
      STUHELPER_ALERTMANAGER_WEBHOOK_ENABLED: 'false',
      STUHELPER_ALERTMANAGER_BOT_SELF_ID: '',
      ALERTMANAGER_WEBHOOK_TOKEN: '',
    },
    detached: process.platform !== 'win32',
    shell: startup.shell,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  let output = ''

  child.stdout.on('data', (chunk) => {
    output += chunk.toString()
  })

  child.stderr.on('data', (chunk) => {
    output += chunk.toString()
  })

  const exitPromise = waitForExit(child)

  await sleep(STARTUP_TIMEOUT_MS)
  stopProcessGroup(child.pid)

  const exitCode = await exitPromise

  assert.match(output, /loader apply plugin stuhelper-core:/, 'Koishi 启动时没有加载 stuhelper-core。')
  assert.match(output, /loader apply plugin stuhelper-binding:/, 'Koishi 启动时没有加载 stuhelper-binding。')
  assert.match(output, /loader apply plugin stuhelper-group-guard:/, 'Koishi 启动时没有加载 stuhelper-group-guard。')
  assert.match(output, /loader apply plugin stuhelper-admin:/, 'Koishi 启动时没有加载 stuhelper-admin。')
  assert.match(output, /StuHelper 群管中心服务已注册/, 'stuhelper-core 没有注册群管中心服务。')
  assert.match(output, /StuHelper 群管中心控制台入口已注册/, 'stuhelper-core 没有注册群管中心控制台入口。')
  assert.match(output, /stuhelper-core 旧群管运行时模块已停用，仅注册 WebUI 与 Console API/, 'stuhelper-core 没有明确停用旧群管运行时模块。')
  assert.match(output, /绑定插件已加载，命令字和提示文案由 WebUI runtime settings 控制/, 'stuhelper-binding 没有按 WebUI runtime settings 装配。')
  assert.match(output, /目标群由后端 admission policy 同步为 Koishi 执行态缓存/, 'stuhelper-group-guard 没有按后端 admission policy 同步边界装配。')
  assert.match(output, /管理员命令已注册，执行开关和提示文案由 StuHelper WebUI runtime settings 控制/, 'stuhelper-admin 没有按 WebUI runtime settings 装配。')
  assert.match(output, /WebSocket API registered/, 'stuhelper-core 没有完成 console API 注入与注册。')
  assert.doesNotMatch(output, /启动文件监视器失败/, 'stuhelper-core 仍然在首次启动时错误地监视不存在的 settings.json。')
  assert.match(output, new RegExp(`server listening at http://127\\.0\\.0\\.1:${SMOKE_PORT}`), 'Koishi 控制台没有在烟雾端口完成监听。')
  assert.doesNotMatch(output, /ERR_UNSUPPORTED_DIR_IMPORT/, 'Koishi 仍然存在目录导入错误。')
  assert.doesNotMatch(output, /ERR_MODULE_NOT_FOUND/, 'Koishi 仍然存在模块解析错误。')
  assert.ok(isExpectedShutdownExitCode(exitCode), `Koishi 启动探针异常退出：${exitCode}\n${output}`)

  process.stdout.write(output)
} finally {
  await rm(tempConfigDir, { recursive: true, force: true })
}

function startupInvocation(configPath) {
  if (process.platform === 'win32') {
    return {
      command: `${COREPACK_BIN} yarn exec koishi start "${configPath}"`,
      args: [],
      shell: true,
    }
  }
  return {
    command: COREPACK_BIN,
    args: ['yarn', 'exec', 'koishi', 'start', configPath],
    shell: false,
  }
}

function isExpectedShutdownExitCode(exitCode) {
  if (exitCode === 0 || exitCode === -15) {
    return true
  }
  return process.platform === 'win32' && exitCode === 1
}

function stopProcessGroup(pid) {
  if (process.platform === 'win32') {
    try {
      execFileSync('taskkill', ['/PID', String(pid), '/T', '/F'], { stdio: 'ignore' })
    } catch (error) {
      if (!isCommandNoMatchError(error) && !isWindowsTaskkillNoMatchError(error)) {
        throw error
      }
    }
    return
  }

  try {
    process.kill(-pid, 'SIGTERM')
  } catch (error) {
    if (!isMissingProcessError(error)) {
      throw error
    }
  }
}

function waitForExit(childProcess) {
  return new Promise((resolveExit, rejectExit) => {
    childProcess.once('error', rejectExit)
    childProcess.once('close', (code, signal) => {
      if (typeof code === 'number') {
        resolveExit(code)
        return
      }
      resolveExit(signal === 'SIGTERM' ? -15 : -1)
    })
  })
}

function isMissingProcessError(error) {
  return error instanceof Error && 'code' in error && error.code === 'ESRCH'
}

function isWindowsTaskkillNoMatchError(error) {
  return typeof error === 'object'
    && error !== null
    && 'status' in error
    && error.status === 128
}

function isCommandNoMatchError(error) {
  return typeof error === 'object'
    && error !== null
    && 'status' in error
    && error.status === 1
}

function findAvailablePort() {
  return new Promise((resolvePort, rejectPort) => {
    const server = createServer()
    server.unref()
    server.once('error', rejectPort)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (typeof address !== 'object' || address === null) {
        server.close()
        rejectPort(new Error('无法为 Koishi 启动探针分配本地端口。'))
        return
      }
      server.close((error) => {
        if (error) {
          rejectPort(error)
          return
        }
        resolvePort(address.port)
      })
    })
  })
}

async function createTempConfigDir() {
  const tempRoot = join(cwd, '.tmp')
  await mkdir(tempRoot, { recursive: true })
  return mkdtemp(join(tempRoot, 'startup-smoke-'))
}

async function writeSmokeConfig(tempDir) {
  const sourcePath = join(cwd, 'koishi.yml')
  const source = await readFile(sourcePath, 'utf8')
  const config = load(source)
  const plugins = config.plugins
  plugins['group:server']['server:chm356'].port = SMOKE_PORT
  plugins['group:server']['server:chm356'].maxPort = SMOKE_PORT
  const targetPath = join(tempDir, 'koishi.yml')
  await writeFile(targetPath, dump(config))
  return targetPath
}
