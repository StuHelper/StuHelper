<template>
  <div class="sh-dashboard">
    <section class="sh-section sh-dashboard-status">
      <div class="sh-dashboard-status__grid">
        <article
          v-for="item in model.statusBand"
          :key="item.label"
          class="sh-dashboard-status__item"
          :data-tone="item.tone || 'neutral'"
        >
          <span class="sh-dashboard-status__label">{{ item.label }}</span>
          <strong class="sh-dashboard-status__value">{{ item.value }}</strong>
          <span class="sh-dashboard-status__note">{{ item.note }}</span>
        </article>
      </div>
    </section>

    <DashboardMetrics :metrics="model.metrics" />

    <div class="sh-dashboard-grid">
      <WorkspaceSection
        title="待处理事项主卡"
        description="首页只做跳转与定向，复杂处置统一进入目标页完成。"
        flush
      >
        <DashboardTodoTable :rows="model.todoRows" @open="openTarget" />
      </WorkspaceSection>

      <WorkspaceSection
        title="系统状态"
        description="观察数据同步、复核队列和身份认证积压。"
      >
        <div class="sh-dashboard-status-list">
          <article
            v-for="item in model.systemStatus"
            :key="item.label"
            class="sh-dashboard-status-list__item"
          >
            <div>
              <div class="sh-dashboard-status-list__label">{{ item.label }}</div>
              <div class="sh-dashboard-status-list__note">{{ item.note }}</div>
            </div>
            <SeverityTag :label="item.value" :intent="statusIntent(item.tone)" />
          </article>
        </div>
      </WorkspaceSection>
    </div>

    <WorkspaceSection
      title="快捷入口"
      description="固定进入常用工作区，不在首页承载复杂动作。"
    >
      <div class="sh-shortcut-list">
        <button
          v-for="item in model.shortcuts"
          :key="item.label"
          type="button"
          class="sh-shortcut"
          @click="emit('open-target', item.target)"
        >
          <span class="sh-shortcut__label">{{ item.label }}</span>
          <span class="sh-shortcut__description">{{ item.description }}</span>
        </button>
      </div>
    </WorkspaceSection>

    <WorkspaceSection
      title="最近事件"
      description="保留最近风险事件，便于快速回溯。"
      flush
    >
      <div v-if="model.recentEvents.length > 0" class="sh-lane">
        <div v-for="item in model.recentEvents" :key="item.id" class="sh-lane__row">
          <span class="sh-lane__dot" :class="activityDotClass(item.kind)"></span>
          <div>
            <div class="sh-lane__title">{{ item.title }}</div>
            <div class="sh-lane__subtitle">{{ item.meta }}</div>
          </div>
          <span class="sh-lane__time">{{ item.time }}</span>
        </div>
      </div>
      <div v-else class="sh-section__body">
        <div class="sh-dashboard-empty">当前没有最近事件。</div>
      </div>
    </WorkspaceSection>

    <div class="sh-dashboard-grid">
      <WorkspaceSection
        title="策略概况"
        description="快速查看各类策略对象的配置体量。"
      >
        <div class="sh-dashboard-policy">
          <div
            v-for="item in model.policySummary"
            :key="item.label"
            class="sh-dashboard-policy__row"
          >
            <span>{{ item.label }}</span>
            <strong class="sh-num">{{ item.value }}</strong>
          </div>
        </div>
      </WorkspaceSection>

      <WorkspaceSection
        title="最近变更"
        description="展示规则、模板、绑定和权限配置的最近更新。"
        flush
      >
        <div v-if="model.recentChanges.length > 0" class="sh-lane">
          <div v-for="item in model.recentChanges" :key="item.id" class="sh-lane__row">
            <SeverityTag :label="item.kindLabel" intent="neutral" />
            <div>
              <div class="sh-lane__title">{{ item.title }}</div>
              <div class="sh-lane__subtitle">{{ item.meta }}</div>
            </div>
            <span class="sh-lane__time">{{ item.time }}</span>
          </div>
        </div>
        <div v-else class="sh-section__body">
          <div class="sh-dashboard-empty">当前没有最近变更。</div>
        </div>
      </WorkspaceSection>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DashboardActivityRow, DashboardModel, DashboardStatusItem, DashboardTarget } from '../../dashboard/model'
import SeverityTag, { type TagIntent } from '../SeverityTag.vue'
import WorkspaceSection from '../layout/WorkspaceSection.vue'
import DashboardMetrics from './DashboardMetrics.vue'
import DashboardTodoTable from './DashboardTodoTable.vue'

defineProps<{
  model: DashboardModel
}>()

const emit = defineEmits<{
  'open-target': [target: DashboardTarget]
}>()

function openTarget(target: DashboardTarget) {
  emit('open-target', target)
}

function activityDotClass(kind: DashboardActivityRow['kind']) {
  return kind === 'report' ? 'sh-lane__dot--warning' : 'sh-lane__dot--primary'
}

function statusIntent(tone?: DashboardStatusItem['tone']): TagIntent {
  switch (tone) {
    case 'danger':
      return 'danger'
    case 'warning':
      return 'warning'
    case 'success':
      return 'success'
    case 'primary':
      return 'primary'
    case 'info':
      return 'info'
    default:
      return 'neutral'
  }
}
</script>
