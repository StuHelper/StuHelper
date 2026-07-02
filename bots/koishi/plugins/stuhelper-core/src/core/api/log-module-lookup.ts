import { type CommandLogRecord, normalizeCommandLogRecords } from '../data/command-log-records'
import type { WebSocketAPIContext } from './api-context'

/**
 * 命令日志唯一来源是 DataManager 的 commandLogs 存储。
 * （旧运行时模块体系已删除，原 findLogModule 的模块表查找路径随之移除。）
 */
export async function readCommandLogs(api: WebSocketAPIContext): Promise<CommandLogRecord[]> {
  return normalizeCommandLogRecords(api.service.data.commandLogs.getAll()).reverse()
}
