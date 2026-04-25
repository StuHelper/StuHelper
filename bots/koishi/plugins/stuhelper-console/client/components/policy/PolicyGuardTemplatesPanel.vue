<template>
  <WorkspaceSection
    title="群模板"
    description="统一维护入群提醒、禁言时长和超时踢出策略。"
    :meta="`${templates.length} 条`"
    tone="accent"
    flush
  >
    <div class="sh-section__body">
      <div class="sh-form-grid">
        <label class="sh-field">
          <span class="sh-field__label">模板 ID</span>
          <el-input
            v-model="templateForm.id"
            class="sh-control sh-control--mono"
            placeholder="dormitory"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">模板名称</span>
          <el-input v-model="templateForm.name" class="sh-control" placeholder="宿舍群模板" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <el-input-number v-model="templateForm.muteDurationSeconds" class="sh-control" :min="1" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">踢出分钟数</span>
          <el-input-number v-model="templateForm.kickAfterMinutes" class="sh-control" :min="1" />
        </label>
        <label class="sh-field" style="grid-column: span 2">
          <span class="sh-field__label">提醒文案</span>
          <el-input
            v-model="templateForm.reminderTemplate"
            class="sh-control"
            placeholder="请先完成 StuHelper 注册与认证。"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">白名单成员</span>
          <el-input
            v-model="templateForm.exemptUsersText"
            class="sh-control sh-control--mono"
            placeholder="成员 ID，逗号分隔"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">状态</span>
          <el-checkbox v-model="templateForm.enabled" class="sh-check">模板启用</el-checkbox>
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button type="primary" class="sh-button sh-button--primary" @click="runTask(submitTemplate)">
          保存模板
        </el-button>
      </div>
    </div>

    <div class="sh-table-shell">
      <el-table :data="templates" row-key="id" @row-click="handleRowClick">
        <template #empty>
          <EmptyState title="暂无模板" body="先创建一条模板，再挂载到具体群。" />
        </template>
        <el-table-column prop="id" label="ID" />
        <el-table-column prop="name" label="名称" />
        <el-table-column label="禁言">
          <template #default="{ row }">
            {{ row.muteDurationSeconds }} 秒
          </template>
        </el-table-column>
        <el-table-column label="超时">
          <template #default="{ row }">
            {{ row.kickAfterMinutes }} 分钟
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
              @click.stop="loadTemplate(row)"
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
import type { StuhelperConsoleGuardTemplate } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'

const props = defineProps<{
  templates: readonly StuhelperConsoleGuardTemplate[]
  templateForm: {
    id: string
    name: string
    muteDurationSeconds: number
    kickAfterMinutes: number
    reminderTemplate: string
    exemptUsersText: string
    enabled: boolean
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitTemplate: () => Promise<unknown>
  loadTemplate: (template: StuhelperConsoleGuardTemplate) => void
  inspectTemplate?: (template: StuhelperConsoleGuardTemplate) => void
}>()

function handleRowClick(template: StuhelperConsoleGuardTemplate) {
  props.inspectTemplate?.(template)
}
</script>
