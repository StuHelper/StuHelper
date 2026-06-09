import type { Universal } from 'koishi'

import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type StuhelperAdmissionReminderDeliveryConfig,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

export interface AdmissionReminderDeliveryResult {
  readonly messageID?: string
  readonly groupMessageID?: string
  readonly directMessageID?: string
  readonly cancelled?: boolean
}

export type AdmissionReminderDeliveryConfigProvider =
  | Partial<StuhelperAdmissionReminderDeliveryConfig>
  | (() =>
      | Partial<StuhelperAdmissionReminderDeliveryConfig>
      | undefined
      | Promise<Partial<StuhelperAdmissionReminderDeliveryConfig> | undefined>
    )

export interface AdmissionReminderDeliveryInput {
  readonly bot: Universal.Methods
  readonly guildId: string
  readonly channelId: string
  readonly memberId: string
  readonly content: string
  readonly delivery?: Partial<StuhelperAdmissionReminderDeliveryConfig>
  readonly messages?: Partial<StuhelperGroupGuardMessageConfig>
  readonly sendGroupMessage?: (content: string) => Promise<unknown>
  readonly shouldSend?: () => boolean | Promise<boolean>
}

interface ResolvedAdmissionReminderDeliveryConfig {
  readonly groupEnabled: boolean
  readonly directEnabled: boolean
}

interface DeliveryAttemptError {
  readonly channel: 'group' | 'direct'
  readonly error: unknown
}

interface SendAttemptResult {
  readonly sent: boolean
  readonly messageID?: string
}

type BotWithInternal = Universal.Methods & {
  readonly internal?: Record<string, unknown>
}

type FriendList = {
  readonly data?: readonly unknown[]
  readonly next?: string
}

export function resolveAdmissionReminderDeliveryConfig(
  input?: Partial<StuhelperAdmissionReminderDeliveryConfig>,
): ResolvedAdmissionReminderDeliveryConfig {
  const groupEnabled = input?.groupEnabled !== false
  const directEnabled = input?.directEnabled === true
  return { groupEnabled, directEnabled }
}

export async function resolveAdmissionReminderDeliveryInput(
  provider?: AdmissionReminderDeliveryConfigProvider,
) {
  if (typeof provider === 'function') {
    return provider()
  }
  return provider
}

export async function sendAdmissionReminderMessage(
  input: AdmissionReminderDeliveryInput,
): Promise<AdmissionReminderDeliveryResult> {
  if (!input.content) return {}
  if (!await shouldSend(input)) return { cancelled: true }

  const delivery = resolveAdmissionReminderDeliveryConfig(input.delivery)
  if (!delivery.groupEnabled && !delivery.directEnabled) {
    return {}
  }
  const errors: DeliveryAttemptError[] = []
  let groupMessageID: string | undefined
  let directMessageID: string | undefined
  let sent = false

  if (delivery.groupEnabled) {
    if (!await shouldSend(input)) return { cancelled: true }
    try {
      if (await shouldSend(input)) {
        const result = input.sendGroupMessage
          ? await input.sendGroupMessage(input.content)
          : await input.bot.sendMessage(input.channelId, input.content)
        groupMessageID = firstMessageID(result)
        sent = true
      }
    } catch (error) {
      errors.push({ channel: 'group', error })
    }
  }

  if (delivery.directEnabled) {
    if (!await shouldSend(input)) {
      return sent
        ? { messageID: groupMessageID, groupMessageID }
        : { cancelled: true }
    }
    try {
      if (await shouldSend(input)) {
        const result = await sendDirectAdmissionReminder(input)
        directMessageID = result.messageID
        sent = result.sent || sent
      }
    } catch (error) {
      errors.push({ channel: 'direct', error })
    }
  }

  if (!sent) {
    if (!await shouldSend(input)) return { cancelled: true }
    throw deliveryError(errors, input.messages)
  }

  return {
    messageID: groupMessageID ?? directMessageID,
    groupMessageID,
    directMessageID,
  }
}

async function shouldSend(input: AdmissionReminderDeliveryInput) {
  return input.shouldSend ? input.shouldSend() : true
}

