import assert from 'node:assert/strict'
import test from 'node:test'

import { useActionError } from './use-action-error'

test('useActionError stores persistent action error state', () => {
  const panel = useActionError()

  const message = panel.setActionError('保存失败', new Error('backend failed'), '保存失败')

  assert.equal(message, 'backend failed')
  assert.equal(panel.actionErrorTitle.value, '保存失败')
  assert.equal(panel.actionError.value, 'backend failed')

  panel.clearActionError()

  assert.equal(panel.actionErrorTitle.value, '操作失败')
  assert.equal(panel.actionError.value, '')
})

test('useActionError supports custom defaults and optional error callbacks', () => {
  const seen: Array<{ title: string, message: string }> = []
  const panel = useActionError({
    defaultTitle: '导入失败',
    onError: (title, message) => seen.push({ title, message }),
  })

  panel.setActionError('加载失败', undefined, '加载失败')

  assert.equal(panel.actionErrorTitle.value, '加载失败')
  assert.equal(panel.actionError.value, '加载失败')
  assert.deepEqual(seen, [{ title: '加载失败', message: '加载失败' }])

  panel.clearActionError()

  assert.equal(panel.actionErrorTitle.value, '导入失败')
  assert.equal(panel.actionError.value, '')
})
