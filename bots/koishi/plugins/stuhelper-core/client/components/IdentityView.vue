<template>
  <div class="sh-view">
    <header class="sh-workspace-head">
      <div class="sh-workspace-head__copy">
        <h1 class="sh-workspace-head__title">身份认证</h1>
        <p class="sh-workspace-head__description">
          群优先视角 — 按群锁定受限成员,右侧看跨群绑定与认证上下文。
        </p>
        <div class="sh-workspace-head__meta" v-if="data">
          <span class="sh-meta-chip sh-mono">
            更新 · {{ formatTimestamp(data.generatedAt) }}
          </span>
        </div>
      </div>
      <div class="sh-workspace-head__actions">
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="loading"
          @click="loadData"
        >
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
      </div>
    </header>

    <EmptyState
      v-if="error"
      title="加载失败"
      :body="error"
      tone="error"
    />
    <ConsolePageSkeleton v-else-if="loading && !data" />

    <template v-else-if="data">
      <section class="sh-dashboard-metrics">
        <article
          v-for="(metric, index) in metrics"
          :key="metric.label"
          class="sh-stat sh-dashboard-metric"
          :class="metricIntent(index, metric.value)"
        >
          <span class="sh-stat__label">{{ metric.label }}</span>
          <span class="sh-stat__value sh-num">{{ metric.value }}</span>
          <span class="sh-stat__note">{{ metric.note }}</span>
        </article>
      </section>

      <section class="sh-toolbar sh-identity-toolbar">
        <label class="sh-field sh-identity-toolbar__field">
          <span class="sh-field__label">群组</span>
          <el-select
            v-model="selectedGuildId"
            class="sh-control"
            placeholder="全部群组"
            clearable
            @change="syncSelection()"
          >
            <el-option label="全部群组" value="" />
            <el-option
              v-for="group in data.groups"
              :key="group.guildId"
              :label="`${group.guildId} · ${group.pendingCount} 待处理`"
              :value="group.guildId"
            />
          </el-select>
        </label>
        <label class="sh-field sh-identity-toolbar__field sh-identity-toolbar__field--wide">
          <span class="sh-field__label">检索</span>
          <el-input
            v-model.trim="keyword"
            class="sh-control"
            placeholder="成员 ID 或昵称"
            @input="syncSelection()"
          />
        </label>
      </section>

      <div class="sh-split sh-split--7-5">
        <WorkspaceSection
          title="当前受限成员"
          description="展示仍在限制中的成员。"
          :meta="`${filteredMembers.length} 条`"
          flush
        >
          <EmptyState
            v-if="filteredMembers.length === 0"
            title="没有匹配的成员"
            body="调整群组或检索关键词,或在 Dashboard 查看最近变更。"
          />
          <div v-else class="sh-lane">
            <button
              v-for="member in filteredMembers"
              :key="member.id"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{ 'sh-lane__row--active': selectedMember?.id === member.id }"
              @click="selectMember(member.id)"
            >
              <span class="sh-lane__dot sh-lane__dot--warning"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">
                  {{ member.memberName || member.memberId }}
                </div>
                <div class="sh-lane__subtitle">
                  <span class="sh-mono">{{ member.guildId }}</span>
                  · {{ member.profile?.verificationState || member.verificationState }}
                  · 截止 {{ formatTimestamp(member.deadlineAt) }}
                </div>
              </div>
              <span class="sh-lane__chevron" aria-hidden="true">›</span>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          title="成员详情"
          description="绑定状态、认证状态与限制上下文。"
        >
          <EmptyState
            v-if="!selectedMember"
            title="请选择左侧成员"
            body="选中后这里会展示绑定、认证、限制截止与最近错误。"
          />
          <dl v-else class="sh-keylist">
            <dt>成员</dt>
            <dd>{{ selectedMember.memberName || selectedMember.memberId }}</dd>
            <dt>成员 ID</dt>
            <dd class="sh-mono">{{ selectedMember.memberId }}</dd>
            <dt>所属群</dt>
            <dd class="sh-mono">{{ selectedMember.guildId }}</dd>
            <dt>认证状态</dt>
            <dd>{{ detailCards[0]?.value || '—' }}</dd>
            <dt>绑定记录</dt>
            <dd>{{ detailCards[1]?.value || '—' }}</dd>
            <dt>限制截止</dt>
            <dd>{{ detailCards[2]?.value || '—' }}</dd>
            <template v-if="selectedMember.lastError">
              <dt>最近错误</dt>
              <dd>{{ detailCards[3]?.value || '—' }}</dd>
            </template>
          </dl>
        </WorkspaceSection>
      </div>

      <WorkspaceSection
        title="最近自动解除"
        description="已完成认证或已被手动释放的成员。"
        :meta="`${data.recentReleases.length} 条`"
        flush
      >
        <EmptyState
          v-if="data.recentReleases.length === 0"
          title="暂无自动解除记录"
          body="当成员通过认证或被管理员手动释放时会出现在这里。"
        />
        <div v-else class="sh-table-shell">
          <el-table :data="data.recentReleases" row-key="id">
            <el-table-column label="成员" prop="memberName">
              <template #default="{ row }">
                <div class="sh-table__stack">
                  <div>{{ row.memberName || row.memberId }}</div>
                  <div class="sh-table__id">{{ row.memberId }}</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="群组">
              <template #default="{ row }">
                <span class="sh-table__mono">{{ row.guildId }}</span>
              </template>
            </el-table-column>
            <el-table-column label="释放时间">
              <template #default="{ row }">
                <span class="sh-table__mono">{{ formatTimestamp(row.releasedAt) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="认证状态">
              <template #default="{ row }">
                <SeverityTag
                  :label="row.profile?.verificationState || row.verificationState"
                  intent="success"
                />
              </template>
            </el-table-column>
          </el-table>
        </div>
      </WorkspaceSection>

      <WorkspaceSection
        v-if="data.lookupErrors.length"
        title="查询错误"
        description="这些成员的实时平台认证状态没有成功返回。"
        :meta="`${data.lookupErrors.length} 条`"
        flush
      >
        <div class="sh-lane">
          <div
            v-for="item in data.lookupErrors"
            :key="`${item.memberId}:${item.message}`"
            class="sh-lane__row"
          >
            <span class="sh-lane__dot sh-lane__dot--danger"></span>
            <div class="sh-lane__body">
              <div class="sh-lane__title sh-mono">{{ item.memberId }}</div>
              <div class="sh-lane__subtitle">{{ item.message }}</div>
            </div>
          </div>
        </div>
      </WorkspaceSection>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { IdentityPageData } from '../page-types'
import { formatTimestamp } from '../models/formatters'
import { buildIdentityModel } from '../models/identity'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const error = ref('')
const data = ref<IdentityPageData | null>(null)
const selectedGuildId = ref('')
const selectedMemberId = ref('')
const keyword = ref('')

const metrics = computed(() => {
  const summary = data.value?.summary
  if (!summary) return []
  return [
    { label: '待认证成员', value: summary.pendingMembers, note: '仍处于限制中的成员数。' },
    { label: '已认证成员', value: summary.verifiedMembers, note: '已完成 StuHelper 学生认证。' },
    { label: '待完善认证', value: summary.boundUnverifiedMembers, note: '已经绑定但尚未完成认证。' },
    { label: '最近释放', value: summary.releasedMembers, note: '已自动解除限制的成员数。' },
  ]
})

const model = computed(() => {
  if (!data.value) return null
  return buildIdentityModel(data.value, {
    guildId: selectedGuildId.value,
    itemId: selectedMemberId.value,
    keyword: keyword.value,
  })
})
const filteredMembers = computed(() => model.value?.filteredMembers ?? [])
const selectedMember = computed(() => model.value?.selectedMember ?? null)
const detailCards = computed(() => model.value?.detailCards ?? [])

loadData()

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await consolePageApi.identity()
    selectedGuildId.value = props.navigation?.state.value.guildId || ''
    keyword.value = props.navigation?.state.value.keyword || ''
    const requestedId = props.navigation?.state.value.itemId || ''
    selectedMemberId.value = requestedId || data.value.members[0]?.id || ''
    syncSelection()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function selectMember(memberId: string) {
  selectedMemberId.value = memberId
  syncSelection()
}

function syncSelection() {
  const fallback = filteredMembers.value[0] ?? null
  const member = filteredMembers.value.find((item) => item.id === selectedMemberId.value) ?? fallback
  if (!member) {
    props.navigation?.replaceState({
      workspace: 'members',
      guildId: selectedGuildId.value || null,
      memberId: null,
      itemId: null,
      keyword: keyword.value,
    })
    return
  }
  selectedMemberId.value = member.id
  props.navigation?.replaceState({
    workspace: 'members',
    guildId: member.guildId,
    memberId: member.memberId,
    itemId: member.id,
    keyword: keyword.value,
  })
}

function metricIntent(index: number, value: number): string {
  if (index === 0) {
    if (value === 0) return ''
    if (value > 20) return 'sh-stat--danger'
    return 'sh-stat--warning'
  }
  if (index === 1 && value > 0) return 'sh-stat--success'
  if (index === 2 && value > 0) return 'sh-stat--warning'
  return ''
}
</script>

<style scoped>
.sh-workspace-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--sh-s-4);
  padding: var(--sh-s-5) var(--sh-s-6);
  background: var(--sh-surface-0);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-3);
  box-shadow: var(--sh-shadow-card);
}

