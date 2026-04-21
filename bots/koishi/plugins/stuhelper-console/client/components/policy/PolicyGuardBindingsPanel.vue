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
          <el-input
            v-model="bindingForm.platform"
            class="sh-control sh-control--mono"
            placeholder="onebot / qq / mock"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">群号</span>
          <el-input v-model="bindingForm.guildId" class="sh-control sh-control--mono" placeholder="群号" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">模板</span>
          <el-select v-model="bindingForm.templateId" class="sh-control" placeholder="选择模板">
            <el-option value="" label="选择模板" />
            <el-option
              v-for="template in templates"
              :key="template.id"
              :value="template.id"
              :label="`${template.name} (${template.id})`"
            />
          </el-select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">备注</span>
          <el-input v-model="bindingForm.note" class="sh-control" placeholder="如 2026 级宿舍群" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">状态</span>
          <el-checkbox v-model="bindingForm.enabled" class="sh-check">绑定启用</el-checkbox>
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button type="primary" class="sh-button sh-button--primary" @click="runTask(submitBinding)">
          保存绑定
        </el-button>
        <span class="sh-field__hint">禁用后会显式停用数据库规则，不回退静态配置。</span>
      </div>
    </div>

    <div class="sh-table-shell">
      <el-table :data="bindings" row-key="id" @row-click="handleRowClick">
        <template #empty>
          <EmptyState title="暂无绑定" body="选择一个群并挂载模板后会显示在这里。" />
        </template>
        <el-table-column prop="platform" label="平台" />
        <el-table-column prop="guildId" label="群" />
        <el-table-column prop="templateId" label="模板" />
        <el-table-column label="状态">
          <template #default="{ row }">
            <SeverityTag
              :label="row.enabled ? '启用' : '停用'"
              :intent="row.enabled ? 'success' : 'muted'"
            />
          </template>
        </el-table-column>
        <el-table-column label="备注">
          <template #default="{ row }">
            {{ row.note || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" align="right">
          <template #default="{ row }">
            <el-button
              class="sh-button sh-button--ghost sh-button--sm"
              @click.stop="loadBinding(row)"
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
import type {
  StuhelperConsoleGuardBinding,
  StuhelperConsoleGuardTemplate,
} from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import SeverityTag from '../SeverityTag.vue'

const props = defineProps<{
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

function handleRowClick(binding: StuhelperConsoleGuardBinding) {
  props.inspectBinding?.(binding)
}
</script>
