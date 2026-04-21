import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

async function readSource(relativePath: string) {
  return readFile(new URL(relativePath, import.meta.url), 'utf8')
}

test('控制台关键页面改用 Element Plus 基础控件', async () => {
  const [
    page,
    drawer,
    queueTable,
    dashboardTodoTable,
    identityQueue,
    reviewQueue,
    keywordRules,
    commandPolicies,
    guardTemplates,
    guardBindings,
  ] =
    await Promise.all([
      readSource('./page.vue'),
      readSource('./components/Drawer.vue'),
      readSource('./components/queue/QueueTable.vue'),
      readSource('./components/dashboard/DashboardTodoTable.vue'),
      readSource('./components/queue/IdentityQueuePage.vue'),
      readSource('./components/queue/ReviewQueuePage.vue'),
      readSource('./components/policy/PolicyKeywordRulesPanel.vue'),
      readSource('./components/policy/PolicyCommandPoliciesPanel.vue'),
      readSource('./components/policy/PolicyGuardTemplatesPanel.vue'),
      readSource('./components/policy/PolicyGuardBindingsPanel.vue'),
    ])

  assert.match(page, /<el-button[\s>]/)
  assert.match(drawer, /<el-button[\s>]/)
  assert.match(queueTable, /<el-table[\s>]/)
  assert.doesNotMatch(queueTable, /<table class="sh-table"/)
  assert.match(dashboardTodoTable, /<el-table[\s>]/)
  assert.doesNotMatch(dashboardTodoTable, /<table class="sh-table"/)

  assert.match(identityQueue, /<el-select[\s>]/)
  assert.match(identityQueue, /<el-input[\s>]/)
  assert.match(identityQueue, /<el-checkbox[\s>]/)
  assert.doesNotMatch(identityQueue, /class="sh-btn(?:\s|")/)

  assert.match(reviewQueue, /<el-input[\s>]/)
  assert.doesNotMatch(reviewQueue, /class="sh-btn(?:\s|")/)

  assert.match(keywordRules, /<el-input[\s>]/)
  assert.match(keywordRules, /<el-table[\s>]/)
  assert.doesNotMatch(keywordRules, /<table class="sh-table"/)

  assert.match(commandPolicies, /<el-select[\s>]/)
  assert.match(commandPolicies, /<el-table[\s>]/)
  assert.doesNotMatch(commandPolicies, /<table class="sh-table"/)

  assert.match(guardTemplates, /<el-input[\s>]/)
  assert.match(guardTemplates, /<el-table[\s>]/)
  assert.match(guardBindings, /<el-select[\s>]/)
  assert.match(guardBindings, /<el-table[\s>]/)
})
