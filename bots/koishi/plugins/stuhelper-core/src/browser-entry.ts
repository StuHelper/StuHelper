import { resolve } from 'node:path'

const PACKAGE_NAME = 'koishi-plugin-stuhelper-core'

export interface BrowserEntryFiles {
  dev: string
  prod: string
}

export function resolveBrowserEntry(): BrowserEntryFiles {
  return {
    dev: resolve(__dirname, '../client/index.ts'),
    prod: resolve(__dirname, `../../../node_modules/${PACKAGE_NAME}/dist`),
  }
}
