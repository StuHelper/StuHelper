<template>
  <div id="sh-view-identity" class="sh-view" role="tabpanel">
    <WorkspaceSection
      title="批量处置"
      description="选择目标后执行动作；踢人或踢人并拉黑会创建人工复核任务而不直接执行。"
      :meta="selectedGuardIds.length ? `已选 ${selectedGuardIds.length}` : '未选择成员'"
      tone="accent"
    >
      <div class="sh-form-grid">
        <label class="sh-field">
          <span class="sh-field__label">动作</span>
          <el-select v-model="guardForm.action" class="sh-control" placeholder="选择动作">
            <el-option value="mute" label="批量禁言" />
            <el-option value="unmute" label="批量解除禁言" />
            <el-option value="kick" label="提交踢出复核" />
            <el-option value="set-role" label="批量设置角色" />
            <el-option value="unset-role" label="批量移除角色" />
          </el-select>
        </label>
        <label class="sh-field">
          <span class="sh-field__label">禁言秒数</span>
          <el-input-number v-model="guardForm.seconds" class="sh-control" :min="0" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">角色 ID</span>
          <el-input
            v-model="guardForm.roleId"
            class="sh-control sh-control--mono"
            placeholder="role_id"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">操作原因</span>
          <el-input
            v-model="guardForm.reason"
            class="sh-control"
            placeholder="控制台批量操作"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">附加动作</span>
          <el-checkbox v-model="guardForm.permanent" class="sh-check">同时拉黑</el-checkbox>
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button
          type="primary"
          class="sh-button sh-button--primary"
          :disabled="selectedGuardIds.length === 0 || loading"
          @click="runTask(submitGuardAction)"
        >
          执行操作
        </el-button>
        <span v-if="guardForm.action === 'kick'" class="sh-field__hint">
          踢人和踢人并拉黑会先创建人工复核申请，不会直接执行。
        </span>
      </div>
    </WorkspaceSection>

    <WorkspaceSection
      title="待认证成员"
      description="点击成员行打开详情抽屉；勾选后在上方执行批量动作。"
      :meta="`${filteredMembers.length} / ${pendingMembers.length}`"
    >
      <QueueToolbar :stats="queueStats">
        <label class="sh-field sh-identity-queue__search">
          <span class="sh-field__label">检索</span>
          <el-input v-model="search" class="sh-control" placeholder="成员 / 群号 / 错误信息" />
        </label>
      </QueueToolbar>

      <QueueTable
        :columns="memberColumns"
        :rows="memberRows"
        :selected-id="selectedId"
        empty-title="队列清零"
        empty-body="所有待认证成员都已被处理；后台扫描任务会把新入群且未认证的成员加回这里。"
        @select="handleMemberSelect"
      >
        <template #cell-select="{ row }">
          <el-checkbox
            :model-value="isSelected(row.id)"
            aria-label="选中此成员"
            @click.stop
            @change="(value) => toggleSelectedGuardId(row.id, value)"
          />
        </template>
      </QueueTable>
    </WorkspaceSection>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import type { StuhelperConsoleGuardMember } from '../../../src/console-types'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import QueueTable from './QueueTable.vue'
import QueueToolbar from './QueueToolbar.vue'
import {
  describeVerificationState,
  formatTimestamp,
  type ActionIntent,
} from '../../formatters'

const MEMBER_COLUMNS = [
  { key: 'select', label: '', width: '40px' },
  { key: 'member', label: '成员' },
  { key: 'guild', label: '群' },
  { key: 'state', label: '状态' },
  { key: 'deadline', label: '截止' },
  { key: 'error', label: '最近错误' },
] as const

const props = defineProps<{
  pendingMembers: readonly StuhelperConsoleGuardMember[]
  selectedId: string
  loading: boolean
  guardForm: {
    action: 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role'
    seconds: number
    reason: string
    roleId: string
    permanent: boolean
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitGuardAction: () => Promise<unknown>
  inspectMember: (member: StuhelperConsoleGuardMember) => void
}>()

const selectedGuardIds = defineModel<string[]>('selectedGuardIds', { required: true })
const search = ref('')

const filteredMembers = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return props.pendingMembers
  return props.pendingMembers.filter((member) =>
    [member.memberName, member.memberId, member.guildId, member.lastError]
      .filter(Boolean)
      .some((field) => String(field).toLowerCase().includes(keyword)),
  )
})

const queueStats = computed(() => [
  { label: '待处理', value: props.pendingMembers.length },
  { label: '当前过滤', value: filteredMembers.value.length },
  { label: '已选', value: selectedGuardIds.value.length },
])

const memberColumns = MEMBER_COLUMNS

const memberRows = computed(() =>
  filteredMembers.value.map((member) => ({
    id: member.id,
    cells: {
      select: '',
      member: {
        text: member.memberName || member.memberId,
        secondary: member.memberId,
      },
      guild: {
        text: member.guildId,
        mono: true,
      },
      state: {
        text: describeVerificationState(member.verificationState),
        tone: stateIntent(member.verificationState),
      },
      deadline: {
        text: formatTimestamp(member.deadlineAt),
        mono: true,
      },
      error: member.lastError || '—',
    },
  })),
)

function stateIntent(state: StuhelperConsoleGuardMember['verificationState']): ActionIntent {
  switch (state) {
    case 'verified':
      return 'success'
    case 'bound_unverified':
      return 'warning'
    case 'unbound':
      return 'danger'
  }
}

function handleMemberSelect(memberId: string) {
  const member = props.pendingMembers.find((item) => item.id === memberId)
  if (!member) return
  props.inspectMember(member)
}

function isSelected(memberId: string) {
  return selectedGuardIds.value.includes(memberId)
}

function toggleSelectedGuardId(memberId: string, checked: string | number | boolean) {
  const nextIds = selectedGuardIds.value.filter((id) => id !== memberId)
  if (checked) {
    selectedGuardIds.value = [...nextIds, memberId]
    return
  }
  selectedGuardIds.value = nextIds
}
</script>

<style scoped>
.sh-identity-queue__search {
  min-width: 220px;
  flex: 0 1 280px;
}
</style>
