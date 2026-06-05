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
      v-if="error && !data"
      title="加载身份认证数据失败"
      :body="error"
      tone="error"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="loadData">重试</el-button>
      </template>
    </EmptyState>
    <ConsolePageSkeleton v-else-if="loading && !data" />

    <template v-else-if="data">
      <div v-if="error" class="sh-load-error" role="alert">
        <div class="sh-load-error__body">
          <strong>刷新身份认证数据失败</strong>
          <span>{{ error }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="loadData">重试</el-button>
      </div>

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
                  <EntityChip kind="guild" :id="member.guildId" inline />
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
            <dd>
              <EntityChip
                kind="user"
                :id="String(selectedMember.memberId)"
                :name="selectedMember.memberName || undefined"
                :guild-id="selectedMember.guildId || undefined"
              />
            </dd>
            <dt>所属群</dt>
            <dd>
              <EntityChip kind="guild" :id="String(selectedMember.guildId)" />
            </dd>
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
                <EntityChip
                  kind="user"
                  :id="String(row.memberId)"
                  :name="row.memberName || undefined"
                  :guild-id="row.guildId || undefined"
                />
              </template>
            </el-table-column>
            <el-table-column label="群组">
              <template #default="{ row }">
                <EntityChip kind="guild" :id="String(row.guildId)" />
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
              <div class="sh-lane__title">
                <EntityChip kind="user" :id="String(item.memberId)" />
              </div>
              <div class="sh-lane__subtitle">{{ item.message }}</div>
            </div>
          </div>
        </div>
      </WorkspaceSection>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { IdentityPageData } from '../page-types'
import { formatTimestamp } from '../models/formatters'
import { buildIdentityModel } from '../models/identity'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const NAV_KEY_SEPARATOR = '|'

const loading = ref(false)
const error = ref('')
const data = ref<IdentityPageData | null>(null)
const selectedGuildId = ref('')
const selectedMemberId = ref('')
const keyword = ref('')
let loadRequestSeq = 0

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

watch(
  () => navigationStateKey(),
  () => {
    const state = props.navigation?.state.value
    if (state?.view !== 'identity' || !data.value) return
    applyNavigationState()
    syncSelection()
  },
)

async function loadData() {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  error.value = ''
  try {
    const next = await consolePageApi.identity()
    if (requestSeq !== loadRequestSeq) return
    data.value = next
    applyNavigationState()
    syncSelection()
  } catch (cause) {
    if (requestSeq !== loadRequestSeq) return
    error.value = errorMessage(cause, '加载身份认证数据失败')
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

function errorMessage(cause: unknown, fallback: string): string {
  if (cause instanceof Error && cause.message) return cause.message
  if (typeof cause === 'string' && cause.trim()) return cause
  return fallback
}

function applyNavigationState() {
  const state = props.navigation?.state.value
  selectedGuildId.value = state?.guildId || ''
  keyword.value = typeof state?.keyword === 'string' ? state.keyword : state?.memberId || ''
  selectedMemberId.value = state?.itemId || ''
}

function navigationStateKey(): string {
  const state = props.navigation?.state.value
  if (!state) return ''
  return [
    state.view,
    state.guildId ?? '',
    state.memberId ?? '',
    state.itemId ?? '',
    state.keyword,
  ].join(NAV_KEY_SEPARATOR)
}

function selectMember(memberId: string) {
  selectedMemberId.value = memberId
  syncSelection()
}

function syncSelection() {
  const fallback = filteredMembers.value[0] ?? null
  const member = filteredMembers.value.find((item) => item.id === selectedMemberId.value) ?? fallback
  const filterGuildId = selectedGuildId.value || null
  if (!member) {
    props.navigation?.replaceState({
      workspace: 'members',
      guildId: filterGuildId,
      memberId: null,
      itemId: null,
      keyword: keyword.value,
    })
    return
  }
  selectedMemberId.value = member.id
  props.navigation?.replaceState({
    workspace: 'members',
    guildId: filterGuildId,
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

.sh-table__stack {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
</style>
