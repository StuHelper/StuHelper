import { Context } from 'koishi'

export interface ExecuteCommandInput {
  readonly ctx: Context
  readonly session: any
  readonly commandName: string
  readonly args?: readonly string[]
  readonly options?: Readonly<Record<string, any>>
  readonly useAdmin?: boolean
}

export async function executeCommand(input: ExecuteCommandInput) {
  const { ctx, session, commandName } = input
  const logger = ctx.logger('stuhelper-core:utils')
  try {
    logger.info('准备执行命令: %s，参数: %s', commandName, JSON.stringify(input.args || []))
    const command = ctx.$commander.get(commandName, session)
    if (!command) return handleMissingCommand(logger, commandName)

    const commandSession = input.useAdmin ? createAdminCommandSession(session) : session
    if (input.useAdmin) logger.info('已临时提升权限至管理员权限(5)执行命令: %s', commandName)

    logger.info('正在执行命令: %s', commandName)
    const result = await command.execute({
      session: commandSession,
      args: [...(input.args || [])],
      options: input.options || {},
    })
    logger.info('命令 %s 执行结果: %o', commandName, result)
    return result
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    logger.error('%s %o', `执行命令 ${commandName} 失败: ${message}`, error)
    return `执行失败: ${message || '未知错误'}`
  }
}

function handleMissingCommand(logger: ReturnType<Context['logger']>, commandName: string): string {
  const message = `命令 ${commandName} 不存在`
  logger.error(message)
  return `执行失败: ${message}`
}

function createAdminCommandSession(session: any) {
  return Object.assign(Object.create(Object.getPrototypeOf(session)), session, {
    user: { ...(session.user || {}), authority: 5 },
  })
}
