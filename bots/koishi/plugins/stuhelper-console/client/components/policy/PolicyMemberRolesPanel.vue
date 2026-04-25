<template>
  <WorkspaceSection
    title="成员角色"
    description="维护命令放行、协作分工和复核责任。"
    :meta="`${memberRoles.length} 条`"
    flush
  >
    <div class="sh-section__body">
      <div class="sh-form-grid sh-form-grid--narrow">
        <label class="sh-field">
          <span class="sh-field__label">群号</span>
          <el-input
            v-model="roleForm.guildId"
            class="sh-control sh-control--mono"
            placeholder="guild-id"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">成员 ID</span>
          <el-input
            v-model="roleForm.memberId"
            class="sh-control sh-control--mono"
            placeholder="member-id"
          />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">角色</span>
          <el-input
            v-model="roleForm.rolesText"
            class="sh-control"
            placeholder="admin, reviewer"
          />
        </label>
      </div>
      <div class="sh-btn-row">
        <el-button type="primary" class="sh-button sh-button--primary" @click="runTask(submitRoles)">
          保存角色
        </el-button>
      </div>
    </div>

    <div class="sh-table-shell">
      <el-table :data="memberRoles" row-key="id">
        <template #empty>
          <EmptyState title="暂无成员角色" body="需要时再绑定角色即可。" />
        </template>
        <el-table-column prop="guildId" label="群" />
        <el-table-column prop="memberId" label="成员" />
        <el-table-column label="角色">
          <template #default="{ row }">
            {{ row.roles.join(', ') || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="操作" align="right">
          <template #default="{ row }">
            <el-button class="sh-button sh-button--ghost sh-button--sm" @click="loadMemberRoles(row)">
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </WorkspaceSection>
</template>

<script setup lang="ts">
import type { StuhelperConsoleMemberRole } from '../../../src/console-types'
import EmptyState from '../EmptyState.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'

defineProps<{
  memberRoles: readonly StuhelperConsoleMemberRole[]
  roleForm: {
    guildId: string
    memberId: string
    rolesText: string
  }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitRoles: () => Promise<unknown>
  loadMemberRoles: (entry: StuhelperConsoleMemberRole) => void
}>()
</script>
