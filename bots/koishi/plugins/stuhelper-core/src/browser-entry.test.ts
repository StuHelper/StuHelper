import assert from 'node:assert/strict'
import { access } from 'node:fs/promises'
import { join } from 'node:path'
import test from 'node:test'

import { resolveBrowserEntry } from './browser-entry'

test('stuhelper-core 控制台生产入口必须使用 Koishi 可服务的 node_modules 路径', async () => {
  const entry = resolveBrowserEntry()

  assert.match(
    entry.prod,
    /node_modules\/koishi-plugin-stuhelper-core\/dist$/,
    '生产入口必须保留 node_modules 路径段，否则 Koishi console 会拒绝提供 @plugin 资源',
  )

  await access(join(entry.prod, 'index.js'))
  await access(join(entry.prod, 'style.css'))
})
