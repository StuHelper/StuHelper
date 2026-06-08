import { Context, Schema } from 'koishi'

import {
  PlatformAPIError,
  createBindingPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  renderMessageTemplate,
  resolveBindingMessages,
  type PlatformVerificationState,
  type StuhelperBindingPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-binding'

export type Config = StuhelperBindingPluginConfig

export const Config: Schema<Config> = createBindingPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'binding')
  const platform = createPlatformClient(config.platform)
  const messages = resolveBindingMessages(config.binding.messages)

  ctx.command(`${config.binding.command} <code:text>`, '绑定 StuHelper 账号')
    .action(async ({ session }, code) => {
      if (!session) {
        throw new Error('binding command requires an active session')
      }
      if (!session.isDirect) {
        return renderMessageTemplate(messages.directOnly)
      }
      if (!code?.trim()) {
        return renderMessageTemplate(messages.missingCode, { command: config.binding.command })
      }

      try {
        const result = await platform.consumeQQBindingCode({
          code: code.trim(),
          qqID: session.userId,
        })
        logger.info('qq binding succeeded', {
          qqID: session.userId,
          userID: result.binding.userID,
          verificationState: result.verificationState.verificationState,
        })
        return buildBindingSuccessMessage(result.verificationState.verificationState, messages)
      } catch (error) {
        const message = resolveBindingErrorMessage(error, messages)
        logger.warn('qq binding failed', {
          qqID: session.userId,
          error: error instanceof Error ? error.message : String(error),
        })
        return message
      }
    })

  logger.info(`绑定插件已加载，命令字：${config.binding.command}`)
}

function buildBindingSuccessMessage(
  state: PlatformVerificationState,
  messages: ReturnType<typeof resolveBindingMessages>,
) {
  if (state === 'verified') {
    return renderMessageTemplate(messages.successVerified)
  }
  return renderMessageTemplate(messages.successUnverified)
}

function resolveBindingErrorMessage(
  error: unknown,
  messages: ReturnType<typeof resolveBindingMessages>,
) {
  if (!(error instanceof PlatformAPIError)) {
    return renderMessageTemplate(messages.unavailable)
  }
  if (error.status === 400) {
    return renderMessageTemplate(messages.invalidCode)
  }
  if (error.status === 401) {
    return renderMessageTemplate(messages.unauthorized)
  }
  if (error.status === 409) {
    return renderMessageTemplate(messages.conflict)
  }
  if (error.status === 503) {
    return renderMessageTemplate(messages.notConfigured)
  }
  return renderMessageTemplate(messages.unavailable)
}

export default {
  name,
  Config,
  apply,
}
