import { Context, Schema, Session } from 'koishi'

import {
  PlatformAPIError,
  createBindingPluginConfigSchema,
  createPlatformClient,
  createPluginLogger,
  type PlatformVerificationState,
  type StuhelperBindingPluginConfig,
} from '@stuhelper/koishi-shared'

export const name = 'stuhelper-binding'

export type Config = StuhelperBindingPluginConfig

export const Config: Schema<Config> = createBindingPluginConfigSchema()

export function apply(ctx: Context, config: Config) {
  const logger = createPluginLogger(ctx, 'binding')
  const platform = createPlatformClient(config.platform)

  ctx.command(`${config.binding.command} <code:text>`, '绑定 StuHelper 账号')
    .action(async ({ session }, code) => {
      if (!session) {
        throw new Error('binding command requires an active session')
      }
      if (!session.isDirect) {
        return '请在私聊中发送绑定命令。'
      }
      if (!code?.trim()) {
        return `请输入绑定码，例如：${config.binding.command} ABCD1234`
      }

      try {
        const result = await platform.consumeQQBindingCode({
          code: code.trim(),
          qqID: session.userId,
          qqNickname: resolveQQNickname(session),
        })
        logger.info('qq binding succeeded', {
          qqID: session.userId,
          userID: result.binding.userID,
          verificationState: result.verificationState.verificationState,
        })
        return buildBindingSuccessMessage(result.verificationState.verificationState)
      } catch (error) {
        const message = resolveBindingErrorMessage(error)
        logger.warn('qq binding failed', {
          qqID: session.userId,
          error: error instanceof Error ? error.message : String(error),
        })
        return message
      }
    })

  logger.info(`绑定插件已加载，命令字：${config.binding.command}`)
}

function resolveQQNickname(session: Session) {
  return session.username || session.event.user?.nick || session.author?.nick
}

function buildBindingSuccessMessage(state: PlatformVerificationState) {
  if (state === 'verified') {
    return '绑定成功，当前账号已完成学生认证，加入受控群时会自动放行。'
  }
  return '绑定成功。当前账号还未完成学生认证，请先回到 StuHelper 完成认证。'
}

function resolveBindingErrorMessage(error: unknown) {
  if (!(error instanceof PlatformAPIError)) {
    return '绑定失败，平台暂时不可用。'
  }
  if (error.status === 400) {
    return '绑定码无效或已过期，请重新生成后再试。'
  }
  if (error.status === 401) {
    return '机器人服务鉴权失败，请联系管理员检查后端配置。'
  }
  if (error.status === 409) {
    return '该 QQ 号或 StuHelper 账号已经绑定过其他对象。'
  }
  if (error.status === 503) {
    return '后端机器人接口尚未配置完成，请联系管理员。'
  }
  return '绑定失败，平台暂时不可用。'
}

export default {
  name,
  Config,
  apply,
}
