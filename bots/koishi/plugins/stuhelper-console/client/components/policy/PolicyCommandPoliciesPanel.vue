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
          <select v-model="policyForm.commandId" class="sh-select">
            <option v-for="id in supportedCommandIds" :key="id" :value="id">{{ id }}</option>
          </select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">最小 authority</span>
          <input v-model.number="policyForm.minAuthority" class="sh-input" type="number" min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">允许角色</span>
          <input v-model="policyForm.rolesText" class="sh-input" placeholder="reviewer, admin" />
        </label>
      </div>
      <div class="sh-btn-row">
        <button class="sh-btn sh-btn--primary" @click="runTask(submitPolicy)">保存策略</button>
      </div>
    </div>

    <EmptyState
      v-if="commandPolicies.length === 0"
      title="暂无策略"
      body="默认按 authority 生效，需要时再补充角色约束。"
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
