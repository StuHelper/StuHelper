<template>
  <div class="console-view">
    <section class="console-header">
      <div>
        <h2 class="console-header__title">控制台总览</h2>
        <div class="console-header__meta">
          <span class="console-chip">StuHelper 群管中心</span>
          <span class="console-chip" v-if="dashboardData">页面数据 {{ formatTimestamp(dashboardData.generatedAt) }}</span>
          <span class="console-chip" v-if="stats.timestamp">统计数据 {{ formatTimestamp(stats.timestamp) }}</span>
        </div>
      </div>
      <div class="console-toolbar">
        <button class="console-button" @click="loadData">刷新</button>
      </div>
    </section>

    <div v-if="error" class="console-error">{{ error }}</div>
    <div v-else-if="loading" class="console-empty">正在加载总览数据…</div>
    <template v-else-if="dashboardData">
      <section class="console-metrics">
        <article v-for="metric in baseMetrics" :key="metric.label" class="console-metric">
          <div class="console-metric__label">{{ metric.label }}</div>
          <div class="console-metric__value">{{ metric.value }}</div>
          <div class="console-metric__note">{{ metric.note }}</div>
        </article>
      </section>

      <section class="console-metrics">
        <article v-for="metric in pendingMetrics" :key="metric.label" class="console-metric">
          <div class="console-metric__label">{{ metric.label }}</div>
          <div class="console-metric__value">{{ metric.value }}</div>
          <div class="console-metric__note">{{ metric.note }}</div>
        </article>
      </section>

      <section class="console-grid">
        <div class="console-stack">
          <div class="console-grid--cards">
            <TrendChartCard :data="chartData.trend" :loading="chartLoading" />
            <DistChartCard :data="chartData.distribution" :loading="chartLoading" />
            <RankChartCard type="guild" title="群聊排行" icon="octicons.people" :data="chartData.guildRank" :loading="chartLoading" />
            <RankChartCard type="user" title="个人排行" icon="octicons.person" :data="chartData.userRank" :loading="chartLoading" />
          </div>
        </div>

        <div class="console-stack">
          <section class="console-panel">
            <div>
              <h3 class="console-panel__title">待处理事项</h3>
              <div class="console-panel__subtitle">从总览直接进入身份认证或处置中心。</div>
            </div>
            <div class="console-list">
              <button
                v-for="item in todoRows"
                :key="item.id"
                class="console-list__item console-list__item--interactive"
                type="button"
                @click="goToTarget(item.target)"
              >
                <span class="console-list__title">{{ item.title }}</span>
                <span class="console-list__meta">{{ item.meta }}</span>
              </button>
            </div>
          </section>

          <section class="console-panel">
            <div>
              <h3 class="console-panel__title">快捷入口</h3>
              <div class="console-panel__subtitle">常用治理动作直达。</div>
            </div>
            <div class="console-list">
              <button
                v-for="item in shortcuts"
                :key="item.label"
                class="console-list__item console-list__item--interactive"
                type="button"
                @click="goToTarget(item.target)"
              >
                <span class="console-list__title">{{ item.label }}</span>
                <span class="console-list__meta">{{ item.description }}</span>
              </button>
            </div>
          </section>

          <section class="console-panel">
            <div>
              <h3 class="console-panel__title">系统状态</h3>
              <div class="console-panel__subtitle">核心模块运行状况。</div>
            </div>
            <div class="console-list">
              <div v-for="item in dashboardData.systemStatus" :key="item.name" class="console-list__item">
                <span class="console-list__title">{{ item.description }}</span>
                <span class="console-list__meta">
                  <span :class="statusClass(item.state)" class="console-chip">{{ item.state }}</span>
                  <span v-if="item.error"> · {{ item.error }}</span>
                </span>
              </div>
            </div>
          </section>

          <section class="console-panel">
            <div>
              <h3 class="console-panel__title">最近事件与举报</h3>
              <div class="console-panel__subtitle">帮助快速判断是否需要进入处置中心。</div>
            </div>
            <div class="console-list">
              <div v-for="item in activityRows" :key="item.id" class="console-list__item">
                <span class="console-list__title">{{ item.title }}</span>
                <span class="console-list__meta">{{ item.meta }}</span>
              </div>
            </div>
          </section>
        </div>
      </section>
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
import { buildDashboardModel } from '../models/dashboard'
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
const dashboardModel = computed(() => dashboardData.value ? buildDashboardModel(dashboardData.value) : null)

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

function statusClass(state: string) {
  if (state === 'loaded') return 'console-chip console-chip--success'
  if (state === 'error') return 'console-chip console-chip--danger'
  return 'console-chip'
}
</script>
