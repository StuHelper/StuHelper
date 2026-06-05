import type { RuntimeModuleInstance } from '../../runtime/types'
import type { CommandLogRecord, LogModule } from '../modules/log.module'
import { normalizeCommandLogRecords } from '../modules/command-log-records'
import type { WebSocketAPIContext } from './api-context'

export interface LogModuleReader {
  getAllLogs(): Promise<CommandLogRecord[]>
}

type ServiceWithOptionalGetModule = Omit<WebSocketAPIContext['service'], 'getModule'> & {
  getModule?: WebSocketAPIContext['service']['getModule']
}

export function findLogModule(api: WebSocketAPIContext): LogModuleReader | undefined {
  const service = api.service as ServiceWithOptionalGetModule
  const directModule = typeof service.getModule === 'function'
    ? service.getModule<LogModule>('log')
    : undefined

  return directModule ?? service.getAllModules().find(isLogModuleReader)
}

export async function readCommandLogs(api: WebSocketAPIContext): Promise<CommandLogRecord[]> {
  const logModule = findLogModule(api)
  if (logModule) {
    return logModule.getAllLogs()
  }

  return normalizeCommandLogRecords(api.service.data.commandLogs.getAll()).reverse()
}

function isLogModuleReader(module: RuntimeModuleInstance): module is RuntimeModuleInstance & LogModuleReader {
  return module.meta.name === 'log' && typeof (module as { getAllLogs?: unknown }).getAllLogs === 'function'
}
