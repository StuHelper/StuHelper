<template>
  <WorkspaceSection
    title="命令权限"
    description="限制关键命令的 authority 下限和角色范围。"
    :meta="`${commandPolicies.length} 条`"
    flush
  >
    <div class="sh-section__body">
      <div class="sh-form-grid sh-form-grid--narrow">
        <label class="sh-field">
          <span class="sh-field__label">命令</span>
          <el-select v-model="policyForm.commandId" class="sh-control" placeholder="选择命令">
            <el-option v-for="id in supportedCommandIds" :key="id" :value="id" :label="id" />
          </el-select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">最小 authority</span>
          <el-input-number v-model="policyForm.minAuthority" class="sh-control" :min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">允许角色</span>
          <el-input
            v-model="policyForm.rolesText"
            class="sh-control"
            placeholder="reviewer, admin"
          />
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button type="primary" class="sh-button sh-button--primary" @click="runTask(submitPolicy)">
          保存策略
        </el-button>
      </div>
    </div>

    <div class="sh-table-shell">
      <el-table :data="commandPolicies" row-key="commandId">
        <template #empty>
          <EmptyState
            title="暂无策略"
            body="默认按 authority 生效，需要时再补充角色约束。"
          />
        </template>
        <el-table-column label="命令">
          <template #default="{ row }">
            <span class="sh-table__mono">{{ row.commandId }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="minAuthority" label="Authority" />
        <el-table-column label="角色">
          <template #default="{ row }">
            {{ row.roles.join(', ') || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" align="right">
          <template #default="{ row }">
            <el-button class="sh-button sh-button--ghost sh-button--sm" @click="loadPolicy(row)">
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </WorkspaceSection>
</template>

<script setup lang="ts">
import type { StuhelperConsoleCommandPolicy } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'

defineProps<{
  commandPolicies: readonly StuhelperConsoleCommandPolicy[]
  supportedCommandIds: readonly string[]
  policyForm: {
    commandId: string
    minAuthority: number
    rolesText: string
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitPolicy: () => Promise<unknown>
  loadPolicy: (policy: StuhelperConsoleCommandPolicy) => void
}>()
</script>