.sh-workspace-head__copy {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-2);
  min-width: 0;
}

.sh-workspace-head__title {
  font-size: var(--sh-t-heading);
  font-weight: var(--sh-w-semibold);
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.sh-workspace-head__description {
  font-size: var(--sh-t-body);
  color: var(--sh-fg-2);
  line-height: var(--sh-l-normal);
  max-width: 64ch;
}

.sh-workspace-head__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sh-s-2);
  margin-top: var(--sh-s-1);
}

.sh-workspace-head__actions {
  flex-shrink: 0;
}

.sh-identity-toolbar {
  gap: var(--sh-s-3);
  padding: 0;
}

.sh-identity-toolbar__field {
  min-width: 200px;
  flex: 0 1 220px;
}

.sh-identity-toolbar__field--wide {
  flex-basis: 320px;
}

.sh-lane__body {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-align: left;
}

.sh-lane__row--interactive {
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--sh-border);
  color: var(--sh-fg);
  cursor: pointer;
  padding: 10px 14px;
  transition: background var(--sh-dur-fast) var(--sh-ease);
  width: 100%;
}

.sh-lane__row--interactive:last-child {
  border-bottom: none;
}

.sh-lane__row--interactive:hover {
  background: var(--sh-surface-hover);
}

.sh-lane__row--active {
  background: var(--sh-primary-soft);
  box-shadow: inset 2px 0 0 var(--sh-primary);
}

.sh-lane__chevron {
  color: var(--sh-fg-3);
  font-size: var(--sh-t-title);
  line-height: 1;
}

.sh-table__stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

@media (max-width: 1080px) {
  .sh-workspace-head {
    flex-direction: column;
  }
}
</style>
