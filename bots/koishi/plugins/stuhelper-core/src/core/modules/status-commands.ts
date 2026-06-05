import { segment, type Context } from 'koishi'

import type { DataManager } from '../data'
import type { RuntimeCommand, RuntimeCommandDef } from '../../runtime/types'
import { commandErrorMessage } from './command-error-message'
import { getSystemStatusData } from './status-data'
import { renderStatusHtml } from './status-html'

type StatusCommandResult = string | ReturnType<typeof segment.image>

export interface StatusCommandHost {
  readonly ctx: Context
  readonly data: DataManager
  registerCommand(def: RuntimeCommandDef): RuntimeCommand
}

export function registerStatusCommands(host: StatusCommandHost): void {
  host.registerCommand({
    name: 'gstatus',
    desc: '查看系统状态',
    permNode: 'status.view',
    permDesc: '查看系统状态图片',
    skipAuth: true,
    usage: '生成系统状态信息图片',
  }).action(async () => {
    return renderStatusImage(host)
  })
}

async function renderStatusImage(host: StatusCommandHost): Promise<StatusCommandResult> {
  if (!host.ctx.puppeteer) {
    return '错误：未安装 puppeteer 插件，无法生成状态图片。'
  }

  try {
    const data = await getSystemStatusData(host.ctx, host.data)
    const html = renderStatusHtml(data)
    const page = await host.ctx.puppeteer.page()
    try {
      await page.setViewport({ width: 900, height: 800, deviceScaleFactor: 2 })
      await page.setContent(html, { waitUntil: 'load' })

      const element = await page.$('.container')
      if (element) {
        const image = await element.screenshot({ encoding: 'binary', omitBackground: true })
        return segment.image(image, 'image/png')
      }

      const fullPage = await page.screenshot({ encoding: 'binary', fullPage: true })
      return segment.image(fullPage, 'image/png')
    } finally {
      await page.close()
    }
  } catch (error) {
    return `生成状态图失败：${commandErrorMessage(error)}`
  }
}