async function sendDirectAdmissionReminder(input: AdmissionReminderDeliveryInput): Promise<SendAttemptResult> {
  const isFriend = await isBotFriend(input.bot, input.memberId)
  if (isFriend === true) {
    return sendStandardPrivateMessage(input.bot, input.memberId, input.content)
  }

  try {
    return await sendStandardPrivateMessage(input.bot, input.memberId, input.content, input.guildId)
  } catch (standardError) {
    const internalResult = await trySendOneBotTemporaryMessage(input)
    if (internalResult) return internalResult
    throw standardError
  }
}

async function sendStandardPrivateMessage(
  bot: Universal.Methods,
  memberId: string,
  content: string,
  guildId?: string,
): Promise<SendAttemptResult> {
  const result = await bot.sendPrivateMessage(memberId, content, guildId)
  return { sent: true, messageID: firstMessageID(result) }
}

async function isBotFriend(bot: Universal.Methods, memberId: string): Promise<boolean | undefined> {
  let next: string | undefined
  for (let page = 0; page < 20; page += 1) {
    let list: FriendList
    try {
      list = normalizeFriendList(await bot.getFriendList(next))
    } catch {
      return undefined
    }

    if (list.data?.some((friend) => friendIDOf(friend) === memberId)) {
      return true
    }
    if (!list.next) return false
    next = list.next
  }
  return undefined
}

function normalizeFriendList(value: unknown): FriendList {
  if (Array.isArray(value)) return { data: value }
  return value as FriendList
}

async function trySendOneBotTemporaryMessage(input: AdmissionReminderDeliveryInput): Promise<SendAttemptResult | null> {
  const internal = (input.bot as BotWithInternal).internal
  const sendPrivateMsg = internalFunction(internal, 'sendPrivateMsg') ?? internalFunction(internal, 'send_private_msg')
  if (!sendPrivateMsg) return null

  const payloads = [
    {
      user_id: onebotID(input.memberId),
      group_id: onebotID(input.guildId),
      message: input.content,
    },
    {
      userId: input.memberId,
      groupId: input.guildId,
      message: input.content,
    },
  ]
  let lastError: unknown
  for (const payload of payloads) {
    try {
      const result = await sendPrivateMsg.call(internal, payload)
      return { sent: true, messageID: firstMessageID(result) }
    } catch (error) {
      lastError = error
    }
  }

  if (lastError) throw lastError
  return null
}

function internalFunction(internal: Record<string, unknown> | undefined, name: string) {
  const value = internal?.[name]
  return typeof value === 'function' ? value : undefined
}

function friendIDOf(friend: unknown): string | undefined {
  if (!friend || typeof friend !== 'object') return undefined
  const record = friend as Record<string, unknown>
  const value = record.id ?? record.userId ?? record.userID ?? record.user_id
  return value == null ? undefined : String(value)
}

function onebotID(value: string) {
  if (!/^\d+$/.test(value)) return value
  const numeric = Number(value)
  return Number.isSafeInteger(numeric) ? numeric : value
}

export function firstMessageID(result: unknown): string | undefined {
  if (Array.isArray(result)) {
    return firstMessageID(result[0])
  }
  if (typeof result === 'string') return result
  if (typeof result === 'number' || typeof result === 'bigint') return String(result)
  if (!result || typeof result !== 'object') return undefined

  const record = result as Record<string, unknown>
  const value = record.messageID ?? record.messageId ?? record.message_id
  if (value != null) return String(value)
  return firstMessageID(record.data)
}

function deliveryError(
  errors: readonly DeliveryAttemptError[],
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  if (errors.length === 1) {
    return errors[0].error
  }
  return new Error(deliveryMessage(messages, 'admissionReminderDeliveryFailure', {
    errors: errors.map((error) => formatAttemptError(error, messages)).join('；'),
  }))
}

function formatAttemptError(
  error: DeliveryAttemptError,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  const channel = error.channel === 'group'
    ? deliveryMessage(messages, 'admissionReminderDeliveryGroupChannelLabel')
    : deliveryMessage(messages, 'admissionReminderDeliveryDirectChannelLabel')
  return `${channel}: ${errorMessage(error.error)}`
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

function deliveryMessage(
  messages: Partial<StuhelperGroupGuardMessageConfig> | undefined,
  key: keyof ReturnType<typeof resolveGroupGuardMessages>,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(resolveGroupGuardMessages(messages)[key], variables)
}
