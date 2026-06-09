import { Context, Schema, type Session } from 'koishi'

import {
  BindingRuntimeSettingsStore,
  DEFAULT_BINDING_COMMAND,
  DEFAULT_BINDING_RUNTIME_SETTINGS,
  PlatformAPIError,
  createBindingPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  registerBindingRuntimeSettingsModel,
  renderMessageTemplate,
  type PlatformVerificationState,
  type BindingRuntimeSettingsRecord,
  type StuhelperBindingPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-binding'
export const inject = ['database']

export type Config = StuhelperBindingPluginConfig

export const Config: Schema<Config> = createBindingPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerBindingRuntimeSettingsModel(ctx)
  const logger = createPluginLogger(ctx, 'binding')
  const platform = createPlatformClient(config.platform)
  const settingsStore = new BindingRuntimeSettingsStore(ctx, DEFAULT_BINDING_RUNTIME_SETTINGS)

  ctx.middleware(async (session, next) => {
    const settings = await settingsStore.getSettings()
    const parsed = parseBindingCommand(session, settings.command)
    if (!parsed) {
      return next()
    }
    const messages = settings.messages

    if (!session.isDirect) {
      return renderMessageTemplate(messages.directOnly)
    }
    if (!parsed.code) {
      return renderMessageTemplate(messages.missingCode, { command: settings.command })
    }

    try {
      const result = await platform.consumeQQBindingCode({
        code: parsed.code,
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

  logger.info('绑定插件已加载，命令字和提示文案由 WebUI runtime settings 控制')
}

function parseBindingCommand(
  session: Session,
  configuredCommand: string,
) {
  const content = session.content?.trim()
  if (!content) {
    return null
  }
  const command = configuredCommand.trim() || DEFAULT_BINDING_COMMAND
  if (content === command) {
    return { code: '' }
  }
  if (!content.startsWith(command)) {
    return null
  }
  const rest = content.slice(command.length)
  if (!/^\s/.test(rest)) {
    return null
  }
  return { code: rest.trim() }
}

function buildBindingSuccessMessage(
  state: PlatformVerificationState,
  messages: BindingRuntimeSettingsRecord['messages'],
) {
  if (state === 'verified') {
    return renderMessageTemplate(messages.successVerified)
  }
  return renderMessageTemplate(messages.successUnverified)
}

function resolveBindingErrorMessage(
  error: unknown,
  messages: BindingRuntimeSettingsRecord['messages'],
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
  inject,
  Config,
  apply,
}
