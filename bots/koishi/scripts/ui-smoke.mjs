import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { createServer } from 'node:http'
import { setTimeout as sleep } from 'node:timers/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { load, dump } from 'js-yaml'

/**
 * P0a UI smoke 启动器。复用 startup-smoke.mjs 的端口释放 + 临时配置 + spawn 模式；
 * 等 koishi 输出 "server listening" 后启动 Playwright；spec 跑完后 SIGTERM 关闭进程组。
 *
 * 与 startup-smoke 的差异：
 * - 此处用临时 SQLite 路径，避免污染 bots/koishi/data/koishi.db（P0b 登录 fixture 需要干净 DB）
 * - 全程跟踪 koishi 子进程的 close 状态，子进程提前退出时立即抛错而非等监听超时
 * - 启动 Koishi 前先构建 workspace，保证 ignored 的 production dist 与源码同步
 */

const SMOKE_PORT = Number.parseInt(process.env.STUHELPER_UI_SMOKE_PORT ?? '5140', 10)
const STARTUP_LISTEN_TIMEOUT_MS = 30_000
const SHUTDOWN_TIMEOUT_MS = 5_000
const COREPACK_BIN = process.platform === 'win32' ? 'corepack.cmd' : 'corepack'
const cwd = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tempConfigDir = await createTempConfigDir()
const tempConfigPath = await writeSmokeConfig(tempConfigDir)
const platformStub = process.env.STUHELPER_PLATFORM_BASE_URL
  ? null
  : await startPlatformStub()

let koishiChild
let koishiClosed = false
let koishiExitCode = null
let koishiExitSignal = null
let playwrightExitCode = 1

