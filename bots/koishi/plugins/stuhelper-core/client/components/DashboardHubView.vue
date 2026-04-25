<template>
  <div class="sh-view sh-dashboard">
    <header class="sh-dashboard__head">
      <div class="sh-dashboard__head-copy">
        <h1 class="sh-dashboard__title">控制台总览</h1>
        <p class="sh-dashboard__description">
          看板 · 待办 · 分发台 — 只做摘要与跳转,复杂处置进入目标页完成。
        </p>
        <div class="sh-dashboard__meta">
          <span class="sh-meta-chip">StuHelper 群管中心</span>
          <span v-if="dashboardData" class="sh-meta-chip sh-mono">
            页面 · {{ formatTimestamp(dashboardData.generatedAt) }}
          </span>
          <span v-if="stats.timestamp" class="sh-meta-chip sh-mono">
            统计 · {{ formatTimestamp(stats.timestamp) }}
          </span>
        </div>
      </div>
      <div class="sh-dashboard__actions">
        <el-button
          class="sh-button sh-button--ghost"
          :disabled="loading"
          @click="loadData"
        >
          {{ loading ? '刷新中…' : '刷新' }}
        </el-button>
      </div>
    </header>

    <ConsolePageSkeleton v-if="loading && !dashboardData" />
    <EmptyState
      v-else-if="error"
      title="加载失败"
      :body="error"
      tone="error"
    />
    <template v-else-if="dashboardData">
      <section class="sh-dashboard-metrics">
        <article
          v-for="(metric, index) in baseMetrics"
          :key="metric.label"
          class="sh-stat sh-dashboard-metric"
          :class="baseIntent(index)"
        >
          <span class="sh-stat__label">{{ metric.label }}</span>
          <span class="sh-stat__value sh-num">{{ metric.value }}</span>
          <span class="sh-stat__note">{{ metric.note }}</span>
        </article>
      </section>

      <section class="sh-dashboard-metrics">
        <article
          v-for="metric in pendingMetrics"
          :key="metric.label"
          class="sh-stat sh-dashboard-metric"
          :class="pendingIntent(metric.value)"
        >
          <span class="sh-stat__label">{{ metric.label }}</span>
          <span class="sh-stat__value sh-num">{{ metric.value }}</span>
          <span class="sh-stat__note">{{ metric.note }}</span>
        </article>
      </section>

      <div class="sh-dashboard-grid">
        <div class="sh-dashboard__charts">
          <TrendChartCard :data="chartData.trend" :loading="chartLoading" />
          <DistChartCard :data="chartData.distribution" :loading="chartLoading" />
          <RankChartCard
            type="guild"
            title="群聊排行"
            icon="octicons.people"
            :data="chartData.guildRank"
            :loading="chartLoading"
          />
          <RankChartCard
            type="user"
            title="个人排行"
            icon="octicons.person"
            :data="chartData.userRank"
            :loading="chartLoading"
          />
        </div>

        <div class="sh-dashboard__side">
          <WorkspaceSection
            title="待处理事项"
            description="从总览直接进入身份认证或处置中心。"
            :meta="`${todoRows.length} 条`"
            flush
          >
            <div v-if="todoRows.length > 0" class="sh-lane">
              <button
                v-for="item in todoRows"
                :key="item.id"
                type="button"
                class="sh-lane__row sh-lane__row--interactive"
                @click="goToTarget(item.target)"
              >
                <span class="sh-lane__dot" :class="todoDotClass(item)"></span>
                <div class="sh-lane__body">
                  <div class="sh-lane__title">{{ item.title }}</div>
                  <div class="sh-lane__subtitle">{{ item.meta }}</div>
                </div>
                <span class="sh-lane__chevron" aria-hidden="true">›</span>
              </button>
            </div>
            <div v-else class="sh-dashboard-empty sh-section__body">
              暂无待处理事项。
            </div>
          </WorkspaceSection>

          <WorkspaceSection title="快捷入口" description="常用治理动作直达。">
            <div class="sh-shortcut-list">
              <button
                v-for="item in shortcuts"
                :key="item.label"
                type="button"
                class="sh-shortcut"
                @click="goToTarget(item.target)"
              >
                <span class="sh-shortcut__label">{{ item.label }}</span>
                <span class="sh-shortcut__description">{{ item.description }}</span>
              </button>
            </div>
          </WorkspaceSection>

          <WorkspaceSection title="系统状态" description="核心模块运行状况。">
            <div class="sh-dashboard-status-list">
              <article
                v-for="item in dashboardData.systemStatus"
                :key="item.name"
                class="sh-dashboard-status-list__item"
              >
                <div>
                  <div class="sh-dashboard-status-list__label">
                    {{ item.description }}
                  </div>
                  <div v-if="item.error" class="sh-dashboard-status-list__note">
                    {{ item.error }}
                  </div>
                </div>
                <SeverityTag :label="item.state" :intent="statusIntent(item.state)" />
              </article>
            </div>
          </WorkspaceSection>

          <WorkspaceSection
            title="最近事件与举报"
            description="快速判断是否需要进入处置中心。"
            flush
          >
            <div v-if="activityRows.length > 0" class="sh-lane">
              <div v-for="item in activityRows" :key="item.id" class="sh-lane__row">
                <span class="sh-lane__dot" :class="activityDotClass(item)"></span>
                <div class="sh-lane__body">
                  <div class="sh-lane__title">{{ item.title }}</div>
                  <div class="sh-lane__subtitle">{{ item.meta }}</div>
                </div>
              </div>
            </div>
            <div v-else class="sh-dashboard-empty sh-section__body">
              暂无最近事件。
            </div>
          </WorkspaceSection>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'

