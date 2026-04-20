<template>
  <div
    :id="props.showHeader ? 'sh-view-rules' : undefined"
    :class="props.showHeader ? 'sh-view' : 'sh-rules-view'"
    :role="props.showHeader ? 'tabpanel' : undefined"
  >
    <header v-if="props.showHeader" class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">RULES / 通用群规</span>
        <h1 class="sh-view__title">关键词与命令权限</h1>
        <p class="sh-view__lead">
          把高频文本治理收敛成规则表；命令权限表则是控制谁能跑关键指令的第二道闸。
        </p>
      </div>
    </header>

    <Section
      v-if="props.mode !== 'command-policies'"
      eyebrow="Keyword rules"
      title="关键词规则"
      description="左侧编辑、下方列表。命中后的动作可以是警告、撤回、禁言或转人工复核。"
      :meta="`${keywordRules.length} 条规则`"
      flush
    >
      <div class="sh-section__body">
        <div class="sh-form-grid">
          <label class="sh-field">
            <span class="sh-field__label">规则 ID</span>
            <input
              v-model="ruleForm.id"
              class="sh-input sh-input--mono"
              placeholder="spam-link"
            />
          </label>
          <label class="sh-field">
            <span class="sh-field__label">群号或 *</span>
            <input v-model="ruleForm.guildId" class="sh-input sh-input--mono" placeholder="*" />
          </label>
          <label class="sh-field" style="grid-column: span 2">
            <span class="sh-field__label">关键词 / 正则</span>
            <input
              v-model="ruleForm.pattern"
              class="sh-input sh-input--mono"
              placeholder="输入规则内容"
            />
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
            <input
              v-model.number="ruleForm.muteSeconds"
              class="sh-input"
              type="number"
              min="0"
            />
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
          <button class="sh-btn sh-btn--primary" @click="runTask(submitRule)">
            保存规则
          </button>
        </div>
      </div>

      <EmptyState
        v-if="keywordRules.length === 0"
        title="暂无规则"
        body="在上方表单填写规则内容，保存后会写入 SQLite 并立即生效。"
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
    </Section>

    <Section
      v-if="props.mode !== 'keyword-rules'"
      eyebrow="Command policies"
      title="命令权限策略"
      description="为关键指令设置 authority 下限与角色白名单，避免命令入口失控。"
      :meta="`${commandPolicies.length} 条策略`"
      flush
    >
      <div class="sh-section__body">
        <div class="sh-form-grid">
          <label class="sh-field">
            <span class="sh-field__label">命令</span>
            <select v-model="policyForm.commandId" class="sh-select">
              <option v-for="id in supportedCommandIds" :key="id" :value="id">{{ id }}</option>
            </select>
          </label>
          <label class="sh-field">
            <span class="sh-field__label">最小 authority</span>
            <input
              v-model.number="policyForm.minAuthority"
              class="sh-input"
              type="number"
              min="0"
            />
          </label>
          <label class="sh-field">
            <span class="sh-field__label">允许角色</span>
            <input
              v-model="policyForm.rolesText"
              class="sh-input"
              placeholder="reviewer, admin"
            />
          </label>
        </div>
        <div class="sh-btn-row">
          <button class="sh-btn sh-btn--primary" @click="runTask(submitPolicy)">保存策略</button>
        </div>
      </div>

      <EmptyState
        v-if="commandPolicies.length === 0"
        title="暂无策略"
        body="默认按命令的 authority 值判断；需要限定到角色时在上面添加。"
      />
      <div v-else class="sh-table-shell">
        <table class="sh-table">
          <thead>
            <tr>
              <th>命令</th>
              <th>Authority</th>
              <th>角色</th>
              <th style="text-align: right">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="policy in commandPolicies" :key="policy.commandId">
              <td class="sh-table__mono">{{ policy.commandId }}</td>
              <td class="sh-num">{{ policy.minAuthority }}</td>
              <td>{{ policy.roles.join(', ') || '—' }}</td>
              <td class="sh-table__actions">
                <button class="sh-btn sh-btn--sm sh-btn--ghost" @click="loadPolicy(policy)">
                  编辑
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </Section>
  </div>
</template>

<script setup lang="ts">
import Section from '../ConsolePanel.vue'
import SeverityTag from '../SeverityTag.vue'
import EmptyState from '../EmptyState.vue'
import type {
  StuhelperConsoleCommandPolicy,
  StuhelperConsoleKeywordRule,
} from '../../../src/console-types'
import { describeAction } from '../../use-console-page'

const props = withDefaults(defineProps<{
  keywordRules: readonly StuhelperConsoleKeywordRule[]
  commandPolicies: readonly StuhelperConsoleCommandPolicy[]
  supportedCommandIds: readonly string[]
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
  policyForm: { commandId: string; minAuthority: number; rolesText: string }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitRule: () => Promise<unknown>
  submitPolicy: () => Promise<unknown>
  inspectRule?: (rule: StuhelperConsoleKeywordRule) => void
  loadRule: (rule: StuhelperConsoleKeywordRule) => void
  loadPolicy: (policy: StuhelperConsoleCommandPolicy) => void
  mode?: 'keyword-rules' | 'command-policies' | 'all'
  showHeader?: boolean
}>(), {
  mode: 'all',
  showHeader: true,
})
</script>

<style scoped>
.sh-rules-view {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-4);
}
</style>
