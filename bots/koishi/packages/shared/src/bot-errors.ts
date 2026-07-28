const ONEBOT_SET_GROUP_BAN_PERMISSION_RETCODES = new Set([1200])

export function isOneBotSetGroupBanPermissionError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error)
  if (!message.includes('set_group_ban')) {
    return false
  }
  const retcode = matchOneBotRetcode(message)
  return retcode !== null && ONEBOT_SET_GROUP_BAN_PERMISSION_RETCODES.has(retcode)
}

export function formatBotOperationError(error: unknown) {
  if (isOneBotSetGroupBanPermissionError(error)) {
    return '机器人缺少群管理员权限，无法修改成员禁言状态。请先将机器人设为群管理员后重试。'
  }
  return error instanceof Error ? error.message : String(error)
}

function matchOneBotRetcode(message: string) {
  const match = message.match(/\bretcode\b\s*(?:[:=]|,)?\s*(\d+)/)
  if (!match) {
    return null
  }
  const value = Number(match[1])
  return Number.isSafeInteger(value) ? value : null
}