import type { ConsoleNavigationController } from '../composables/use-console-navigation'
import { consolePageApi } from '../page-api'
import type { DashboardPageData } from '../page-types'
import { statsApi, type ChartData } from '../api'
import { formatTimestamp } from '../models/formatters'
import {
  buildDashboardModel,
  type DashboardActivityRow,
  type DashboardTodoRow,
} from '../models/dashboard'
import ConsolePageSkeleton from './primitives/ConsolePageSkeleton.vue'
import EmptyState from './primitives/EmptyState.vue'
import SeverityTag, { type TagIntent } from './primitives/SeverityTag.vue'
import WorkspaceSection from './primitives/WorkspaceSection.vue'
import DistChartCard from './dashboard/DistChartCard.vue'
import RankChartCard from './dashboard/RankChartCard.vue'
import TrendChartCard from './dashboard/TrendChartCard.vue'

const props = defineProps<{
  navigation?: ConsoleNavigationController
}>()

const loading = ref(false)
const chartLoading = ref(false)
const error = ref('')
const dashboardData = ref<DashboardPageData | null>(null)
const stats = reactive({
  totalGroups: 0,
  totalWarns: 0,
  totalBlacklisted: 0,
  totalSubscriptions: 0,
  timestamp: 0,
})
const chartData = reactive<ChartData>({
  trend: [],
  distribution: [],
  successRate: { success: 0, fail: 0 },
  guildRank: [],
  userRank: [],
})
const dashboardModel = computed(() => (dashboardData.value ? buildDashboardModel(dashboardData.value) : null))

const baseMetrics = computed(() => [
  { label: '已配置群组', value: stats.totalGroups, note: '当前已有治理配置的群组总数。' },
  { label: '警告记录', value: stats.totalWarns, note: '警告统计来自现有日志与计数器。' },
  { label: '黑名单用户', value: stats.totalBlacklisted, note: '当前已加入黑名单的成员数。' },
  { label: '活跃订阅', value: stats.totalSubscriptions, note: '订阅推送仍沿用现有模块。' },
])

const pendingMetrics = computed(() => [
  { label: '待复核', value: dashboardData.value?.overview.pendingReviews ?? 0, note: '需要人工执行或驳回的复核动作。' },
  { label: '待认证', value: dashboardData.value?.overview.pendingAdmissions ?? 0, note: '仍处于入群限制中的成员。' },
  { label: '最近举报', value: dashboardData.value?.overview.openReports ?? 0, note: '需要结合上下文继续判断的举报项。' },
  { label: '策略项', value: dashboardData.value?.overview.policyItems ?? 0, note: '模板、绑定与命令策略总量。' },
])

const todoRows = computed(() => dashboardModel.value?.todoRows ?? [])
const shortcuts = computed(() => dashboardModel.value?.shortcuts ?? [])
const activityRows = computed(() => dashboardModel.value?.activityRows ?? [])

onMounted(loadData)

async function loadData() {
  loading.value = true
  chartLoading.value = true
  error.value = ''
  try {
    const [pageData, dashboardStats, charts] = await Promise.all([
      consolePageApi.dashboard(),
      statsApi.dashboard(),
      statsApi.charts(7),
    ])
    dashboardData.value = pageData
    Object.assign(stats, dashboardStats)
    Object.assign(chartData, charts)
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
    chartLoading.value = false
  }
}

function goToTarget(target: {
  view: 'review' | 'identity' | 'config'
  workspace: string
  guildId: string | null
  memberId: string | null
  itemId: string | null
  keyword: string
}) {
  props.navigation?.jumpTo(target)
}

function statusIntent(state: string): TagIntent {
  if (state === 'loaded') return 'success'
  if (state === 'error') return 'danger'
  if (state === 'loading') return 'info'
  return 'neutral'
}

function baseIntent(index: number): string {
  return index === 0 ? 'sh-stat--primary' : ''
}

function pendingIntent(value: number): string {
  if (value === 0) return ''
  if (value > 20) return 'sh-stat--danger'
  if (value > 5) return 'sh-stat--warning'
  return 'sh-stat--primary'
}

function todoDotClass(item: DashboardTodoRow): string {
  if (item.target.view === 'identity') return 'sh-lane__dot--warning'
  return 'sh-lane__dot--primary'
}

function activityDotClass(item: DashboardActivityRow): string {
  return item.meta.startsWith('举报') ? 'sh-lane__dot--warning' : 'sh-lane__dot--primary'
}
</script>

<style scoped>
.sh-dashboard__charts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--sh-s-3);
}

.sh-dashboard__charts > * {
  min-width: 0;
}

.sh-dashboard__side {
  display: flex;
  flex-direction: column;
  gap: var(--sh-s-3);
  min-width: 0;
}

.sh-lane__row--interactive:focus-visible {
  outline: none;
  background: var(--sh-primary-soft);
}

@media (max-width: 1080px) {
  .sh-dashboard__charts {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
