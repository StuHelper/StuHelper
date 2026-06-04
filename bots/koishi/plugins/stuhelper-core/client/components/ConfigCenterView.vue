<template>
  <div class="sh-view">
    <header class="sh-workspace-head">
      <div class="sh-workspace-head__copy">
        <h1 class="sh-workspace-head__title">配置治理</h1>
        <p class="sh-workspace-head__description">
          群配置以群为主键,治理子模型(模板 / 绑定 / 命令策略)共享顶层页,不共享同一个编辑表单。
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

    <nav class="sh-subnav" v-if="data" aria-label="配置治理工作区">
      <button
        v-for="workspace in configModel?.workspaceTabs ?? []"
        :key="workspace.id"
        type="button"
        class="sh-subnav__tab"
        :class="{ 'sh-subnav__tab--active': currentWorkspace === workspace.id }"
        @click="selectWorkspace(workspace.id)"
      >
        {{ workspace.label }}
      </button>
    </nav>

    <EmptyState
      v-if="error"
      title="加载失败"
      :body="error"
      tone="error"
    />
    <ConsolePageSkeleton v-else-if="loading && !data" />

    <template v-else-if="data">
      <LegacyConfigView v-if="currentWorkspace === 'guild-config'" :navigation="props.navigation" />

      <div v-else class="sh-split sh-split--1-1">
        <WorkspaceSection
          v-if="currentWorkspace === 'templates'"
          title="模板库"
          description="模板定义 guard 的禁言时长、踢出阈值、提醒文案与豁免名单。"
          :meta="`${configModel?.templateRows.length ?? 0} 条`"
          flush
        >
          <EmptyState
            v-if="(configModel?.templateRows.length ?? 0) === 0"
            title="暂无模板"
            body="在右侧创建新模板,填写 ID 与默认禁言时长后保存。"
          />
          <div v-else class="sh-lane">
            <button
              v-for="item in configModel?.templateRows ?? []"
              :key="item.id"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{ 'sh-lane__row--active': templateForm.id === item.id }"
              @click="loadTemplate(item)"
            >
              <span class="sh-lane__dot sh-lane__dot--primary"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title">{{ item.name }}</div>
                <div class="sh-lane__subtitle">
                  <span class="sh-mono">{{ item.id }}</span>
                  · {{ item.muteDurationSeconds }} 秒
                  · {{ item.kickAfterMinutes }} 分钟
                </div>
              </div>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          v-else-if="currentWorkspace === 'bindings'"
          title="群绑定"
          description="群到模板的映射关系。Guard 生效策略:优先群绑定模板,再退回静态 fallback。"
          :meta="`${configModel?.bindingRows.length ?? 0} 条`"
          flush
        >
          <EmptyState
            v-if="(configModel?.bindingRows.length ?? 0) === 0"
            title="暂无群绑定"
            body="在右侧选择目标群和模板,保存绑定后治理策略会立即生效。"
          />
          <div v-else class="sh-lane">
            <button
              v-for="item in configModel?.bindingRows ?? []"
              :key="item.id"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{
                'sh-lane__row--active':
                  bindingForm.guildId === item.guildId && bindingForm.platform === item.platform,
              }"
              @click="loadBinding(item)"
            >
              <span class="sh-lane__dot sh-lane__dot--primary"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title sh-mono">{{ item.guildId }}</div>
                <div class="sh-lane__subtitle">
                  <span class="sh-mono">{{ item.platform }}</span>
                  · {{ item.effectiveTemplateName }}
                  · {{ item.note || '无备注' }}
                </div>
              </div>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          v-else
          title="命令策略"
          description="全局命令权限与支持的命令列表。CommandPolicy 不参与群配置值覆盖链。"
          :meta="`${configModel?.policyRows.length ?? 0} 条`"
          flush
        >
          <EmptyState
            v-if="(configModel?.policyRows.length ?? 0) === 0"
            title="暂无命令策略"
            body="在右侧选择要治理的命令,设置最小 authority 与角色白名单后保存。"
          />
          <div v-else class="sh-lane">
            <button
              v-for="item in configModel?.policyRows ?? []"
              :key="item.commandId"
              type="button"
              class="sh-lane__row sh-lane__row--interactive"
              :class="{ 'sh-lane__row--active': policyForm.commandId === item.commandId }"
              @click="loadPolicy(item)"
            >
              <span class="sh-lane__dot sh-lane__dot--primary"></span>
              <div class="sh-lane__body">
                <div class="sh-lane__title sh-mono">{{ item.commandId }}</div>
                <div class="sh-lane__subtitle">
                  authority {{ item.minAuthority }}
                  · {{ item.roles.join(', ') || '无角色限制' }}
                </div>
              </div>
            </button>
          </div>
        </WorkspaceSection>

        <WorkspaceSection
          v-if="currentWorkspace === 'templates'"
          title="编辑模板"
          description="模板 ID、提醒文案与白名单直接保存到 guard policy store。"
        >
          <div class="sh-form-grid">
            <label class="sh-field">
              <span class="sh-field__label">模板 ID</span>
              <el-input v-model.trim="templateForm.id" class="sh-control sh-control--mono" placeholder="template-id" />
            </label>
            <label class="sh-field">
              <span class="sh-field__label">模板名称</span>
              <el-input v-model.trim="templateForm.name" class="sh-control" placeholder="对外可读名称" />
            </label>
            <label class="sh-field">
              <span class="sh-field__label">禁言时长(秒)</span>
              <el-input-number v-model="templateForm.muteDurationSeconds" class="sh-control" :min="0" />
            </label>
            <label class="sh-field">
              <span class="sh-field__label">踢出阈值(分钟)</span>
              <el-input-number v-model="templateForm.kickAfterMinutes" class="sh-control" :min="0" />
            </label>
          </div>
          <label class="sh-field">
            <span class="sh-field__label">提醒文案</span>
            <el-input v-model.trim="templateForm.reminderTemplate" class="sh-control" placeholder="向成员发送的提醒" />
          </label>
          <label class="sh-field">
            <span class="sh-field__label">豁免名单(逗号分隔)</span>
            <el-input v-model.trim="templateForm.exemptUsersText" class="sh-control sh-control--mono" placeholder="12345, 67890" />
          </label>
          <div class="sh-btn-row sh-form__actions">
            <el-checkbox v-model="templateForm.enabled" class="sh-check">启用模板</el-checkbox>
            <div class="sh-btn-row__spacer"></div>
            <el-button
              type="primary"
              class="sh-button sh-button--primary"
              :disabled="submittingTemplate"
              @click="submitTemplate"
            >
              {{ submittingTemplate ? '保存中…' : '保存模板' }}
            </el-button>
          </div>
          <p v-if="currentDirty" class="sh-field__hint">有未提交更改</p>
          <p v-if="notice" class="sh-field__hint sh-notice-hint">{{ notice }}</p>
        </WorkspaceSection>

        <WorkspaceSection
          v-else-if="currentWorkspace === 'bindings'"
          title="编辑绑定"
          description="把群号绑定到模板,立即改变 guard 行为。"
        >
          <div class="sh-form-grid">
            <label class="sh-field">
              <span class="sh-field__label">平台</span>
              <el-input v-model.trim="bindingForm.platform" class="sh-control sh-control--mono" placeholder="qq" />
            </label>
            <label class="sh-field">
              <span class="sh-field__label">群号</span>
              <el-input v-model.trim="bindingForm.guildId" class="sh-control sh-control--mono" placeholder="12345678" />
            </label>
          </div>
          <label class="sh-field">
            <span class="sh-field__label">模板</span>
            <el-select v-model="bindingForm.templateId" class="sh-control" placeholder="选择模板" clearable>
              <el-option
                v-for="item in configModel?.templateRows ?? []"
                :key="item.id"
                :label="`${item.name} (${item.id})`"
                :value="item.id"
              />
            </el-select>
          </label>
          <label class="sh-field">
            <span class="sh-field__label">备注</span>
            <el-input v-model.trim="bindingForm.note" class="sh-control" placeholder="记录绑定原因" />
          </label>
          <div class="sh-btn-row sh-form__actions">
            <el-checkbox v-model="bindingForm.enabled" class="sh-check">启用绑定</el-checkbox>
            <div class="sh-btn-row__spacer"></div>
            <el-button
              type="primary"
              class="sh-button sh-button--primary"
              :disabled="submittingBinding"
              @click="submitBinding"
            >
              {{ submittingBinding ? '保存中…' : '保存绑定' }}
            </el-button>
          </div>
          <p v-if="currentDirty" class="sh-field__hint">有未提交更改</p>
          <p v-if="notice" class="sh-field__hint sh-notice-hint">{{ notice }}</p>
        </WorkspaceSection>

        <WorkspaceSection
          v-else
          title="编辑命令策略"
          description="命令与 authority / 角色白名单的全局控制。"
        >
          <label class="sh-field">
            <span class="sh-field__label">命令</span>
            <el-select v-model="policyForm.commandId" class="sh-control sh-control--mono" placeholder="选择命令" clearable>
              <el-option
                v-for="commandId in configModel?.supportedCommandIds ?? []"
                :key="commandId"
                :label="commandId"
                :value="commandId"
              />
            </el-select>
          </label>
          <div class="sh-form-grid">
            <label class="sh-field">
              <span class="sh-field__label">最小 authority</span>
              <el-input-number v-model="policyForm.minAuthority" class="sh-control" :min="0" :max="5" />
            </label>
            <label class="sh-field">
              <span class="sh-field__label">角色白名单(逗号分隔)</span>
              <el-input v-model.trim="policyForm.rolesText" class="sh-control sh-control--mono" placeholder="moderator, admin" />
            </label>
          </div>
          <div class="sh-btn-row sh-form__actions">
            <div class="sh-btn-row__spacer"></div>
            <el-button
              type="primary"
              class="sh-button sh-button--primary"
              :disabled="submittingPolicy"
              @click="submitPolicy"
            >
              {{ submittingPolicy ? '保存中…' : '保存策略' }}
            </el-button>
          </div>
          <p class="sh-field__hint">
            支持命令:{{ configModel?.supportedCommandIds.join(', ') || '—' }}
          </p>
          <p v-if="currentDirty" class="sh-field__hint">有未提交更改</p>
          <p v-if="notice" class="sh-field__hint sh-notice-hint">{{ notice }}</p>
        </WorkspaceSection>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { formatTimestamp } from '../models/formatters'
