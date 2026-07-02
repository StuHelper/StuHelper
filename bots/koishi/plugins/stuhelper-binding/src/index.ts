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

import { BindingAttemptGuard, type BindingAttemptDecision } from './attempt-guard'

const MS_PER_SECOND = 1_000
const MS_PER_MINUTE = 60_000

export const name = 'stuhelper-binding'
export const inject = ['database']

export type Config = StuhelperBindingPluginConfig

export const Config: Schema<Config> = createBindingPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  registerBindingRuntimeSettingsModel(ctx)
  const logger = createPluginLogger(ctx, 'binding')
  const platform = createPlatformClient(config.platform)
  const settingsStore = new BindingRuntimeSettingsStore(ctx, DEFAULT_BINDING_RUNTIME_SETTINGS)
  const attemptGuard = new BindingAttemptGuard()

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

    const attemptKey = bindingAttemptKey(session)
    const decision = attemptGuard.check(attemptKey)
    if (decision.allowed === false) {
      logger.warn('qq binding attempt rejected', {
        qqID: session.userId,
        reason: decision.reason,
        retryAfterMs: decision.retryAfterMs,
      })
      return buildAttemptRejectionMessage(decision, messages)
    }

    try {
      const result = await platform.consumeQQBindingCode({
        code: parsed.code,
        qqID: session.userId,
      })
      attemptGuard.markSuccess(attemptKey)
      logger.info('qq binding succeeded', {
        qqID: session.userId,
        userID: result.binding.userID,
        verificationState: result.verificationState.verificationState,
      })
      return buildBindingSuccessMessage(result.verificationState.verificationState, messages)
    } catch (error) {
      if (isInvalidBindingCodeError(error)) {
        attemptGuard.markFailure(attemptKey)
      }
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

function bindingAttemptKey(session: Session) {
  return `${session.platform}:${session.userId}`
}

function buildAttemptRejectionMessage(
  decision: Extract<BindingAttemptDecision, { allowed: false }>,
  messages: BindingRuntimeSettingsRecord['messages'],
) {
  if (decision.reason === 'locked_out') {
    return renderMessageTemplate(messages.tooManyFailures, {
      minutes: Math.ceil(decision.retryAfterMs / MS_PER_MINUTE),
      seconds: Math.ceil(decision.retryAfterMs / MS_PER_SECOND),
    })
  }
  return renderMessageTemplate(messages.rateLimited, {
    seconds: Math.ceil(decision.retryAfterMs / MS_PER_SECOND),
  })
}

function isInvalidBindingCodeError(error: unknown) {
  return error instanceof PlatformAPIError && error.status === 400
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
