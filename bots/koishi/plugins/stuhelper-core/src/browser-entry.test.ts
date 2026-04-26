import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { resolveBrowserEntry } from './browser-entry'

test('stuhelper-core 控制台生产入口必须使用 Koishi 可服务的 node_modules 路径', async () => {
  const entry = resolveBrowserEntry()

  assert.match(
    entry.prod,
    /node_modules\/koishi-plugin-stuhelper-core\/dist$/,
    '生产入口必须保留 node_modules 路径段，否则 Koishi console 会拒绝提供 @plugin 资源',
  )

  const packageJson = JSON.parse(
    await readFile(join(dirname(fileURLToPath(import.meta.url)), '../package.json'), 'utf8'),
  ) as { files?: string[] }
  assert.ok(
    packageJson.files?.includes('dist'),
    '插件发布文件必须包含 dist，生产入口的真实产物由 build 阶段生成。',
  )
})
