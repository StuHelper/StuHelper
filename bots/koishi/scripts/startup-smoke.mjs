import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { setTimeout as sleep } from 'node:timers/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { load, dump } from 'js-yaml'

const STARTUP_TIMEOUT_MS = 10000
const SMOKE_PORT = 5140
const cwd = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tempConfigDir = await createTempConfigDir()
const tempConfigPath = await writeSmokeConfig(tempConfigDir)

try {
  await releasePort(SMOKE_PORT)

  const child = spawn('corepack', ['yarn', 'exec', 'koishi', 'start', tempConfigPath], {
    cwd,
    env: {
      ...process.env,
      NODE_ENV: 'production',
      KOISHI_CONFIG_FILE: '',
      STUHELPER_CONSOLE_ADMIN_PASSWORD: 'startup-smoke-password',
    },
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  let output = ''

  child.stdout.on('data', (chunk) => {
    output += chunk.toString()
  })

  child.stderr.on('data', (chunk) => {
    output += chunk.toString()
  })

  await sleep(STARTUP_TIMEOUT_MS)
  stopProcessGroup(child.pid)

  const exitCode = await waitForExit(child)

  assert.match(output, /loader apply plugin stuhelper-core:/, 'Koishi 启动时没有加载 stuhelper-core。')
  assert.match(output, /StuHelper 群管中心插件已加载/, 'stuhelper-core 没有完成新的群管中心装配。')
  assert.match(output, /WebSocket API registered/, 'stuhelper-core 没有完成 console API 注入与注册。')
  assert.doesNotMatch(output, /启动文件监视器失败/, 'stuhelper-core 仍然在首次启动时错误地监视不存在的 settings.json。')
  assert.match(output, new RegExp(`server listening at http://127\\.0\\.0\\.1:${SMOKE_PORT}`), 'Koishi 控制台没有在烟雾端口完成监听。')
  assert.doesNotMatch(output, /ERR_UNSUPPORTED_DIR_IMPORT/, 'Koishi 仍然存在目录导入错误。')
  assert.doesNotMatch(output, /ERR_MODULE_NOT_FOUND/, 'Koishi 仍然存在模块解析错误。')
  assert.ok(exitCode === 0 || exitCode === -15, `Koishi 启动探针异常退出：${exitCode}\n${output}`)

  process.stdout.write(output)
} finally {
  await rm(tempConfigDir, { recursive: true, force: true })
}

function stopProcessGroup(pid) {
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

async function releasePort(port) {
  const pids = listListeningPIDs(port)
  for (const pid of pids) {
    process.kill(pid, 'SIGTERM')
  }
  if (!pids.length) {
    return
  }

  const deadline = Date.now() + 5000
  while (Date.now() < deadline) {
    if (!listListeningPIDs(port).length) {
      return
    }
    await sleep(100)
  }

  for (const pid of listListeningPIDs(port)) {
    process.kill(pid, 'SIGKILL')
  }

  const remaining = listListeningPIDs(port)
  assert.equal(remaining.length, 0, `端口 ${port} 仍被占用：${remaining.join(', ')}`)
}

function listListeningPIDs(port) {
  try {
    const output = execFileSync('lsof', [`-tiTCP:${port}`, '-sTCP:LISTEN'], {
      encoding: 'utf8',
    }).trim()
    if (!output) {
      return []
    }
    return output
      .split(/\s+/)
      .filter(Boolean)
      .map((item) => Number(item))
      .filter((item) => Number.isInteger(item) && item > 0)
  } catch (error) {
    if (isCommandNoMatchError(error)) {
      return []
    }
    throw error
  }
}

function isCommandNoMatchError(error) {
  return typeof error === 'object'
    && error !== null
    && 'status' in error
    && error.status === 1
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
