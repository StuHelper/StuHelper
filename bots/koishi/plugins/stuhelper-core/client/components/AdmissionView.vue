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
          :disabled="!props.navigation"
          @click="openPolicyWorkspace('bindings')"
        >
          管理群绑定
        </el-button>
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="!props.navigation"
          @click="openPolicyWorkspace('templates')"
        >
          编辑模板
        </el-button>
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
      title="加载入群认证数据失败"
      :body="error"
      tone="error"
    >
      <template #action>
        <el-button class="sh-button sh-button--ghost" @click="loadData">重试</el-button>
      </template>
    </EmptyState>
    <ConsolePageSkeleton v-else-if="loading && !data" />

    <template v-else-if="data && model">
      <div v-if="error" class="sh-load-error" role="alert">
        <div class="sh-load-error__body">
          <strong>刷新入群认证数据失败</strong>
          <span>{{ error }}</span>
        </div>
        <el-button class="sh-button sh-button--ghost" @click="loadData">重试</el-button>
      </div>

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
              <el-switch
                v-if="row.editable && row.settingKey"
                :model-value="Boolean(row.value)"
                :loading="settingLoadingKey === row.settingKey"
                :aria-label="runtimeSwitchLabel(row)"
                @change="(value: boolean | string | number) => submitRuntimeSetting(row, Boolean(value))"
              />
            </div>
          </div>
          <p v-if="settingsNotice" class="sh-field__hint sh-admission__notice">{{ settingsNotice }}</p>
          <p v-if="settingsError" class="sh-admission__error" role="alert">{{ settingsError }}</p>
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
            <el-table-column label="操作" width="260" fixed="right">
              <template #default="{ row }">
                <div class="sh-table__actions">
                  <el-button
                    v-for="action in row.availableActions"
                    :key="action"
                    size="small"
                    :type="actionTone(action)"
                    :disabled="actionLoadingKey === `${row.id}:${action}`"
                    @click="submitMemberAction(row, action)"
                  >
                    {{ actionLabel(action) }}
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>
        <p v-if="notice" class="sh-field__hint sh-admission__notice">{{ notice }}</p>
        <p v-if="actionError" class="sh-admission__error" role="alert">{{ actionError }}</p>
      </WorkspaceSection>
    </template>

    <ConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :tone="confirmDialog.tone"
      :confirm-text="confirmDialog.confirmText"
      :cancel-text="confirmDialog.cancelText"
      @confirm="acceptConfirm"
      @cancel="cancelConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { message } from '@koishijs/client'
import { computed, ref } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { useActionError } from '../composables/use-action-error'
import { useConfirm } from '../composables/use-confirm'
import { consolePageApi } from '../page-api'
import { formatTimestamp } from '../models/formatters'
import {
  buildAdmissionRuntimeModel,
  type AdmissionMetric,
  type AdmissionRuntimeAction,
  type AdmissionRuntimePageData,
  type AdmissionRuntimeMember,
  type AdmissionRuntimeSettingsPatch,
  type AdmissionSwitchRow,
} from '../models/admission-runtime'
import { errorMessage } from '../utils/error-message'
import ConfirmDialog from './primitives/ConfirmDialog.vue'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import EntityChip from './primitives/EntityChip.vue'
import SeverityTag from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'

type GuardPolicyWorkspace = 'templates' | 'bindings'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const error = ref('')
const notice = ref('')
const settingsNotice = ref('')
const {
  actionError,
  setActionError,
  clearActionError,
} = useActionError({
  onError: (_title, details) => message.error(details),
})
const {
  actionError: settingsError,
  setActionError: setSettingsError,
  clearActionError: clearSettingsError,
} = useActionError({
  defaultTitle: '保存运行开关失败',
  onError: (_title, details) => message.error(details),
})
const actionLoadingKey = ref('')
const settingLoadingKey = ref('')
const data = ref<AdmissionRuntimePageData | null>(null)
let loadRequestSeq = 0
const { state: confirmDialog, confirm, accept: acceptConfirm, cancel: cancelConfirm } = useConfirm()

const model = computed(() => data.value ? buildAdmissionRuntimeModel(data.value) : null)

loadData()

function openPolicyWorkspace(workspace: GuardPolicyWorkspace) {
  props.navigation?.selectView('config', { workspace })
}

async function loadData() {
  const requestSeq = ++loadRequestSeq
  loading.value = true
  error.value = ''
  try {
    const next = await consolePageApi.admissionRuntime()
    if (requestSeq !== loadRequestSeq) return
    data.value = next
    clearActionError()
    clearSettingsError()
  } catch (cause) {
    if (requestSeq !== loadRequestSeq) return
    error.value = errorMessage(cause, '加载入群认证数据失败')
  } finally {
    if (requestSeq === loadRequestSeq) {
      loading.value = false
    }
  }
}

async function submitMemberAction(
  member: AdmissionRuntimeMember,
  action: AdmissionRuntimeAction,
) {
  const label = actionLabel(action)
  if (requiresConfirm(action)) {
    const confirmed = await confirm({
      title: '确认入群认证动作',
      message: `确定要对 QQ ${member.memberId} 执行「${label}」吗？`,
      tone: confirmationTone(action),
      confirmText: label,
    })
    if (!confirmed) return
  }
  actionLoadingKey.value = `${member.id}:${action}`
  clearActionError()
  notice.value = ''
  try {
    notice.value = await consolePageApi.admissionAction({
      recordId: member.id,
      action,
    })
    await loadData()
  } catch (cause) {
    setActionError('入群认证操作失败', cause, '入群认证操作失败')
  } finally {
    actionLoadingKey.value = ''
  }
}

async function submitRuntimeSetting(row: AdmissionSwitchRow, enabled: boolean) {
  if (!row.settingKey) return
  settingLoadingKey.value = row.settingKey
  clearSettingsError()
  settingsNotice.value = ''
  const patch: AdmissionRuntimeSettingsPatch = {
    [row.settingKey]: enabled,
  }
  try {
    settingsNotice.value = await consolePageApi.saveAdmissionRuntimeSettings(patch)
    await loadData()
  } catch (cause) {
    setSettingsError('保存入群认证运行开关失败', cause, '保存入群认证运行开关失败')
  } finally {
    settingLoadingKey.value = ''
  }
}

function actionLabel(action: AdmissionRuntimeAction) {
  return {
    query: '查询',
    resend: '重发',
    regenerate: '重建',
    skip: '跳过',
    'reset-failures': '清次数',
    'release-blacklist': '解拉黑',
  }[action]
}

function actionTone(action: AdmissionRuntimeAction) {
  if (action === 'skip' || action === 'release-blacklist') return 'warning'
  if (action === 'regenerate') return 'primary'
  return undefined
}

function requiresConfirm(action: AdmissionRuntimeAction) {
  return action !== 'query'
}

function confirmationTone(action: AdmissionRuntimeAction) {
  return action === 'skip' || action === 'release-blacklist' ? 'danger' : 'normal'
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

function runtimeSwitchLabel(row: AdmissionSwitchRow) {
  return `切换入群认证运行开关：${row.label}`
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

.sh-table__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sh-admission__notice {
  margin: 12px 0 0;
  color: var(--sh-success);
}

.sh-admission__error {
  margin: 12px 0 0;
  padding: var(--sh-s-3);
  border: 1px solid color-mix(in srgb, var(--sh-danger) 38%, transparent);
  border-radius: var(--sh-radius-md);
  background: color-mix(in srgb, var(--sh-danger) 10%, transparent);
  color: var(--sh-danger);
  font-size: var(--sh-t-body);
}
</style>
