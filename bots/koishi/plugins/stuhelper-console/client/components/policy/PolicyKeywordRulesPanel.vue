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
          <el-input
            v-model="ruleForm.id"
            class="sh-control sh-control--mono"
            placeholder="spam-link"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">群号或 *</span>
          <el-input v-model="ruleForm.guildId" class="sh-control sh-control--mono" placeholder="*" />
        </label>
        <label class="sh-field" style="grid-column: span 2">
          <span class="sh-field__label">关键词 / 正则</span>
          <el-input v-model="ruleForm.pattern" class="sh-control" placeholder="输入规则内容" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">匹配模式</span>
          <el-select v-model="ruleForm.matchMode" class="sh-control">
            <el-option value="includes" label="includes" />
            <el-option value="regex" label="regex" />
          </el-select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">动作</span>
          <el-select v-model="ruleForm.action" class="sh-control">
            <el-option value="warn" label="warn" />
            <el-option value="delete" label="delete" />
            <el-option value="mute" label="mute" />
            <el-option value="review" label="review" />
          </el-select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <el-input-number v-model="ruleForm.muteSeconds" class="sh-control" :min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">备注</span>
          <el-input v-model="ruleForm.note" class="sh-control" placeholder="记录规则来源" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">状态</span>
          <el-checkbox v-model="ruleForm.enabled" class="sh-check">规则启用</el-checkbox>
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button type="primary" class="sh-button sh-button--primary" @click="runTask(submitRule)">
          保存规则
        </el-button>
      </div>
    </div>

    <div class="sh-table-shell">
      <el-table :data="keywordRules" row-key="id" @row-click="handleRowClick">
        <template #empty>
          <EmptyState title="暂无规则" body="填写上方表单后保存。" />
        </template>
        <el-table-column label="规则">
          <template #default="{ row }">
            <div>{{ row.id }}</div>
            <div class="sh-table__id">{{ row.matchMode }}</div>
          </template>
        </el-table-column>
        <el-table-column label="匹配">
          <template #default="{ row }">
            <span class="sh-table__mono">{{ row.pattern }}</span>
          </template>
        </el-table-column>
        <el-table-column label="动作">
          <template #default="{ row }">
            <SeverityTag
              :label="describeAction(row.action).label"
              :intent="describeAction(row.action).intent"
            />
          </template>
        </el-table-column>
        <el-table-column label="群">
          <template #default="{ row }">
            <span class="sh-table__mono">{{ row.guildId }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态">
          <template #default="{ row }">
            <SeverityTag
              :label="row.enabled ? '启用' : '停用'"
              :intent="row.enabled ? 'success' : 'muted'"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" align="right">
          <template #default="{ row }">
            <el-button
              class="sh-button sh-button--ghost sh-button--sm"
              @click.stop="loadRule(row)"
            >
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </WorkspaceSection>
</template>

<script setup lang="ts">
import type { StuhelperConsoleKeywordRule } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'
import { describeAction } from '../../formatters'

const props = defineProps<{
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

function handleRowClick(rule: StuhelperConsoleKeywordRule) {
  props.inspectRule?.(rule)
}
</script>
