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
          <input v-model="roleForm.guildId" class="sh-input sh-input--mono" placeholder="guild-id" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">成员 ID</span>
          <input v-model="roleForm.memberId" class="sh-input sh-input--mono" placeholder="member-id" />
        </label>
        <label class="sh-field">
          <span class="sh-field__label">角色</span>
          <input v-model="roleForm.rolesText" class="sh-input" placeholder="admin, reviewer" />
        </label>
      </div>
      <div class="sh-btn-row">
        <button class="sh-btn sh-btn--primary" @click="runTask(submitRoles)">保存角色</button>
      </div>
    </div>

    <EmptyState
      v-if="memberRoles.length === 0"
      title="暂无成员角色"
      body="需要时再绑定角色即可。"
    />
    <div v-else class="sh-table-shell">
      <table class="sh-table">
        <thead>
          <tr>
            <th>群</th>
            <th>成员</th>
            <th>角色</th>
            <th style="text-align: right">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="entry in memberRoles" :key="entry.id">
            <td>{{ entry.guildId }}</td>
            <td>{{ entry.memberId }}</td>
            <td>{{ entry.roles.join(', ') || '—' }}</td>
            <td class="sh-table__actions">
              <button class="sh-btn sh-btn--sm sh-btn--ghost" @click="loadMemberRoles(entry)">
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
