import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

async function readSource(relativePath: string) {
  return readFile(new URL(relativePath, import.meta.url), 'utf8')
}

test('控制台 listener 事件签名对关键词规则使用输入类型', async () => {
  const source = await readSource('./index.ts')

  assert.match(source, /StuhelperKeywordRuleInput/)
  assert.match(source, /'stuhelper-console\/save-keyword-rule'\(input: StuhelperKeywordRuleInput\)/)
  assert.doesNotMatch(source, /'stuhelper-console\/save-keyword-rule'\(input: StuhelperConsoleKeywordRule\)/)
})
