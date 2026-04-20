<template>
  <div id="sh-view-gate" class="sh-view" role="tabpanel">
    <header class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">GATE / 认证准入</span>
        <h1 class="sh-view__title">待认证成员处置</h1>
        <p class="sh-view__lead">
          入群准入、超时踢出与观察名单共用一条工作流。踢人与拉黑会统一回流到处置中心复核。
        </p>
      </div>
      <div class="sh-view__toolbar">
        <span class="sh-toolbar__count">待处理 {{ pendingMembers.length }}</span>
        <span class="sh-toolbar__count">已选 {{ selectedGuardIds.length }}</span>
      </div>
    </header>

    <Section
      eyebrow="Bulk action"
      title="批量处置"
      description="选择目标后执行动作；踢人/踢人并拉黑会创建人工复核任务而不直接执行。"
      tone="accent"
    >
      <div class="sh-form-grid">
        <label class="sh-field">
          <span class="sh-field__label">动作</span>
          <select v-model="guardForm.action" class="sh-select">
            <option value="mute">批量禁言</option>
            <option value="unmute">批量解除禁言</option>
            <option value="kick">提交踢出复核</option>
            <option value="set-role">批量设置角色</option>
            <option value="unset-role">批量移除角色</option>
          </select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <input v-model.number="guardForm.seconds" class="sh-input" type="number" min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">角色 ID</span>
          <input v-model="guardForm.roleId" class="sh-input sh-input--mono" placeholder="role_id" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">操作原因</span>
          <input v-model="guardForm.reason" class="sh-input" placeholder="控制台批量操作" />
        </label>
        <label class="sh-check">
          <input v-model="guardForm.permanent" type="checkbox" />
          <span>同时拉黑</span>
        </label>
      </div>
      <div class="sh-btn-row">
        <button
          class="sh-btn sh-btn--primary"
          :disabled="selectedGuardIds.length === 0"
          @click="runTask(submitGuardAction)"
        >
          执行操作
        </button>
        <span v-if="guardForm.action === 'kick'" class="sh-field__hint">
          踢人和踢人并拉黑会先创建人工复核申请，不会直接执行。
        </span>
      </div>
    </Section>

    <Section
      eyebrow="Queue"
      title="待认证成员"
      description="点击成员行打开详情；checkbox 勾选后在上方执行批量动作。"
      :meta="`${pendingMembers.length} 条`"
      flush
    >
      <EmptyState
        v-if="pendingMembers.length === 0"
        title="队列清零"
        body="所有待认证成员都已被处理；后台扫描任务会把新入群且未认证的成员加回这里。"
      />
      <div v-else class="sh-table-shell">
        <table class="sh-table">
          <thead>
            <tr>
              <th style="width: 40px"></th>
              <th>成员</th>
              <th>群</th>
              <th>状态</th>
              <th>截止</th>
              <th>最近错误</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="member in pendingMembers"
              :key="member.id"
              :data-clickable="true"
              :aria-selected="inspector.id === member.id"
              @click="onRowClick(member)"
            >
              <td @click.stop>
                <input
                  v-model="selectedGuardIds"
                  type="checkbox"
                  :value="member.id"
                  aria-label="选中此成员"
                />
              </td>
              <td>
                <div>{{ member.memberName || member.memberId }}</div>
                <div class="sh-table__id">{{ member.memberId }}</div>
              </td>
              <td class="sh-table__mono">{{ member.guildId }}</td>
              <td>
                <SeverityTag
                  :label="member.verificationState"
                  :intent="stateIntent(member.verificationState)"
                />
              </td>
              <td class="sh-table__mono">{{ formatTimestamp(member.deadlineAt) }}</td>
              <td>{{ member.lastError || '—' }}</td>
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
import type { StuhelperConsoleGuardMember } from '../../../src/console-types'
import {
  formatTimestamp,
  type InspectorState,
  type ActionIntent,
} from '../../use-console-page'

const props = defineProps<{
  pendingMembers: readonly StuhelperConsoleGuardMember[]
  guardForm: {
    action: 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role'
    seconds: number
    reason: string
    roleId: string
    permanent: boolean
  }
  inspector: InspectorState
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitGuardAction: () => Promise<unknown>
  inspectMember: (member: StuhelperConsoleGuardMember) => void
}>()

const selectedGuardIds = defineModel<string[]>('selectedGuardIds', { required: true })

function stateIntent(state: string): ActionIntent {
  switch (state) {
    case 'verified':
    case 'released':
      return 'success'
    case 'pending':
      return 'warning'
    case 'overdue':
    case 'kicked':
      return 'danger'
    default:
      return 'neutral'
  }
}

function onRowClick(member: StuhelperConsoleGuardMember) {
  props.inspectMember(member)
}
</script>
