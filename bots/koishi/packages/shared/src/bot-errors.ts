const ONEBOT_SET_GROUP_BAN_PERMISSION_RETCODES = new Set([1200])
const MAX_INSPECTED_ERROR_LENGTH = 16_384

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
  const inspected = message.slice(0, MAX_INSPECTED_ERROR_LENGTH)
  let searchFrom = 0

  while (searchFrom < inspected.length) {
    const marker = inspected.indexOf('retcode', searchFrom)
    if (marker < 0) return null
    searchFrom = marker + 'retcode'.length

    const before = marker > 0 ? inspected.charCodeAt(marker - 1) : -1
    const after = searchFrom < inspected.length ? inspected.charCodeAt(searchFrom) : -1
    if (isWordCode(before) || isWordCode(after)) {
      continue
    }

    let cursor = skipASCIIWhitespace(inspected, searchFrom)
    const separator = inspected[cursor]
    if (separator === ':' || separator === '=' || separator === ',') {
      cursor = skipASCIIWhitespace(inspected, cursor + 1)
    }

    const digitStart = cursor
    while (cursor < inspected.length && isASCIIDigit(inspected.charCodeAt(cursor))) {
      cursor += 1
    }
    if (cursor === digitStart) {
      continue
    }

    const value = Number(inspected.slice(digitStart, cursor))
    return Number.isSafeInteger(value) ? value : null
  }

  return null
}

function skipASCIIWhitespace(value: string, start: number): number {
  let cursor = start
  while (cursor < value.length) {
    const code = value.charCodeAt(cursor)
    if (code !== 0x09 && code !== 0x0a && code !== 0x0b && code !== 0x0c && code !== 0x0d && code !== 0x20) {
      break
    }
    cursor += 1
  }
  return cursor
}

function isASCIIDigit(code: number): boolean {
  return code >= 0x30 && code <= 0x39
}

function isWordCode(code: number): boolean {
  return isASCIIDigit(code)
    || (code >= 0x41 && code <= 0x5a)
    || code === 0x5f
    || (code >= 0x61 && code <= 0x7a)
}
