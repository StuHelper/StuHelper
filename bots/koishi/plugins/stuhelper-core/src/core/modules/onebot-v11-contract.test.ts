import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = dirname(fileURLToPath(import.meta.url))
const coreRoot = join(currentDir, '../..')

test('runtime modules do not call non-standard OneBot essence-message actions', async () => {
  const files = await Promise.all([
    readCoreFile('core/modules/messageManage.module.ts'),
    readCoreFile('core/settings/settings.manager.ts'),
    readCoreFile('types/index.ts'),
    readCoreFile('config.ts'),
  ])

  for (const content of files) {
    assert.doesNotMatch(content, /setEssenceMsg|deleteEssenceMsg|essence|精华/)
  }
})

test('OneBot group-management commands reject missing group ids before adapter calls', async () => {
  const [memberCommands, orderBanCommands, orderGroupCommands] = await Promise.all([
    readCoreFile('core/modules/member-manage-commands.ts'),
    readCoreFile('core/modules/order-manage-ban-commands.ts'),
    readCoreFile('core/modules/order-manage-group-commands.ts'),
  ])

  assert.match(memberCommands, /失败：缺少群号/)
  assert.match(orderBanCommands, /失败：缺少群号/)
  assert.match(orderGroupCommands, /失败：缺少群号/)
})

test('delmsg uses canonical quote id with legacy messageId fallback', async () => {
  const content = await readCoreFile('core/modules/messageManage.module.ts')

  assert.match(content, /resolveQuoteMessageId\(session\.quote\)/)
  assert.match(content, /quote\.id \|\| quote\.messageId/)
  assert.match(content, /deleteMessage\(session\.channelId, messageId\)/)
})

test('OneBot internal group admin calls fail explicitly when unsupported', async () => {
  const [commands, helper] = await Promise.all([
    readCoreFile('core/modules/member-manage-commands.ts'),
    readCoreFile('core/onebot-internal.ts'),
  ])

  assert.match(commands, /requireOneBotInternalMethod\(session\.bot, 'setGroupAdmin', 'set_group_admin'\)/)
  assert.match(helper, /当前适配器不支持 OneBot \$\{actionName\}/)
  assert.doesNotMatch(commands, /internal\?\.setGroupAdmin\([^)]*\)/)
})

test('OneBot-only internals go through the shared internal helper', async () => {
  const files = await Promise.all([
    readCoreFile('core/api/chat-image-fetch.ts'),
    readCoreFile('core/modules/crossGroupManage.module.ts'),
    readCoreFile('core/modules/event-handlers.ts'),
    readCoreFile('core/modules/event-support.ts'),
    readCoreFile('core/modules/getauth.module.ts'),
    readCoreFile('core/modules/member-manage-commands.ts'),
    readCoreFile('core/modules/member-manage-title-commands.ts'),
    readCoreFile('core/modules/order-manage-group-commands.ts'),
  ])

  for (const content of files) {
    assert.doesNotMatch(content, /session\.bot\.internal/)
    assert.doesNotMatch(content, /\(session\.bot as any\)/)
    assert.doesNotMatch(content, /\(bot as any\)\.internal/)
    assert.doesNotMatch(content, /botInternal\(/)
  }
})

test('whole-guild mute uses Koishi universal channel mute API', async () => {
  const content = await readCoreFile('core/modules/order-manage-group-commands.ts')

  assert.match(content, /session\.bot\.muteChannel\(/)
  assert.doesNotMatch(content, /setGroupWholeBan/)
})

test('request approvals use Koishi universal request APIs', async () => {
  const handlers = await readCoreFile('core/modules/event-handlers.ts')
  const support = await readCoreFile('core/modules/event-support.ts')

  assert.match(handlers, /session\.bot\.handleFriendRequest\(/)
  assert.match(handlers, /session\.bot\.handleGuildRequest\(/)
  assert.match(handlers, /session\.bot\.handleGuildMemberRequest\(/)
  assert.doesNotMatch(handlers, /setFriendAddRequest|setGroupAddRequest/)
  assert.doesNotMatch(support, /session\.event as \{ _data/)
})

test('console chat prefers Koishi universal login and message APIs', async () => {
  const content = await readCoreFile('core/api/chat-message-broadcast.ts')

  assert.match(content, /bot\.getLogin\(/)
  assert.match(content, /bot\.getMessage\(/)
  assert.doesNotMatch(content, /getLoginInfo/)
  assert.doesNotMatch(content, /internal\?\.getMsg|internal\.getMsg/)
})

test('getauth reads standard guild member data before OneBot-only details', async () => {
  const content = await readCoreFile('core/modules/getauth.module.ts')

  assert.match(content, /session\.bot\.getGuildMember\(/)
  assert.match(content, /readOneBotMuteLine/)
})

async function readCoreFile(relativePath: string): Promise<string> {
  return readFile(join(coreRoot, relativePath), 'utf8')
}
