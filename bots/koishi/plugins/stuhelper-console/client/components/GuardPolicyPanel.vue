<template>
  <div class="sh-split sh-split--7-5">
    <ConsolePanel eyebrow="Policy" title="群模板" description="把提醒文案、禁言时长和超时踢出收敛到一套模板，用于多个群快速复用。" :meta="`${templates.length} 条模板`" tone="accent">
      <div class="sh-form-grid">
        <label class="sh-field"><span class="sh-field__label">模板 ID</span><input v-model="templateForm.id" class="sh-input sh-input--mono" placeholder="dormitory" /></label>
        <label class="sh-field"><span class="sh-field__label">模板名称</span><input v-model="templateForm.name" class="sh-input" placeholder="宿舍群模板" /></label>
        <label class="sh-field"><span class="sh-field__label">禁言秒数</span><input v-model.number="templateForm.muteDurationSeconds" class="sh-input sh-input--mono" type="number" min="1" /></label>
        <label class="sh-field"><span class="sh-field__label">踢出分钟数</span><input v-model.number="templateForm.kickAfterMinutes" class="sh-input sh-input--mono" type="number" min="1" /></label>
        <label class="sh-field"><span class="sh-field__label">提醒文案</span><input v-model="templateForm.reminderTemplate" class="sh-input" placeholder="请先完成 StuHelper 注册与认证。" /></label>
        <label class="sh-field"><span class="sh-field__label">白名单成员</span><input v-model="templateForm.exemptUsersText" class="sh-input sh-input--mono" placeholder="成员 ID，逗号分隔" /></label>
        <label class="sh-check"><input v-model="templateForm.enabled" type="checkbox" /><span>模板启用</span></label>
        <button class="sh-btn sh-btn--primary" @click="runTask(submitTemplate)">保存模板</button>
      </div>
      <div class="sh-table-shell">
        <table class="sh-table">
          <thead><tr><th>ID</th><th>名称</th><th>禁言</th><th>超时</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-if="!templates.length"><td colspan="6" class="sh-table__empty"><strong>暂无模板</strong>先创建第一条群模板。</td></tr>
            <tr v-for="template in templates" :key="template.id" data-clickable="true" @click="inspectTemplate?.(template)">
              <td>{{ template.id }}</td>
              <td>{{ template.name }}</td>
              <td>{{ template.muteDurationSeconds }} 秒</td>
              <td>{{ template.kickAfterMinutes }} 分钟</td>
              <td><SeverityTag :label="template.enabled ? '启用' : '停用'" :intent="template.enabled ? 'success' : 'muted'" /></td>
              <td class="sh-table__actions"><button class="sh-btn sh-btn--ghost sh-btn--sm" @click.stop="loadTemplate(template)">载入</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </ConsolePanel>

    <ConsolePanel eyebrow="Binding" title="群绑定" description="按 platform + guildId 挂载模板。已绑定群优先读取数据库规则，未绑定群才回退静态 guard 配置。" :meta="`${bindings.length} 条绑定`">
      <div class="sh-form-grid">
        <label class="sh-field"><span class="sh-field__label">平台</span><input v-model="bindingForm.platform" class="sh-input sh-input--mono" placeholder="onebot / qq / mock" /></label>
        <label class="sh-field"><span class="sh-field__label">群号</span><input v-model="bindingForm.guildId" class="sh-input sh-input--mono" placeholder="群号" /></label>
        <label class="sh-field"><span class="sh-field__label">模板</span><select v-model="bindingForm.templateId" class="sh-select"><option value="">选择模板</option><option v-for="template in templates" :key="template.id" :value="template.id">{{ template.name }} ({{ template.id }})</option></select></label>
        <label class="sh-field"><span class="sh-field__label">备注</span><input v-model="bindingForm.note" class="sh-input" placeholder="如 2026 级宿舍群" /></label>
        <label class="sh-check"><input v-model="bindingForm.enabled" type="checkbox" /><span>绑定启用</span></label>
        <button class="sh-btn sh-btn--primary" @click="runTask(submitBinding)">保存绑定</button>
      </div>
      <p class="sh-field__hint">数据库里禁用绑定后，该群会显式停用数据库规则，不再回退静态配置。</p>
      <div class="sh-table-shell">
        <table class="sh-table">
          <thead><tr><th>平台</th><th>群</th><th>模板</th><th>状态</th><th>备注</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-if="!bindings.length"><td colspan="6" class="sh-table__empty"><strong>暂无绑定</strong>选一个群并挂载模板后才会出现在这里。</td></tr>
            <tr v-for="binding in bindings" :key="binding.id" data-clickable="true" @click="inspectBinding?.(binding)">
              <td>{{ binding.platform }}</td>
              <td>{{ binding.guildId }}</td>
              <td>{{ binding.templateId }}</td>
              <td><SeverityTag :label="binding.enabled ? '启用' : '停用'" :intent="binding.enabled ? 'success' : 'muted'" /></td>
              <td>{{ binding.note || '无' }}</td>
              <td class="sh-table__actions"><button class="sh-btn sh-btn--ghost sh-btn--sm" @click.stop="loadBinding(binding)">载入</button></td>
            </tr>
          </tbody>
        </table>
      </div>
    </ConsolePanel>
  </div>
</template>

<script setup lang="ts">
import ConsolePanel from './ConsolePanel.vue'
import SeverityTag from './SeverityTag.vue'

import type { StuhelperConsoleGuardBinding, StuhelperConsoleGuardTemplate } from '../../src/console-types'

defineProps<{
  templates: StuhelperConsoleGuardTemplate[]
  bindings: StuhelperConsoleGuardBinding[]
  templateForm: {
    id: string
    name: string
    muteDurationSeconds: number
    kickAfterMinutes: number
    reminderTemplate: string
    exemptUsersText: string
    enabled: boolean
  }
  bindingForm: {
    platform: string
    guildId: string
    templateId: string
    enabled: boolean
    note: string
  }
  runTask: (task: () => Promise<unknown>) => Promise<void>
  submitTemplate: () => Promise<unknown>
  submitBinding: () => Promise<unknown>
  loadTemplate: (template: StuhelperConsoleGuardTemplate) => void
  loadBinding: (binding: StuhelperConsoleGuardBinding) => void
  inspectTemplate?: (template: StuhelperConsoleGuardTemplate) => void
  inspectBinding?: (binding: StuhelperConsoleGuardBinding) => void
}>()
</script>
