<template>
  <div class="console-view">
    <section class="console-header">
      <div>
        <h2 class="console-header__title">配置治理</h2>
        <div class="console-header__meta">
          <span class="console-chip">群配置 / 模板库 / 群绑定 / 命令策略</span>
          <span class="console-chip" v-if="data">更新于 {{ formatTimestamp(data.generatedAt) }}</span>
        </div>
      </div>
      <div class="console-toolbar">
        <button class="console-button" @click="loadData">刷新</button>
      </div>
    </section>

    <div class="console-tabs" v-if="data">
      <button
        v-for="workspace in configModel?.workspaceTabs ?? []"
        :key="workspace.id"
        class="console-tab"
        :class="{ 'console-tab--active': currentWorkspace === workspace.id }"
        @click="selectWorkspace(workspace.id)"
      >
        {{ workspace.label }}
      </button>
    </div>

    <div v-if="error" class="console-error">{{ error }}</div>
    <div v-else-if="loading" class="console-empty">正在加载治理配置…</div>
    <template v-else-if="data">
      <LegacyConfigView v-if="currentWorkspace === 'guild-config'" />

      <section v-else class="console-split">
        <section class="console-panel">
          <template v-if="currentWorkspace === 'templates'">
            <div>
              <h3 class="console-panel__title">模板库</h3>
              <div class="console-panel__subtitle">模板定义 guard 的禁言时长、提醒文案与豁免名单。</div>
            </div>
            <div class="console-list">
              <button
                v-for="item in configModel?.templateRows ?? []"
                :key="item.id"
                type="button"
                class="console-list__item console-list__item--interactive"
                :class="{ 'console-list__item--active': templateForm.id === item.id }"
                @click="loadTemplate(item)"
              >
                <span class="console-list__title">{{ item.name }}</span>
                <span class="console-list__meta">{{ item.id }} · {{ item.muteDurationSeconds }} 秒 · {{ item.kickAfterMinutes }} 分钟</span>
              </button>
            </div>
          </template>

          <template v-else-if="currentWorkspace === 'bindings'">
            <div>
              <h3 class="console-panel__title">群绑定</h3>
              <div class="console-panel__subtitle">查看每个群组当前绑定到哪个模板。</div>
            </div>
            <div class="console-list">
              <button
                v-for="item in configModel?.bindingRows ?? []"
                :key="item.id"
                type="button"
                class="console-list__item console-list__item--interactive"
                :class="{ 'console-list__item--active': bindingForm.guildId === item.guildId && bindingForm.platform === item.platform }"
                @click="loadBinding(item)"
              >
                <span class="console-list__title">{{ item.guildId }}</span>
                <span class="console-list__meta">{{ item.platform }} · {{ item.effectiveTemplateName }} · {{ item.note || '无备注' }}</span>
              </button>
            </div>
          </template>

          <template v-else>
            <div>
              <h3 class="console-panel__title">命令策略</h3>
              <div class="console-panel__subtitle">全局命令权限与支持的命令列表。</div>
            </div>
            <div class="console-list">
              <button
                v-for="item in configModel?.policyRows ?? []"
                :key="item.commandId"
                type="button"
                class="console-list__item console-list__item--interactive"
                :class="{ 'console-list__item--active': policyForm.commandId === item.commandId }"
                @click="loadPolicy(item)"
              >
                <span class="console-list__title">{{ item.commandId }}</span>
                <span class="console-list__meta">authority {{ item.minAuthority }} · {{ item.roles.join(', ') || '无角色限制' }}</span>
              </button>
            </div>
            <div class="console-note">支持命令：{{ configModel?.supportedCommandIds.join(', ') || '' }}</div>
          </template>
        </section>

        <section class="console-panel">
          <template v-if="currentWorkspace === 'templates'">
            <div>
              <h3 class="console-panel__title">编辑模板</h3>
              <div class="console-panel__subtitle">模板 ID、提醒文案和白名单直接保存到 guard policy store。</div>
            </div>
            <div class="console-stack">
              <input v-model.trim="templateForm.id" class="console-input" type="text" placeholder="模板 ID" />
              <input v-model.trim="templateForm.name" class="console-input" type="text" placeholder="模板名称" />
              <input v-model.number="templateForm.muteDurationSeconds" class="console-input" type="number" placeholder="禁言秒数" />
              <input v-model.number="templateForm.kickAfterMinutes" class="console-input" type="number" placeholder="踢出分钟阈值" />
              <input v-model.trim="templateForm.reminderTemplate" class="console-input" type="text" placeholder="提醒文案" />
              <input v-model.trim="templateForm.exemptUsersText" class="console-input" type="text" placeholder="白名单成员，逗号分隔" />
              <label class="console-chip"><input v-model="templateForm.enabled" type="checkbox" /> 启用模板</label>
              <button class="console-button console-button--primary" :disabled="submitting" @click="submitTemplate">保存模板</button>
            </div>
          </template>

          <template v-else-if="currentWorkspace === 'bindings'">
            <div>
              <h3 class="console-panel__title">编辑绑定</h3>
              <div class="console-panel__subtitle">群与模板的映射关系。</div>
            </div>
            <div class="console-stack">
              <input v-model.trim="bindingForm.platform" class="console-input" type="text" placeholder="平台，例如 onebot" />
              <input v-model.trim="bindingForm.guildId" class="console-input" type="text" placeholder="群号" />
              <select v-model="bindingForm.templateId" class="console-select">
                <option value="">选择模板</option>
                <option v-for="item in configModel?.templateRows ?? []" :key="item.id" :value="item.id">{{ item.name }}</option>
              </select>
              <input v-model.trim="bindingForm.note" class="console-input" type="text" placeholder="备注" />
              <label class="console-chip"><input v-model="bindingForm.enabled" type="checkbox" /> 启用绑定</label>
              <button class="console-button console-button--primary" :disabled="submitting" @click="submitBinding">保存绑定</button>
            </div>
          </template>

          <template v-else>
            <div>
              <h3 class="console-panel__title">编辑命令策略</h3>
              <div class="console-panel__subtitle">命令与 authority / 角色白名单的全局控制。</div>
            </div>
            <div class="console-stack">
              <select v-model="policyForm.commandId" class="console-select">
                <option value="">选择命令</option>
                <option v-for="commandId in configModel?.supportedCommandIds ?? []" :key="commandId" :value="commandId">{{ commandId }}</option>
              </select>
              <input v-model.number="policyForm.minAuthority" class="console-input" type="number" min="0" max="5" placeholder="最小 authority" />
              <input v-model.trim="policyForm.rolesText" class="console-input" type="text" placeholder="角色列表，逗号分隔" />
              <button class="console-button console-button--primary" :disabled="submitting" @click="submitPolicy">保存策略</button>
            </div>
          </template>
          <div v-if="notice" class="console-note">{{ notice }}</div>
        </section>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { ConfigGovernancePageData } from '../page-types'
