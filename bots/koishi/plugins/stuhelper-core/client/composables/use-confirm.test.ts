import assert from 'node:assert/strict'
import test from 'node:test'

import { useConfirm } from './use-confirm'

test('useConfirm resolves accepted and cancelled dialogs', async () => {
  const dialog = useConfirm()

  const accepted = dialog.confirm({
    title: '删除角色',
    message: '确定要删除角色吗？',
    tone: 'danger',
  })

  assert.equal(dialog.state.value.open, true)
  assert.equal(dialog.state.value.title, '删除角色')
  assert.equal(dialog.state.value.tone, 'danger')

  dialog.accept()

  assert.equal(await accepted, true)
  assert.equal(dialog.state.value.open, false)

  const cancelled = dialog.confirm({
    title: '放弃更改',
    message: '确定要放弃更改吗？',
  })

  dialog.cancel()

  assert.equal(await cancelled, false)
  assert.equal(dialog.state.value.open, false)
})

test('useConfirm rejects overlapping dialogs without throwing', async () => {
  const dialog = useConfirm()

  const first = dialog.confirm({
    title: '第一个确认',
    message: '第一个确认正在等待处理。',
  })

  const duplicate = await dialog.confirm({
    title: '第二个确认',
    message: '第二个确认不应该覆盖第一个。',
  })

  assert.equal(duplicate, false)
  assert.equal(dialog.state.value.open, true)
  assert.equal(dialog.state.value.title, '第一个确认')

  dialog.cancel()

  assert.equal(await first, false)
  assert.equal(dialog.state.value.open, false)
})
