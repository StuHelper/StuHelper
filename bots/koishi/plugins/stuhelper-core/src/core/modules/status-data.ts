import type { Context } from 'koishi'
import os from 'os'

import type { DataManager } from '../data'

export type StatusData = {
  readonly os: {
    readonly platform: string
    readonly release: string
    readonly arch: string
    readonly hostname: string
    readonly uptime: number
  }
  readonly process: {
    readonly uptime: number
    readonly version: string
    readonly memory: {
      readonly rss: number
      readonly heapTotal: number
      readonly heapUsed: number
    }
  }
  readonly system: {
    readonly cpuModel: string
    readonly cpuCount: number
    readonly totalMem: number
    readonly usedMem: number
    readonly loadavg: number[]
  }
  readonly bot: {
    readonly version: string
    readonly plugins: number
  }
  readonly stuhelperGroupCenter: {
    readonly version: string
    readonly groupCount: number
    readonly logCount: unknown
  }
}

export async function getSystemStatusData(ctx: Context, data: DataManager): Promise<StatusData> {
  const memoryUsage = process.memoryUsage()
  const totalMem = os.totalmem()
  const freeMem = os.freemem()
  const usedMem = totalMem - freeMem
  const cpus = os.cpus()
  const pkg = require('../../../package.json')

  return {
    os: {
      platform: os.platform(),
      release: os.release(),
      arch: os.arch(),
      hostname: os.hostname(),
      uptime: os.uptime(),
    },
    process: {
      uptime: process.uptime(),
      version: process.version,
      memory: {
        rss: memoryUsage.rss,
        heapTotal: memoryUsage.heapTotal,
        heapUsed: memoryUsage.heapUsed,
      },
    },
    system: {
      cpuModel: cpus[0]?.model || 'Unknown CPU',
      cpuCount: cpus.length,
      totalMem,
      usedMem,
      loadavg: os.loadavg(),
    },
    bot: {
      version: '4.18.7',
      plugins: ctx.registry.size,
    },
    stuhelperGroupCenter: {
      version: `${pkg.version || 'unknown'}`,
      groupCount: Object.keys(await data.groupConfig.getAll()).length,
      logCount: (await data.commandLogs.getAll() as any).length,
    },
  }
}
