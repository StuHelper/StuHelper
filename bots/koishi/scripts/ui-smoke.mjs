import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { setTimeout as sleep } from 'node:timers/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { load, dump } from 'js-yaml'

/**
 * P0a UI smoke 启动器。复用 startup-smoke.mjs 的端口释放 + 临时配置 + spawn 模式；
 * 等 koishi 输出 "server listening" 后启动 Playwright；spec 跑完后 SIGTERM 关闭进程组。
 */

const SMOKE_PORT = Number.parseInt(process.env.STUHELPER_UI_SMOKE_PORT ?? '5140', 10)
const STARTUP_LISTEN_TIMEOUT_MS = 30_000
const cwd = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tempConfigDir = await createTempConfigDir()
const tempConfigPath = await writeSmokeConfig(tempConfigDir)

let koishiChild
let playwrightExitCode = 1

try {
  await releasePort(SMOKE_PORT)

  koishiChild = spawn('corepack', ['yarn', 'exec', 'koishi', 'start', tempConfigPath], {
    cwd,
    env: {
      ...process.env,
      NODE_ENV: 'production',
      KOISHI_CONFIG_FILE: '',
      STUHELPER_CONSOLE_ADMIN_PASSWORD: process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD ?? 'ui-smoke-password',
    },
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  let listening = false

  koishiChild.stdout.on('data', (chunk) => {
    const text = chunk.toString()
    process.stdout.write(text)
    if (text.includes(`server listening at http://127.0.0.1:${SMOKE_PORT}`)) {
      listening = true
    }
  })

  koishiChild.stderr.on('data', (chunk) => {
    process.stderr.write(chunk.toString())
  })

  await waitFor(() => listening, STARTUP_LISTEN_TIMEOUT_MS, 'koishi 控制台未在超时内监听 5140')

  playwrightExitCode = await runPlaywright()
} finally {
  if (koishiChild?.pid) {
    stopProcessGroup(koishiChild.pid)
    await waitForExit(koishiChild)
  }
  await rm(tempConfigDir, { recursive: true, force: true })
}

if (playwrightExitCode !== 0) {
  process.exit(playwrightExitCode)
}

function runPlaywright() {
  return new Promise((resolveExit, rejectExit) => {
    const child = spawn('corepack', ['yarn', 'exec', 'playwright', 'test'], {
      cwd,
      env: {
        ...process.env,
        STUHELPER_UI_SMOKE_PORT: String(SMOKE_PORT),
      },
      stdio: 'inherit',
    })

    child.once('error', rejectExit)
    child.once('close', (code, signal) => {
      if (typeof code === 'number') {
        resolveExit(code)
        return
      }
      resolveExit(signal === 'SIGTERM' ? -15 : -1)
    })
  })
}

async function waitFor(predicate, timeoutMs, message) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (predicate()) return
    await sleep(200)
  }
  throw new Error(message)
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
  return new Promise((resolveExit) => {
    childProcess.once('close', () => resolveExit())
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
  if (!pids.length) return

  const deadline = Date.now() + 5000
  while (Date.now() < deadline) {
    if (!listListeningPIDs(port).length) return
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
    if (!output) return []
    return output
      .split(/\s+/)
      .filter(Boolean)
      .map((item) => Number(item))
      .filter((item) => Number.isInteger(item) && item > 0)
  } catch (error) {
    if (isCommandNoMatchError(error)) return []
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
  return mkdtemp(join(tempRoot, 'ui-smoke-'))
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
