<template>
  <div class="sh-view">
    <header class="sh-workspace-head">
      <div class="sh-workspace-head__copy">
        <h1 class="sh-workspace-head__title">入群认证</h1>
        <p class="sh-workspace-head__description">
          Admission 运行态、目标群策略、受限成员队列与学生认证联动状态。
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

    <template v-else-if="data && model">
      <section class="sh-dashboard-metrics">
        <article
          v-for="metric in model.metrics"
          :key="metric.label"
          class="sh-stat sh-dashboard-metric"
          :class="metricClass(metric.tone)"
        >
          <span class="sh-stat__label">{{ metric.label }}</span>
          <span class="sh-stat__value sh-num">{{ metric.value }}</span>
          <span class="sh-stat__note">{{ metric.note }}</span>
        </article>
      </section>

      <div class="sh-split sh-split--7-5">
        <WorkspaceSection
          title="运行开关"
          description="来自 stuhelper-group-guard 当前实例的实际配置。"
          :meta="`${model.switchRows.length} 项`"
          flush
        >
          <div class="sh-lane">
            <div
              v-for="row in model.switchRows"
              :key="row.id"
              class="sh-lane__row"
            >
              <span class="sh-lane__dot" :class="switchDotClass(row.tone)"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">{{ row.label }}</div>
                <div class="sh-lane__subtitle">
                  <SeverityTag :label="formatSwitchValue(row.value)" :intent="row.tone" />
                  <span>{{ row.note }}</span>
                </div>
              </div>
            </div>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          title="实例"
          description="服务端地址、Bot 实例与默认 guard 参数。"
        >
          <dl class="sh-keylist">
            <dt>平台 API</dt>
            <dd class="sh-mono">{{ data.platform.baseUrl || '未配置' }}</dd>
            <dt>服务凭据</dt>
            <dd>{{ data.platform.serviceTokenConfigured ? '已配置' : '未配置' }}</dd>
            <dt>默认禁言</dt>
            <dd>{{ data.guard.muteDurationSeconds }} 秒</dd>
            <dt>宽限期</dt>
            <dd>{{ data.guard.kickAfterMinutes }} 分钟</dd>
            <dt>豁免人数</dt>
            <dd>{{ data.guard.exemptUserCount }}</dd>
            <dt>Bot</dt>
            <dd>
              <span v-if="data.bots.length === 0">未连接</span>
              <template v-else>
                <span
                  v-for="bot in data.bots"
                  :key="`${bot.platform}:${bot.selfId}`"
                  class="sh-inline-chip sh-mono"
                >
                  {{ bot.platform }}/{{ bot.selfId }}
                </span>
              </template>
            </dd>
          </dl>
        </WorkspaceSection>
      </div>

      <div class="sh-split sh-split--1-1">
        <WorkspaceSection
          title="目标群与绑定"
          description="静态 targetGroups 和数据库群绑定都会影响本地 guard policy 解析。"
          :meta="`${data.guard.targetGroups.length + data.bindings.length} 条`"
          flush
        >
          <div class="sh-lane">
            <div
              v-for="guildId in data.guard.targetGroups"
              :key="`static:${guildId}`"
              class="sh-lane__row"
            >
              <span class="sh-lane__dot sh-lane__dot--warning"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">
                  <EntityChip kind="guild" :id="guildId" />
                </div>
                <div class="sh-lane__subtitle">静态 fallback · targetGroups</div>
              </div>
            </div>
            <div
              v-for="binding in data.bindings"
              :key="binding.id"
              class="sh-lane__row"
            >
              <span class="sh-lane__dot" :class="binding.enabled ? 'sh-lane__dot--primary' : ''"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">
                  <EntityChip kind="guild" :id="binding.guildId" />
                </div>
                <div class="sh-lane__subtitle">
                  <span class="sh-mono">{{ binding.platform }}</span>
                  · {{ binding.templateId }}
                  · {{ binding.enabled ? '已启用' : '已停用' }}
                </div>
              </div>
            </div>
          </div>
          <EmptyState
            v-if="data.guard.targetGroups.length === 0 && data.bindings.length === 0"
            title="暂无目标群"
            body="当前实例没有静态目标群，也没有数据库群绑定。"
          />
        </WorkspaceSection>

        <WorkspaceSection
          title="模板"
          description="数据库模板会覆盖静态默认 guard 参数。"
          :meta="`${data.templates.length} 条`"
          flush
        >
          <EmptyState
            v-if="data.templates.length === 0"
            title="暂无模板"
            body="可在群组配置的模板库工作区创建。"
          />
          <div v-else class="sh-table-shell">
            <el-table :data="data.templates" row-key="id">
              <el-table-column label="模板" min-width="160">
                <template #default="{ row }">
                  <div class="sh-table__stack">
                    <span>{{ row.name }}</span>
                    <span class="sh-mono">{{ row.id }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="96">
                <template #default="{ row }">
                  <SeverityTag
                    :label="row.enabled ? '启用' : '停用'"
                    :intent="row.enabled ? 'success' : 'muted'"
                  />
                </template>
              </el-table-column>
              <el-table-column label="禁言" width="96">
                <template #default="{ row }">{{ row.muteDurationSeconds }} 秒</template>
              </el-table-column>
              <el-table-column label="踢出" width="96">
                <template #default="{ row }">{{ row.kickAfterMinutes }} 分</template>
              </el-table-column>
            </el-table>
          </div>
        </WorkspaceSection>
      </div>

      <WorkspaceSection
        title="受限成员队列"
        description="当前仍未释放、未踢出的本地 guard records。"
        :meta="`${model.activeMembers.length} 条`"
        flush
      >
        <EmptyState
          v-if="model.activeMembers.length === 0"
          title="暂无受限成员"
          body="当前没有待认证或待后端同步的本地成员记录。"
        />
        <div v-else class="sh-table-shell">
          <el-table :data="model.activeMembers" row-key="id">
            <el-table-column label="成员" min-width="170">
              <template #default="{ row }">
                <EntityChip
                  kind="user"
                  :id="row.memberId"
                  :name="row.memberName || undefined"
                  :guild-id="row.guildId"
                />
              </template>
            </el-table-column>
            <el-table-column label="群" width="130">
              <template #default="{ row }">
                <EntityChip kind="guild" :id="row.guildId" />
              </template>
            </el-table-column>
            <el-table-column label="状态" width="150">
              <template #default="{ row }">
                <div class="sh-table__stack">
                  <SeverityTag
                    :label="row.verificationState"
                    :intent="row.backendSyncPending ? 'warning' : 'primary'"
                  />
                  <span v-if="row.backendSyncPending" class="sh-mono">backend pending</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="截止" width="170">
              <template #default="{ row }">
                <span class="sh-mono">{{ formatTimestamp(row.deadlineAt) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="会话" min-width="180">
              <template #default="{ row }">
                <span class="sh-mono">{{ row.admissionSessionID || '未创建' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="错误" min-width="180">
              <template #default="{ row }">
                {{ row.lastError || '无' }}
              </template>
            </el-table-column>
          </el-table>
        </div>
      </WorkspaceSection>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import { consolePageApi } from '../page-api'
import { formatTimestamp } from '../models/formatters'
import {
  buildAdmissionRuntimeModel,
  type AdmissionMetric,
  type AdmissionRuntimePageData,
  type AdmissionSwitchRow,
} from '../models/admission-runtime'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

const loading = ref(false)
const error = ref('')
const data = ref<AdmissionRuntimePageData | null>(null)

const model = computed(() => data.value ? buildAdmissionRuntimeModel(data.value) : null)

loadData()

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await consolePageApi.admissionRuntime()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function metricClass(tone: AdmissionMetric['tone']) {
  if (!tone) return ''
  return `sh-stat--${tone}`
}

function switchDotClass(tone: AdmissionSwitchRow['tone']) {
  if (tone === 'success') return 'sh-lane__dot--primary'
  if (tone === 'warning') return 'sh-lane__dot--warning'
  if (tone === 'danger') return 'sh-lane__dot--danger'
  return 'sh-lane__dot--primary'
}

function formatSwitchValue(value: AdmissionSwitchRow['value']) {
  if (typeof value === 'boolean') return value ? '启用' : '关闭'
  return String(value)
}
</script>

<style scoped>
.sh-inline-chip {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 8px;
  margin: 0 6px 6px 0;
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-1);
  background: var(--sh-surface-1);
  color: var(--sh-fg-2);
}

.sh-table__stack {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
</style>