import { assignBindingForm, assignPolicyForm, assignTemplateForm, splitCommaTokens } from '../models/config-forms'
import { formatTimestamp } from '../models/formatters'
import { buildConfigGovernanceModel } from '../models/config'
import LegacyConfigView from './ConfigView.vue'

const props = defineProps<{ navigation?: ConsoleNavigationController }>()

const loading = ref(false)
const error = ref('')
const data = ref<ConfigGovernancePageData | null>(null)
const currentWorkspace = ref<'guild-config' | 'templates' | 'bindings' | 'command-policies'>('guild-config')
const notice = ref('')
const submitting = ref(false)
const templateForm = reactive({
  id: '',
  name: '',
  muteDurationSeconds: 1800,
  kickAfterMinutes: 30,
  reminderTemplate: '',
  exemptUsersText: '',
  enabled: true,
})
const bindingForm = reactive({
  platform: 'onebot',
  guildId: '',
  templateId: '',
  note: '',
  enabled: true,
})
const policyForm = reactive({
  commandId: '',
  minAuthority: 3,
  rolesText: '',
})
const configModel = computed(() => data.value ? buildConfigGovernanceModel(data.value, { workspace: currentWorkspace.value }) : null)

watch(
  () => props.navigation?.state.value.workspace,
  (workspace) => {
    if (workspace === 'templates' || workspace === 'bindings' || workspace === 'command-policies') {
      currentWorkspace.value = workspace
      return
    }
    currentWorkspace.value = 'guild-config'
  },
  { immediate: true },
)

loadData()

async function loadData() {
  loading.value = true
  error.value = ''
  try {
    data.value = await consolePageApi.configGovernance()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function selectWorkspace(workspace: 'guild-config' | 'templates' | 'bindings' | 'command-policies') {
  currentWorkspace.value = workspace
  props.navigation?.replaceState({ workspace })
}

function loadTemplate(item: ConfigGovernancePageData['templates'][number]) {
  assignTemplateForm(templateForm, item)
}

function loadBinding(item: ConfigGovernancePageData['bindings'][number]) {
  assignBindingForm(bindingForm, item)
}

function loadPolicy(item: ConfigGovernancePageData['commandPolicies'][number]) {
  assignPolicyForm(policyForm, item)
}

async function submitTemplate() {
  submitting.value = true
  notice.value = ''
  error.value = ''
  try {
    notice.value = await consolePageApi.saveGuardTemplate({
      id: templateForm.id,
      name: templateForm.name,
      muteDurationSeconds: Number(templateForm.muteDurationSeconds),
      kickAfterMinutes: Number(templateForm.kickAfterMinutes),
      reminderTemplate: templateForm.reminderTemplate,
      exemptUsers: splitCommaTokens(templateForm.exemptUsersText),
      enabled: templateForm.enabled,
    })
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    submitting.value = false
  }
}

async function submitBinding() {
  submitting.value = true
  notice.value = ''
  error.value = ''
  try {
    notice.value = await consolePageApi.saveGuardBinding({
      platform: bindingForm.platform,
      guildId: bindingForm.guildId,
      templateId: bindingForm.templateId,
      enabled: bindingForm.enabled,
      note: bindingForm.note || undefined,
    })
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    submitting.value = false
  }
}

async function submitPolicy() {
  submitting.value = true
  notice.value = ''
  error.value = ''
  try {
    notice.value = await consolePageApi.saveCommandPolicy({
      commandId: policyForm.commandId,
      minAuthority: Number(policyForm.minAuthority),
      roles: splitCommaTokens(policyForm.rolesText),
    })
    await loadData()
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    submitting.value = false
  }
}
</script>
