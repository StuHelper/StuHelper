<template>
  <WorkspaceSection
    title="群绑定"
    description="按平台和群号挂载模板，显式控制规则生效范围。"
    :meta="`${bindings.length} 条`"
    flush
  >
    <div class="sh-section__body">
      <div class="sh-form-grid sh-form-grid--narrow">
        <label class="sh-field">
          <span class="sh-field__label">平台</span>
          <input v-model="bindingForm.platform" class="sh-input sh-input--mono" placeholder="onebot / qq / mock" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">群号</span>
          <input v-model="bindingForm.guildId" class="sh-input sh-input--mono" placeholder="群号" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">模板</span>
          <select v-model="bindingForm.templateId" class="sh-select">
            <option value="">选择模板</option>
            <option v-for="template in templates" :key="template.id" :value="template.id">
              {{ template.name }} ({{ template.id }})
            </option>
          </select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">备注</span>
          <input v-model="bindingForm.note" class="sh-input" placeholder="如 2026 级宿舍群" />
        </label>
        <label class="sh-check">
          <input v-model="bindingForm.enabled" type="checkbox" />
          <span>绑定启用</span>
        </label>
      </div>
      <div class="sh-btn-row">
        <button class="sh-btn sh-btn--primary" @click="runTask(submitBinding)">保存绑定</button>
        <span class="sh-field__hint">禁用后会显式停用数据库规则，不回退静态配置。</span>
      </div>
    </div>

    <EmptyState
      v-if="bindings.length === 0"
      title="暂无绑定"
      body="选择一个群并挂载模板后会显示在这里。"
    />
    <div v-else class="sh-table-shell">
      <table class="sh-table">
        <thead>
          <tr>
            <th>平台</th>
            <th>群</th>
            <th>模板</th>
            <th>状态</th>
            <th>备注</th>
            <th style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="binding in bindings"
            :key="binding.id"
            data-clickable="true"
            @click="inspectBinding?.(binding)"
          >
            <td>{{ binding.platform }}</td>
            <td>{{ binding.guildId }}</td>
            <td>{{ binding.templateId }}</td>
            <td>
              <SeverityTag
                :label="binding.enabled ? '启用' : '停用'"
                :intent="binding.enabled ? 'success' : 'muted'"
              />
            </td>
            <td>{{ binding.note || '—' }}</td>
            <td class="sh-table__actions">
              <button class="sh-btn sh-btn--sm sh-btn--ghost" @click.stop="loadBinding(binding)">
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
import type {
  StuhelperConsoleGuardBinding,
  StuhelperConsoleGuardTemplate,
} from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'

defineProps<{
  templates: readonly StuhelperConsoleGuardTemplate[]
  bindings: readonly StuhelperConsoleGuardBinding[]
  bindingForm: {
    platform: string
    guildId: string
    templateId: string
    enabled: boolean
    note: string
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitBinding: () => Promise<unknown>
  loadBinding: (binding: StuhelperConsoleGuardBinding) => void
  inspectBinding?: (binding: StuhelperConsoleGuardBinding) => void
}>()
</script>
