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
          <input v-model="templateForm.id" class="sh-input sh-input--mono" placeholder="dormitory" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">模板名称</span>
          <input v-model="templateForm.name" class="sh-input" placeholder="宿舍群模板" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <input v-model.number="templateForm.muteDurationSeconds" class="sh-input" type="number" min="1" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">踢出分钟数</span>
          <input v-model.number="templateForm.kickAfterMinutes" class="sh-input" type="number" min="1" />
        </label>
        <label class="sh-field" style="grid-column: span 2">
          <span class="sh-field__label">提醒文案</span>
          <input v-model="templateForm.reminderTemplate" class="sh-input" placeholder="请先完成 StuHelper 注册与认证。" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">白名单成员</span>
          <input v-model="templateForm.exemptUsersText" class="sh-input sh-input--mono" placeholder="成员 ID，逗号分隔" />
        </label>
        <label class="sh-check">
          <input v-model="templateForm.enabled" type="checkbox" />
          <span>模板启用</span>
        </label>
      </div>
      <div class="sh-btn-row">
        <button class="sh-btn sh-btn--primary" @click="runTask(submitTemplate)">保存模板</button>
      </div>
    </div>

    <EmptyState
      v-if="templates.length === 0"
      title="暂无模板"
      body="先创建一条模板，再挂载到具体群。"
    />
    <div v-else class="sh-table-shell">
      <table class="sh-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>名称</th>
            <th>禁言</th>
            <th>超时</th>
            <th>状态</th>
            <th style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="template in templates"
            :key="template.id"
            data-clickable="true"
            @click="inspectTemplate?.(template)"
          >
            <td>{{ template.id }}</td>
            <td>{{ template.name }}</td>
            <td>{{ template.muteDurationSeconds }} 秒</td>
            <td>{{ template.kickAfterMinutes }} 分钟</td>
            <td>
              <SeverityTag
                :label="template.enabled ? '启用' : '停用'"
                :intent="template.enabled ? 'success' : 'muted'"
              />
            </td>
            <td class="sh-table__actions">
              <button class="sh-btn sh-btn--sm sh-btn--ghost" @click.stop="loadTemplate(template)">
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
import type { StuhelperConsoleGuardTemplate } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'

defineProps<{
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
</script>
