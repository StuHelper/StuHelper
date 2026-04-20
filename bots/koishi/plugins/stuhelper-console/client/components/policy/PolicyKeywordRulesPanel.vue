<template>
  <WorkspaceSection
    title="关键词规则"
    description="集中维护触发词、匹配模式和处置动作。"
    :meta="`${keywordRules.length} 条`"
    flush
  >
    <div class="sh-section__body">
      <div class="sh-form-grid">
        <label class="sh-field">
          <span class="sh-field__label">规则 ID</span>
          <input v-model="ruleForm.id" class="sh-input sh-input--mono" placeholder="spam-link" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">群号或 *</span>
          <input v-model="ruleForm.guildId" class="sh-input sh-input--mono" placeholder="*" />
        </label>
        <label class="sh-field" style="grid-column: span 2">
          <span class="sh-field__label">关键词 / 正则</span>
          <input v-model="ruleForm.pattern" class="sh-input" placeholder="输入规则内容" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">匹配模式</span>
          <select v-model="ruleForm.matchMode" class="sh-select">
            <option value="includes">includes</option>
            <option value="regex">regex</option>
          </select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">动作</span>
          <select v-model="ruleForm.action" class="sh-select">
            <option value="warn">warn</option>
            <option value="delete">delete</option>
            <option value="mute">mute</option>
            <option value="review">review</option>
          </select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <input v-model.number="ruleForm.muteSeconds" class="sh-input" type="number" min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">备注</span>
          <input v-model="ruleForm.note" class="sh-input" placeholder="记录规则来源" />
        </label>
        <label class="sh-check">
          <input v-model="ruleForm.enabled" type="checkbox" />
          <span>规则启用</span>
        </label>
      </div>
      <div class="sh-btn-row">
        <button class="sh-btn sh-btn--primary" @click="runTask(submitRule)">保存规则</button>
      </div>
    </div>

    <EmptyState
      v-if="keywordRules.length === 0"
      title="暂无规则"
      body="填写上方表单后保存。"
    />
    <div v-else class="sh-table-shell">
      <table class="sh-table">
        <thead>
          <tr>
            <th>规则</th>
            <th>匹配</th>
            <th>动作</th>
            <th>群</th>
            <th>状态</th>
            <th style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="rule in keywordRules"
            :key="rule.id"
            data-clickable="true"
            @click="inspectRule?.(rule)"
          >
            <td>
              <div>{{ rule.id }}</div>
              <div class="sh-table__id">{{ rule.matchMode }}</div>
            </td>
            <td class="sh-table__mono">{{ rule.pattern }}</td>
            <td>
              <SeverityTag
                :label="describeAction(rule.action).label"
                :intent="describeAction(rule.action).intent"
              />
            </td>
            <td class="sh-table__mono">{{ rule.guildId }}</td>
            <td>
              <SeverityTag
                :label="rule.enabled ? '启用' : '停用'"
                :intent="rule.enabled ? 'success' : 'muted'"
              />
            </td>
            <td class="sh-table__actions">
              <button class="sh-btn sh-btn--sm sh-btn--ghost" @click.stop="loadRule(rule)">
                编辑
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </WorkspaceSection>
</template>

<script setup lang="ts">
import type { StuhelperConsoleKeywordRule } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'
import { describeAction } from '../../use-console-page'

defineProps<{
  keywordRules: readonly StuhelperConsoleKeywordRule[]
  ruleForm: {
    id: string
    guildId: string
    pattern: string
    matchMode: 'includes' | 'regex'
    action: 'warn' | 'delete' | 'mute' | 'review'
    enabled: boolean
    muteSeconds: number
    note: string
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitRule: () => Promise<unknown>
  inspectRule?: (rule: StuhelperConsoleKeywordRule) => void
  loadRule: (rule: StuhelperConsoleKeywordRule) => void
}>()
</script>