try {
  await releasePort(SMOKE_PORT)
  buildWorkspace()

  const koishiStartup = corepackSpawnInvocation(['yarn', 'exec', 'koishi', 'start', tempConfigPath])
  koishiChild = spawn(koishiStartup.command, koishiStartup.args, {
    cwd,
    env: {
      ...process.env,
      NODE_ENV: 'production',
      KOISHI_CONFIG_FILE: '',
      STUHELPER_GROUP_CENTER_DATA_DIR: join(tempConfigDir, 'stuhelper-data'),
      STUHELPER_CONSOLE_ADMIN_PASSWORD: process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD ?? 'ui-smoke-password',
      STUHELPER_PLATFORM_BASE_URL: process.env.STUHELPER_PLATFORM_BASE_URL ?? platformStub.baseUrl,
      STUHELPER_PLATFORM_SERVICE_TOKEN: process.env.STUHELPER_PLATFORM_SERVICE_TOKEN ?? 'ui-smoke-service-token',
    },
    detached: process.platform !== 'win32',
    shell: koishiStartup.shell,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  // 立即注册 close 监听，避免子进程在 finally 阶段之前就退出导致后续 once('close') 永久挂起
  koishiChild.once('close', (code, signal) => {
    koishiClosed = true
    koishiExitCode = code
    koishiExitSignal = signal
  })

  let listening = false
  let cacheWarmed = false

  koishiChild.stdout.on('data', (chunk) => {
    const text = chunk.toString()
    process.stdout.write(text)
    if (text.includes(`server listening at http://127.0.0.1:${SMOKE_PORT}`)) {
      listening = true
    }
    // 等 cache 预热完成才让 spec 开跑，避免 dashboard 首次访问冷启动竞态
    if (text.includes('缓存预热完成')) {
      cacheWarmed = true
    }
  })

  koishiChild.stderr.on('data', (chunk) => {
    process.stderr.write(chunk.toString())
  })

  await waitFor(
    () => {
      if (koishiClosed) {
        throw new Error(
          `koishi 进程在监听就绪前提前退出（exitCode=${koishiExitCode}, signal=${koishiExitSignal}）`,
        )
      }
      return listening && cacheWarmed
    },
    STARTUP_LISTEN_TIMEOUT_MS,
    'koishi 控制台未在超时内完成 listening + cache 预热',
  )

  playwrightExitCode = await runPlaywright()
} finally {
  if (koishiChild?.pid) {
    stopProcessGroup(koishiChild.pid)
    await waitForExit()
  }
  await rm(tempConfigDir, { recursive: true, force: true })
  await platformStub?.close()
}

if (playwrightExitCode !== 0) {
  process.exit(playwrightExitCode)
}

function buildWorkspace() {
  const build = corepackExecInvocation(['yarn', 'build'])
  execFileSync(build.command, build.args, {
    cwd,
    shell: build.shell,
    stdio: 'inherit',
  })
}

function runPlaywright() {
  return new Promise((resolveExit, rejectExit) => {
    const playwright = corepackSpawnInvocation(['yarn', 'exec', 'playwright', 'test'])
    const child = spawn(playwright.command, playwright.args, {
      cwd,
      env: {
        ...process.env,
        STUHELPER_UI_SMOKE_PORT: String(SMOKE_PORT),
      },
      shell: playwright.shell,
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

function corepackExecInvocation(args) {
  if (process.platform === 'win32') {
    return {
      command: 'cmd.exe',
      args: ['/d', '/s', '/c', `corepack ${args.map(quoteWindowsShellArg).join(' ')}`],
      shell: false,
    }
  }
  return { command: COREPACK_BIN, args, shell: false }
}

function corepackSpawnInvocation(args) {
  if (process.platform === 'win32') {
    return {
      command: `${COREPACK_BIN} ${args.map(quoteWindowsShellArg).join(' ')}`,
      args: [],
      shell: true,
    }
  }
  return { command: COREPACK_BIN, args, shell: false }
}

function quoteWindowsShellArg(value) {
  const text = String(value)
  if (/^[\w:./\\-]+$/.test(text)) {
    return text
  }
  return `"${text.replaceAll('"', '""')}"`
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

function killProcessGroup(pid) {
  if (process.platform === 'win32') {
    stopProcessGroup(pid)
    return
  }

  try {
    process.kill(-pid, 'SIGKILL')
  } catch (error) {
    if (!isMissingProcessError(error)) {
      throw error
    }
  }
}

async function waitForExit() {
  if (koishiClosed) return
  const deadline = Date.now() + SHUTDOWN_TIMEOUT_MS
  while (Date.now() < deadline) {
    if (koishiClosed) return
    await sleep(100)
  }
  // SIGTERM 没让 koishi 在超时内退出，升级到 SIGKILL 兜底，避免本脚本永久挂起
  if (koishiChild?.pid) {
    killProcessGroup(koishiChild.pid)
  }
  const killDeadline = Date.now() + 1_000
  while (Date.now() < killDeadline) {
    if (koishiClosed) return
    await sleep(50)
  }
  process.stderr.write('[ui-smoke] WARN: koishi child did not exit after SIGKILL\n')
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
    if (isMissingCommandError(error) && process.platform === 'win32') {
      return listWindowsListeningPIDs(port)
    }
    throw error
  }
}

function listWindowsListeningPIDs(port) {
  const output = execFileSync('netstat', ['-ano', '-p', 'TCP'], { encoding: 'utf8' })
  return output
    .split(/\r?\n/)
    .map((line) => line.trim().split(/\s+/))
    .filter((parts) => parts.length >= 5 && parts[0] === 'TCP' && parts[3] === 'LISTENING')
    .filter((parts) => parts[1].endsWith(`:${port}`) || parts[1].endsWith(`]:${port}`))
    .map((parts) => Number(parts[4]))
    .filter((item) => Number.isInteger(item) && item > 0)
}

function isMissingCommandError(error) {
  return typeof error === 'object'
    && error !== null
    && 'code' in error
    && error.code === 'ENOENT'
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
  // 用临时目录隔离 SQLite，避免 admin 账户/会话残留污染下次 smoke 与开发数据库
  plugins['group:storage']['database-sqlite:q4tbt0'].path = join(tempDir, 'koishi.db')
  const targetPath = join(tempDir, 'koishi.yml')
  await writeFile(targetPath, dump(config))
  return targetPath
}

function startPlatformStub() {
  const blacklistEntries = []
  let blacklistSeq = 0

  const server = createServer(async (request, response) => {
    const url = new URL(request.url ?? '/', 'http://127.0.0.1')
    const method = request.method ?? 'GET'

    if (method === 'GET' && url.pathname === '/health/live') {
      response.writeHead(204)
      response.end()
      return
    }

    if (method === 'GET' && url.pathname === '/api/v1/bot/member-blacklist') {
      const result = filterMemberBlacklistEntries(blacklistEntries, url.searchParams)
      writeJSON(response, result)
      return
    }

    if (method === 'GET' && url.pathname === '/api/v1/bot/member-blacklist/access') {
      writeJSON(response, { canJoin: true, decision: 'allowed' })
      return
    }

    if (method === 'POST' && url.pathname === '/api/v1/bot/member-blacklist') {
      const body = await readJSONBody(request)
      const entry = memberBlacklistEntry(`stub-blacklist-entry-${++blacklistSeq}`, body)
      blacklistEntries.push(entry)
      writeJSON(response, entry)
      return
    }

    if (method === 'POST' && url.pathname === '/api/v1/bot/member-blacklist/release-by-subject') {
      const body = await readJSONBody(request)
      const entry = findMemberBlacklistEntryBySubject(blacklistEntries, body)
        ?? memberBlacklistEntry(`stub-blacklist-entry-${++blacklistSeq}`, body)
      releaseMemberBlacklistEntry(entry, body)
      if (!blacklistEntries.some((item) => item.id === entry.id)) {
        blacklistEntries.push(entry)
      }
      writeJSON(response, entry)
      return
    }

    if (method === 'POST' && /^\/api\/v1\/bot\/member-blacklist\/[^/]+\/release$/.test(url.pathname)) {
      const id = decodeURIComponent(url.pathname.split('/').at(-2) ?? '')
      const body = await readJSONBody(request)
      const entry = blacklistEntries.find((item) => item.id === id) ?? memberBlacklistEntry(id || 'stub-blacklist-entry', {})
      releaseMemberBlacklistEntry(entry, body)
      writeJSON(response, entry)
      return
    }

    const qqVerificationMatch = url.pathname.match(/^\/api\/v1\/bot\/qq-users\/([^/]+)\/verification$/)
    if (method === 'GET' && qqVerificationMatch) {
      writeJSON(response, {
        qqID: decodeURIComponent(qqVerificationMatch[1]),
        bindingStatus: 'unbound',
        profileVerificationStatus: 'unverified',
        studentVerificationStatus: 'unverified',
        canJoin: false,
      })
      return
    }

    if (method === 'GET' && url.pathname === '/api/v1/bot/admission/sessions/pending') {
      writeJSON(response, [])
      return
    }

    if (method === 'GET' && url.pathname === '/api/v1/bot/admission/freshman/applications/pending-forward') {
      writeJSON(response, [])
      return
    }

    if (method === 'POST' && /^\/api\/v1\/bot\/admission\/sessions\/[^/]+\/events$/.test(url.pathname)) {
      writeJSON(response, { message: 'ok' })
      return
    }

    response.writeHead(404, { 'content-type': 'application/json' })
    response.end(JSON.stringify({ success: false, error: { code: 'not_found', message: `stub route not found: ${method} ${url.pathname}` } }))
  })

  return new Promise((resolveStub, rejectStub) => {
    server.once('error', rejectStub)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      assert.equal(typeof address, 'object')
      assert.notEqual(address, null)
      resolveStub({
        baseUrl: `http://127.0.0.1:${address.port}`,
        close: () => new Promise((resolveClose, rejectClose) => {
          server.close((error) => error ? rejectClose(error) : resolveClose())
        }),
      })
    })
  })
}

function writeJSON(response, data) {
  response.writeHead(200, { 'content-type': 'application/json' })
  response.end(JSON.stringify({ success: true, data }))
}

function readJSONBody(request) {
  return new Promise((resolveBody, rejectBody) => {
    const chunks = []
    request.on('data', (chunk) => chunks.push(chunk))
    request.once('error', rejectBody)
    request.once('end', () => {
      if (chunks.length === 0) {
        resolveBody({})
        return
      }
      try {
        resolveBody(JSON.parse(Buffer.concat(chunks).toString('utf8')))
      } catch (error) {
        rejectBody(error)
      }
    })
  })
}

function memberBlacklistEntry(id, input) {
  const now = new Date(0).toISOString()
  return {
    id,
    platform: input.platform ?? 'qq',
    subjectType: input.subjectType ?? 'qq_user',
    subjectID: input.subjectID ?? '100000',
    scopeType: input.scopeType ?? 'global',
    guildID: input.guildID ?? null,
    source: input.source ?? 'manual_admin',
    reasonCode: input.reasonCode ?? 'manual_blacklist',
    reasonText: input.reasonText ?? '',
    metadata: input.metadata ?? {},
    createdByType: 'qq_operator',
    createdByID: input.metadata?.operatorQQID ?? 'ui-smoke',
    createdFrom: input.createdFrom ?? 'ui_smoke',
    createdAt: now,
    updatedAt: now,
    expiresAt: input.expiresAt ?? null,
    releasedAt: input.releasedAt ?? null,
    releasedByType: input.releasedByType ?? null,
    releasedByID: input.releasedByID ?? null,
    releaseReasonCode: input.releaseReasonCode ?? null,
    releaseReason: input.releaseReason ?? null,
  }
}

function filterMemberBlacklistEntries(entries, searchParams) {
  const page = Math.max(Number.parseInt(searchParams.get('page') ?? '1', 10) || 1, 1)
  const pageSize = Math.max(Number.parseInt(searchParams.get('pageSize') ?? '100', 10) || 100, 1)
  const filtered = entries.filter((entry) => {
    if (searchParams.has('platform') && entry.platform !== searchParams.get('platform')) return false
    if (searchParams.has('subjectType') && entry.subjectType !== searchParams.get('subjectType')) return false
    if (searchParams.has('scopeType') && entry.scopeType !== searchParams.get('scopeType')) return false
    if (searchParams.has('guildID') && entry.guildID !== searchParams.get('guildID')) return false
    if (searchParams.get('status') === 'active') {
      return !entry.releasedAt && (!entry.expiresAt || Date.parse(entry.expiresAt) > Date.now())
    }
    return true
  })
  const offset = (page - 1) * pageSize
  return {
    list: filtered.slice(offset, offset + pageSize),
    total: filtered.length,
  }
}

function findMemberBlacklistEntryBySubject(entries, input) {
  return entries.find((entry) =>
    entry.platform === (input.platform ?? 'qq')
    && entry.subjectType === (input.subjectType ?? 'qq_user')
    && entry.subjectID === input.subjectID
    && entry.scopeType === (input.scopeType ?? 'global')
    && (entry.guildID ?? null) === (input.guildID ?? null)
    && !entry.releasedAt)
}

function releaseMemberBlacklistEntry(entry, input) {
  const now = new Date(0).toISOString()
  entry.releasedAt = input.releasedAt ?? now
  entry.releasedByType = input.releasedByType ?? 'qq_operator'
  entry.releasedByID = input.releasedByID ?? 'ui-smoke'
  entry.releaseReasonCode = input.releaseReasonCode ?? 'release_only'
  entry.releaseReason = input.releaseReason ?? ''
  entry.updatedAt = now
}