import { useConfigGovernance } from '../composables/use-config-governance'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'
import LegacyConfigView from './ConfigView.vue'

const props = defineProps<{ navigation?: ConsoleNavigationController }>()
const {
  loading,
  error,
  data,
  currentWorkspace,
  notice,
  submittingTemplate,
  submittingBinding,
  submittingPolicy,
  templateForm,
  bindingForm,
  policyForm,
  configModel,
  currentDirty,
  loadData,
  selectWorkspace,
  loadTemplate,
  loadBinding,
  loadPolicy,
  submitTemplate,
  submitBinding,
  submitPolicy,
} = useConfigGovernance(props.navigation)
</script>

<style scoped>
.sh-subnav {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  background: var(--sh-surface-1);
  border: 1px solid var(--sh-border);
  border-radius: var(--sh-r-2);
  align-self: flex-start;
  overflow-x: auto;
  max-width: 100%;
}

.sh-subnav__tab {
  display: inline-flex;
  align-items: center;
  min-height: 30px;
  padding: 0 14px;
  border: 0;
  border-radius: var(--sh-r-1);
  background: transparent;
  color: var(--sh-fg-2);
  font-size: var(--sh-t-meta);
  font-weight: var(--sh-w-medium);
  cursor: pointer;
  white-space: nowrap;
  transition: background var(--sh-dur-fast) var(--sh-ease),
    color var(--sh-dur-fast) var(--sh-ease);
}

.sh-subnav__tab:hover {
  color: var(--sh-fg);
}

.sh-subnav__tab--active {
  background: var(--sh-surface-0);
  color: var(--sh-fg);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06), 0 0 0 1px var(--sh-border);
}

.sh-form__actions {
  align-items: center;
}

.sh-btn-row__spacer {
  flex: 1;
}

.sh-notice-hint {
  color: var(--sh-success);
}
</style>
