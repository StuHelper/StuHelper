<template>
  <div class="console-view">
    <section class="console-header">
      <div>
        <h2 class="console-header__title">身份认证</h2>
        <div class="console-header__meta">
          <span class="console-chip">群优先视角</span>
          <span class="console-chip" v-if="data">更新于 {{ formatTimestamp(data.generatedAt) }}</span>
        </div>
      </div>
      <div class="console-toolbar">
        <select v-model="selectedGuildId" class="console-select" @change="syncSelection()">
          <option value="">全部群组</option>
          <option v-for="group in data?.groups ?? []" :key="group.guildId" :value="group.guildId">
            {{ group.guildId }} · {{ group.pendingCount }} 待处理
          </option>
        </select>
        <input v-model.trim="keyword" class="console-input" type="text" placeholder="搜索成员 ID 或昵称" @input="syncSelection()" />
        <button class="console-button" @click="loadData">刷新</button>
      </div>
    </section>

    <div v-if="error" class="console-error">{{ error }}</div>
    <div v-else-if="loading" class="console-empty">正在加载认证数据…</div>
    <template v-else-if="data">
      <section class="console-metrics">
        <article v-for="metric in metrics" :key="metric.label" class="console-metric">
          <div class="console-metric__label">{{ metric.label }}</div>
          <div class="console-metric__value">{{ metric.value }}</div>
          <div class="console-metric__note">{{ metric.note }}</div>
        </article>
      </section>

      <section class="console-split">
        <section class="console-panel">
          <div>
            <h3 class="console-panel__title">当前成员</h3>
            <div class="console-panel__subtitle">展示仍在限制中的成员。</div>
          </div>
          <div v-if="filteredMembers.length === 0" class="console-empty">没有匹配的成员。</div>
          <div v-else class="console-list">
            <button
              v-for="member in filteredMembers"
              :key="member.id"
              type="button"
              class="console-list__item console-list__item--interactive"
              :class="{ 'console-list__item--active': selectedMember?.id === member.id }"
              @click="selectMember(member.id)"
            >
              <span class="console-list__title">{{ member.memberName || member.memberId }}</span>
              <span class="console-list__meta">
                {{ member.guildId }} · {{ member.profile?.verificationState || member.verificationState }} · 截止 {{ formatTimestamp(member.deadlineAt) }}
              </span>
            </button>
          </div>
        </section>

        <section class="console-panel">
          <div>
            <h3 class="console-panel__title">成员详情</h3>
            <div class="console-panel__subtitle">绑定状态、认证状态与限制上下文。</div>
          </div>
          <div v-if="!selectedMember" class="console-empty">请选择左侧成员。</div>
          <div v-else class="console-stack">
            <div class="console-list__item">
              <span class="console-list__title">{{ selectedMember.memberName || selectedMember.memberId }}</span>
              <span class="console-list__meta">成员 ID {{ selectedMember.memberId }} · 群 {{ selectedMember.guildId }}</span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">认证状态</span>
              <span class="console-list__meta">{{ detailCards[0]?.value || '—' }}</span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">绑定记录</span>
              <span class="console-list__meta">{{ detailCards[1]?.value || '—' }}</span>
            </div>
            <div class="console-list__item">
              <span class="console-list__title">限制截止</span>
              <span class="console-list__meta">{{ detailCards[2]?.value || '—' }}</span>
            </div>
            <div class="console-list__item" v-if="selectedMember.lastError">
              <span class="console-list__title">最近错误</span>
              <span class="console-list__meta">{{ detailCards[3]?.value || '—' }}</span>
            </div>
          </div>
        </section>
      </section>

      <section class="console-panel">
        <div>
          <h3 class="console-panel__title">最近自动解除</h3>
          <div class="console-panel__subtitle">已完成认证或已被手动释放的成员。</div>
        </div>
        <table class="console-table">
          <thead>
            <tr>
              <th>成员</th>
              <th>群组</th>
              <th>释放时间</th>
              <th>认证状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in data.recentReleases" :key="item.id">
              <td>{{ item.memberName || item.memberId }}</td>
              <td>{{ item.guildId }}</td>
              <td>{{ formatTimestamp(item.releasedAt) }}</td>
              <td>{{ item.profile?.verificationState || item.verificationState }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section v-if="data.lookupErrors.length" class="console-panel">
        <div>
          <h3 class="console-panel__title">查询错误</h3>
          <div class="console-panel__subtitle">这些成员的实时平台认证状态没有成功返回。</div>
        </div>
        <div class="console-list">
          <div v-for="item in data.lookupErrors" :key="`${item.memberId}:${item.message}`" class="console-list__item">
            <span class="console-list__title">{{ item.memberId }}</span>
            <span class="console-list__meta">{{ item.message }}</span>
          </div>
        </div>
      </section>
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
  if (!data.value) {
    return null
  }
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
    props.navigation?.replaceState({ workspace: 'members', guildId: selectedGuildId.value || null, memberId: null, itemId: null, keyword: keyword.value })
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
</script>
