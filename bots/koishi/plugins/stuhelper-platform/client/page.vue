<template>
  <k-layout main="sh-platform-shell">
    <el-scrollbar class="sh-platform__scroll" view-class="sh-platform__view">
      <div class="sh-platform">
        <header class="sh-platform__header">
          <div>
            <h1>StuHelper 平台</h1>
            <p>{{ generatedLabel }}</p>
          </div>
          <button class="sh-platform__button" :disabled="loading" @click="refresh">
            {{ loading ? '处理中' : '刷新' }}
          </button>
        </header>

        <nav class="sh-platform__tabs">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            class="sh-platform__tab"
            :class="{ 'is-active': activeTab === tab.id }"
            @click="activeTab = tab.id"
          >
            {{ tab.label }}
          </button>
        </nav>

        <p v-if="error" class="sh-platform__message sh-platform__message--error">
          {{ error }}
        </p>
        <p v-else-if="notice" class="sh-platform__message">
          {{ notice }}
        </p>

        <section v-if="!data" class="sh-platform__panel">
          加载中
        </section>

        <section v-else-if="activeTab === 'modules'" class="sh-platform__panel">
          <div class="sh-platform__module-list">
            <article
              v-for="module in view.modules"
              :key="module.id"
              class="sh-platform__module"
              :class="{ 'is-active': module.id === selectedModule?.id }"
            >
              <button class="sh-platform__module-main" @click="selectModule(module.id)">
                <span>{{ module.name }}</span>
                <span>{{ module.statusText }}</span>
              </button>
              <p>{{ module.description }}</p>
              <button
                class="sh-platform__button sh-platform__button--ghost"
                :disabled="loading"
                @click="toggleModule(module.id, !module.enabled)"
              >
                {{ module.enabled ? '停用' : '启用' }}
              </button>
            </article>
          </div>
        </section>

        <section v-else-if="activeTab === 'config'" class="sh-platform__panel">
          <div class="sh-platform__panel-head">
            <h2>{{ selectedModule?.name || '配置' }}</h2>
            <div class="sh-platform__actions">
              <button
                class="sh-platform__button sh-platform__button--ghost"
                :disabled="loading || !selectedModule"
                @click="toggleSelectedModule"
              >
                {{ selectedModule?.enabled ? '停用' : '启用' }}
              </button>
              <button
                class="sh-platform__button"
                :disabled="loading || !selectedModule"
                @click="saveConfig"
              >
                保存
              </button>
            </div>
          </div>
          <textarea
            v-model="configText"
            class="sh-platform__textarea"
            spellcheck="false"
          />
        </section>

        <section v-else-if="activeTab === 'groupPolicy'" class="sh-platform__panel">
          <div v-if="view.groupPolicyRows.length" class="sh-platform__rows">
            <div
              v-for="row in view.groupPolicyRows"
              :key="row.id"
              class="sh-platform__row sh-platform__row--stacked"
            >
              <span>{{ row.label }}</span>
              <span>{{ row.moduleName }}</span>
              <code>{{ row.id }}</code>
            </div>
          </div>
          <p v-else class="sh-platform__empty">暂无群策略</p>
        </section>

        <section v-else-if="activeTab === 'permissions'" class="sh-platform__panel">
          <div v-if="view.policyRows.length" class="sh-platform__rows">
            <div
              v-for="row in view.policyRows"
              :key="row.id"
              class="sh-platform__row sh-platform__row--stacked"
            >
              <span>{{ row.label }}</span>
              <span>{{ row.description }}</span>
              <code>{{ row.id }}</code>
            </div>
          </div>
          <p v-else class="sh-platform__empty">暂无权限</p>
        </section>

        <section v-else class="sh-platform__panel">
          <div v-if="view.auditRows.length" class="sh-platform__rows">
            <div
              v-for="row in view.auditRows"
              :key="row.id"
              class="sh-platform__row sh-platform__row--stacked"
            >
              <span>{{ row.summary }}</span>
              <span>{{ row.moduleName }} · {{ row.actor }} · {{ row.createdAt }}</span>
              <code>{{ row.action }}</code>
            </div>
          </div>
          <p v-else class="sh-platform__empty">暂无审计</p>
        </section>
      </div>
    </el-scrollbar>
  </k-layout>
</template>

<script setup lang="ts">
import { computed, shallowRef, watch } from 'vue'
import { store } from '@koishijs/client'

import {
  refreshPlatformData,
  saveModuleConfig,
  setModuleEnabled,
} from './api'
import {
  buildPlatformView,
  formatTimestamp,
  parseConfigText,
  type PlatformData,
} from './model'

const tabs = [
  { id: 'modules', label: '模块' },
  { id: 'config', label: '配置' },
  { id: 'groupPolicy', label: '群策略' },
  { id: 'permissions', label: '权限' },
  { id: 'audit', label: '审计' },
] as const

type TabId = typeof tabs[number]['id']

const emptyData: PlatformData = {
  generatedAt: '',
  modules: [],
  auditEvents: [],
}

const activeTab = shallowRef<TabId>('modules')
const selectedModuleId = shallowRef<string>()
const configText = shallowRef('')
const loading = shallowRef(false)
const notice = shallowRef('')
const error = shallowRef('')
const data = computed(
  () => (store as Record<string, unknown>).stuhelperPlatform as PlatformData | undefined,
)
const view = computed(() => buildPlatformView(data.value ?? emptyData, selectedModuleId.value))
const selectedModule = computed(() => view.value.selectedModule)
const generatedLabel = computed(() => {
  if (!data.value?.generatedAt) return '等待数据'
  return formatTimestamp(data.value.generatedAt)
})

watch(() => selectedModule.value?.id, (moduleId) => {
  if (moduleId && moduleId !== selectedModuleId.value) {
    selectedModuleId.value = moduleId
  }
}, { immediate: true })

watch(() => selectedModule.value?.configText, (text) => {
  configText.value = text ?? ''
}, { immediate: true })

function selectModule(moduleId: string) {
  selectedModuleId.value = moduleId
  activeTab.value = 'config'
}

async function refresh() {
  await runAction(() => refreshPlatformData(), '已刷新')
}

async function saveConfig() {
  await runAction(async () => {
    const module = requireSelectedModule()
    await saveModuleConfig(module.id, parseConfigText(configText.value))
  }, '已保存')
}

async function toggleSelectedModule() {
  const module = requireSelectedModule()
  await toggleModule(module.id, !module.enabled)
}

async function toggleModule(moduleId: string, enabled: boolean) {
  await runAction(() => setModuleEnabled(moduleId, enabled), enabled ? '已启用' : '已停用')
}

async function runAction(action: () => Promise<void>, successMessage: string) {
  loading.value = true
  error.value = ''
  notice.value = ''

  try {
    await action()
    notice.value = successMessage
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

function requireSelectedModule() {
  if (!selectedModule.value) {
    throw new Error('请选择模块')
  }
  return selectedModule.value
}
</script>
