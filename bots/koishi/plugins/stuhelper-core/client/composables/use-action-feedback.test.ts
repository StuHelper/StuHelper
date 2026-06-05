import assert from 'node:assert/strict'
import test from 'node:test'

import { useActionFeedback } from './use-action-feedback'

test('useActionFeedback keeps persistent action errors separate from transient notices', () => {
  const feedback = useActionFeedback()

  feedback.setActionError('保存失败', new Error('backend failed'), '保存失败')

  assert.equal(feedback.actionErrorTitle.value, '保存失败')
  assert.equal(feedback.actionError.value, 'backend failed')
  assert.equal(feedback.notices.value.length, 1)
  assert.equal(feedback.notices.value[0].kind, 'error')
  assert.equal(feedback.notices.value[0].title, '保存失败')
  assert.equal(feedback.notices.value[0].message, 'backend failed')

  feedback.clearActionError()

  assert.equal(feedback.actionErrorTitle.value, '操作失败')
  assert.equal(feedback.actionError.value, '')
  assert.equal(feedback.notices.value.length, 1)
})

test('useActionFeedback normalizes unknown causes with fallback text', () => {
  const feedback = useActionFeedback()

  assert.equal(feedback.errorMessage(undefined, '默认错误'), '默认错误')
  assert.equal(feedback.errorMessage(null, '默认错误'), '默认错误')
  assert.equal(feedback.errorMessage('  ', '默认错误'), '默认错误')
  assert.equal(feedback.errorMessage({ code: 'E_TEST' }, '默认错误'), '[object Object]')
})
