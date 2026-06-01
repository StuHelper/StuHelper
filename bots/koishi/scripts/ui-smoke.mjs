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
const SEED_READY_MESSAGE = '[ui-smoke-seed] moderation report ready'
const cwd = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const tempConfigDir = await createTempConfigDir()
const smokeDataDir = join(tempConfigDir, 'stuhelper-data')
await writeSmokeSeedPlugin(tempConfigDir)
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
  await seedSmokeData(smokeDataDir)
  buildWorkspace()

  const koishiStartup = corepackSpawnInvocation(['yarn', 'exec', 'koishi', 'start', tempConfigPath])
  koishiChild = spawn(koishiStartup.command, koishiStartup.args, {
    cwd,
    env: {
      ...process.env,
      NODE_ENV: 'production',
      KOISHI_CONFIG_FILE: '',
      STUHELPER_GROUP_CENTER_DATA_DIR: smokeDataDir,
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
  let smokeSeeded = false

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
    if (text.includes(SEED_READY_MESSAGE)) {
      smokeSeeded = true
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
      return listening && cacheWarmed && smokeSeeded
    },
    STARTUP_LISTEN_TIMEOUT_MS,
    'koishi 控制台未在超时内完成 listening + cache 预热 + smoke seed',
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
    const env = {
      ...process.env,
      STUHELPER_UI_SMOKE_PORT: String(SMOKE_PORT),
    }
    // Playwright forces color for workers; inheriting NO_COLOR makes Node warn before tests start.
    delete env.NO_COLOR

    const child = spawn(playwright.command, playwright.args, {
      cwd,
      env,
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
  plugins['group:ui-smoke'] = {
    'mock:ui-smoke': { selfId: '514' },
    './ui-smoke-seed.cjs:seed': {},
  }
  const targetPath = join(tempDir, 'koishi.yml')
  await writeFile(targetPath, dump(config))
  return targetPath
}

async function writeSmokeSeedPlugin(tempDir) {
  await writeFile(join(tempDir, 'ui-smoke-seed.cjs'), `
const { h, Universal } = require('koishi')
const { GUARD_MEMBER_TABLE } = require('@stuhelper/koishi-shared')
const {
  MODERATION_EVENT_TABLE,
  MODERATION_REPORT_TABLE,
} = require('@stuhelper/koishi-moderation-core')

const REPORT_ID = 'ui-smoke-report-1'
const EVENT_ID = 'ui-smoke-report-event-1'
const CREATED_AT = new Date('2026-05-25T00:10:00.000Z')
const CHAT_GUILD_ID = '1001'
const CHAT_CHANNEL_ID = '1001'
const CHAT_IMAGE_FILE = 'ui-smoke-image.png'
const CHAT_IMAGE_URL = 'https://gchat.qpic.cn/gchatpic_new/1001/100100-514-UI-SMOKE/0'
const AUTHORITY_IMPORT_PID = '100003'
const AUTHORITY_IMPORT_NAME = 'E2E Authority 导入用户'
const IDENTITY_PENDING_MEMBER_ID = '200201'
const IDENTITY_BOUND_MEMBER_ID = '200202'
const IDENTITY_RELEASED_MEMBER_ID = '200203'
const TRANSPARENT_PNG_BASE64 = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII='
const DATA_AVATAR = 'data:image/png;base64,' + TRANSPARENT_PNG_BASE64

module.exports = function uiSmokeSeed(ctx) {
  const sentMessages = []
  const recalledMessages = []
  let messageSeq = 0
  const router = ctx.root.server || ctx.server || ctx.router

  ctx.on('ready', async () => {
    try {
      const [report] = await ctx.database.get(MODERATION_REPORT_TABLE, { id: REPORT_ID })
      if (!report) {
        await ctx.database.create(MODERATION_REPORT_TABLE, {
          id: REPORT_ID,
          platform: 'onebot',
          botSelfId: '514',
          guildId: '1001',
          channelId: '2001',
          reporterMemberId: '100999',
          targetMemberId: '200200',
          reason: 'E2E 处置中心举报 dismiss-report-token',
          aiStatus: 'completed',
          aiSeverity: 'medium',
          aiSummary: 'E2E 处置中心 AI 摘要',
          createdAt: CREATED_AT,
          updatedAt: CREATED_AT,
        })
      }

      const [event] = await ctx.database.get(MODERATION_EVENT_TABLE, { id: EVENT_ID })
      if (!event) {
        await ctx.database.create(MODERATION_EVENT_TABLE, {
          id: EVENT_ID,
          platform: 'onebot',
          botSelfId: '514',
          guildId: '1001',
          channelId: '2001',
          memberId: '200200',
          type: 'report_created',
          level: 'medium',
          summary: 'E2E 处置中心关联事件 report-related-token',
          payload: {
            reason: 'E2E 处置中心举报 dismiss-report-token',
            reportID: REPORT_ID,
          },
          createdAt: CREATED_AT,
          updatedAt: CREATED_AT,
        })
      }

      await seedAuthorityImportUser(ctx)
      await seedIdentityGuardMembers(ctx)
      installSmokeBot(ctx, sentMessages, recalledMessages, () => ++messageSeq)
      console.log('${SEED_READY_MESSAGE}')
    } catch (error) {
      console.error('[ui-smoke-seed] failed to prepare smoke data', error)
      throw error
    }
  })

  router.post('/__stuhelper-ui-smoke/chat/incoming', async (koa) => {
    const body = koa.request.body || {}
    const includeImage = body.includeImage !== false
    const messageId = emitChatMessage(ctx, {
      messageId: String(body.messageId || 'ui-smoke-incoming-' + (++messageSeq)),
      userId: String(body.userId || '100100'),
      username: String(body.username || 'E2E 聊天用户'),
      content: String(body.content || 'E2E incoming chat message'),
      includeImage,
      isSelf: false,
    })
    koa.body = { success: true, messageId }
  })

  router.get('/__stuhelper-ui-smoke/chat/actions', (koa) => {
    koa.body = { success: true, sentMessages, recalledMessages }
  })
}

module.exports.name = 'stuhelper-ui-smoke-seed'
module.exports.inject = ['database']

async function seedAuthorityImportUser(ctx) {
  const [existingUser] = await ctx.database.get('user', { name: AUTHORITY_IMPORT_NAME })
  const user = existingUser || await ctx.database.create('user', {
    name: AUTHORITY_IMPORT_NAME,
    authority: 2,
  })

  const existingBindings = await ctx.database.get('binding', {
    aid: user.id,
    platform: 'onebot',
    pid: AUTHORITY_IMPORT_PID,
  })
  if (existingBindings.length === 0) {
    await ctx.database.create('binding', {
      aid: user.id,
      platform: 'onebot',
      pid: AUTHORITY_IMPORT_PID,
    })
  }
}

async function seedIdentityGuardMembers(ctx) {
  const createdAt = new Date('2026-05-25T00:20:00.000Z')
  const records = [
    guardMemberRecord({
      id: 'ui-smoke-identity-pending-1',
      guildId: '1001',
      channelId: '2001',
      memberId: IDENTITY_PENDING_MEMBER_ID,
      memberName: 'E2E 待认证成员',
      verificationState: 'unbound',
      deadlineAt: new Date('2026-05-26T00:20:00.000Z'),
      lastError: '等待用户绑定 StuHelper 账号',
      createdAt,
    }),
    guardMemberRecord({
      id: 'ui-smoke-identity-bound-1',
      guildId: '1002',
      channelId: '2002',
      memberId: IDENTITY_BOUND_MEMBER_ID,
      memberName: 'E2E 待完善认证成员',
      verificationState: 'bound_unverified',
      deadlineAt: new Date('2026-05-26T00:40:00.000Z'),
      createdAt,
    }),
    guardMemberRecord({
      id: 'ui-smoke-identity-released-1',
      guildId: '1001',
      channelId: '2001',
      memberId: IDENTITY_RELEASED_MEMBER_ID,
      memberName: 'E2E 已释放成员',
      verificationState: 'verified',
      deadlineAt: new Date('2026-05-25T01:20:00.000Z'),
      releasedAt: new Date('2026-05-25T01:00:00.000Z'),
      createdAt,
    }),
  ]

  for (const record of records) {
    const [existing] = await ctx.database.get(GUARD_MEMBER_TABLE, { id: record.id })
    if (!existing) {
      await ctx.database.create(GUARD_MEMBER_TABLE, record)
    }
  }
}

function guardMemberRecord(input) {
  return {
    platform: 'onebot',
    botSelfId: '514',
    admissionSessionID: 'session-' + input.id,
    backendSyncPending: false,
    joinedAt: input.createdAt,
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: input.createdAt,
    reminderSentAt: input.createdAt,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    updatedAt: input.createdAt,
    ...input,
  }
}

function installSmokeBot(ctx, sentMessages, recalledMessages, nextSeq) {
  const bot = ctx.bots[0]
  if (!bot) {
    throw new Error('ui smoke mock bot is not registered')
  }

  bot.platform = 'onebot'
  bot.selfId = '514'
  bot.user = { id: '514', name: 'E2E Bot', avatar: DATA_AVATAR }
  bot.internal = {
    getImage: async (file) => {
      if (file !== CHAT_IMAGE_FILE) {
        throw new Error('unexpected ui smoke image file: ' + file)
      }
      return { base64: TRANSPARENT_PNG_BASE64, file }
    },
  }
  bot.getLogin = async () => ({ user: bot.user })
  bot.getGuild = async (guildId) => ({
    id: guildId,
    name: 'E2E 聊天群',
    avatar: DATA_AVATAR,
  })
  bot.getUser = async (userId) => ({
    id: userId,
    name: userId === '100100' ? 'E2E 聊天用户' : 'E2E 用户 ' + userId,
    avatar: DATA_AVATAR,
  })
  bot.getGuildMember = async (_guildId, userId) => ({
    user: { id: userId, name: userId === '100100' ? 'E2E 聊天用户' : 'E2E 成员 ' + userId, avatar: DATA_AVATAR },
    nick: userId === '100100' ? 'E2E 聊天用户' : 'E2E 成员 ' + userId,
    roles: userId === '100001' ? ['owner'] : userId === '100002' ? ['admin'] : [],
    title: userId === '100100' ? '自动化成员' : '',
  })
  bot.getGuildMemberList = async () => ({
    data: [
      {
        user: { id: '100001', name: 'E2E 群主', avatar: DATA_AVATAR },
        nick: 'E2E 群主',
        roles: ['owner'],
      },
      {
        user: { id: '100002', name: 'E2E 管理员', avatar: DATA_AVATAR },
        nick: 'E2E 管理员',
        roles: ['admin'],
      },
      {
        user: { id: '100100', name: 'E2E 聊天用户', avatar: DATA_AVATAR },
        nick: 'E2E 聊天用户',
        roles: [],
        title: '自动化成员',
      },
    ],
    next: undefined,
  })
  bot.sendMessage = async (channelId, content, guildId) => {
    const messageId = 'ui-smoke-outgoing-' + nextSeq()
    sentMessages.push({ channelId, guildId, content, messageId })
    emitChatMessage(ctx, {
      messageId,
      userId: bot.selfId,
      username: 'E2E Bot',
      content: String(content),
      includeImage: false,
      isSelf: true,
    })
    return [messageId]
  }
  bot.deleteMessage = async (channelId, messageId) => {
    recalledMessages.push({ channelId, messageId })
  }
}

function emitChatMessage(ctx, input) {
  const bot = ctx.bots[0]
  const elements = h.parse(input.content)
  if (input.includeImage) {
    elements.push(h('img', { src: CHAT_IMAGE_URL, file: CHAT_IMAGE_FILE }))
  }

  const session = bot.session({
    type: input.isSelf ? 'send' : 'message',
    platform: 'onebot',
    selfId: bot.selfId,
    user: {
      id: input.userId,
      name: input.username,
      avatar: DATA_AVATAR,
    },
    guild: {
      id: CHAT_GUILD_ID,
      name: 'E2E 聊天群',
      avatar: DATA_AVATAR,
    },
    channel: {
      id: CHAT_CHANNEL_ID,
      name: 'E2E 聊天频道',
      type: Universal.Channel.Type.TEXT,
    },
    message: {
      id: input.messageId,
      messageId: input.messageId,
      content: input.content,
      elements,
    },
  })
  session.author = {
    name: input.username,
    nick: input.username,
    avatar: DATA_AVATAR,
  }
  session.content = input.content
  session.elements = elements
  session.messageId = input.messageId
  session.timestamp = Date.now()
  session.guildName = 'E2E 聊天群'
  session.channelName = 'E2E 聊天频道'
  ctx.emit(input.isSelf ? 'send' : 'message', session)
  return input.messageId
}
`)
}

async function seedSmokeData(dataDir) {
  await mkdir(dataDir, { recursive: true })
  await writeFile(join(dataDir, 'command_logs.json'), JSON.stringify([
    {
      id: 'ui-smoke-command-log-1',
      timestamp: '2026-05-25T00:00:00.000Z',
      userId: '100000',
      username: 'E2E 日志用户',
      userAuthority: 4,
      guildId: '1001',
      guildName: 'E2E 日志群',
      channelId: '2001',
      platform: 'onebot',
      command: 'e2e.log-search',
      args: ['--case', 'log-search'],
      options: {
        source: 'ui-smoke',
        match: 'drawer-match-token',
      },
      success: true,
      executionTime: 42,
      result: 'E2E command log result drawer-match-token',
      messageId: 'ui-smoke-message-1',
      isPrivate: false,
    },
  ], null, 2))
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
      writeJSON(response, qqVerificationStatus(decodeURIComponent(qqVerificationMatch[1])))
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

function qqVerificationStatus(qqID) {
  if (qqID === '200202') {
    return {
      qqID,
      userID: 4202,
      boundAt: '2026-05-25T00:30:00.000Z',
      verificationState: 'bound_unverified',
      profileVerificationStatus: 'pending',
      studentVerified: false,
    }
  }

  if (qqID === '200203') {
    return {
      qqID,
      userID: 4203,
      boundAt: '2026-05-25T00:40:00.000Z',
      verificationState: 'verified',
      profileVerificationStatus: 'verified',
      studentVerified: true,
    }
  }

  return {
    qqID,
    userID: null,
    boundAt: null,
    verificationState: 'unbound',
    profileVerificationStatus: 'unverified',
    studentVerified: false,
  }
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
