<template>
  <div id="sh-view-overview" class="sh-view" role="tabpanel">
    <header class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">OVERVIEW / 总览</span>
        <h1 class="sh-view__title">今日群管中心状态</h1>
        <p class="sh-view__lead">待办与风险集中在这里，快速跳转到对应工作流处理。</p>
      </div>
      <div class="sh-view__toolbar">
        <span v-if="generatedAt" class="sh-toolbar__count sh-mono">
          同步于 {{ formatTimestamp(generatedAt) }}
        </span>
      </div>
    </header>

    <div class="sh-split sh-split--7-5">
      <Section
        eyebrow="Recent activity"
        title="最近事件"
        description="自动处罚和人工决策会在这里首先出现；点击跳转到事件日志。"
        :meta="`${recentEvents.length} 条`"
        flush
      >
        <div v-if="recentEvents.length === 0" class="sh-empty">
          <p class="sh-empty__title">暂无事件</p>
          <p class="sh-empty__body">当群管引擎命中规则或执行动作时，会自动写入事件流。</p>
        </div>
        <div v-else class="sh-lane">
          <div
            v-for="event in recentEvents.slice(0, 8)"
            :key="event.id"
            class="sh-lane__row"
          >
            <span class="sh-lane__dot" :class="dotClass(event.level)"></span>
            <div>
              <div class="sh-lane__title">{{ event.summary || event.level }}</div>
              <div class="sh-lane__subtitle sh-mono">
                {{ event.memberId || '—' }} · {{ event.guildId || '系统' }}
              </div>
            </div>
            <span class="sh-lane__time">{{ formatTimestamp(event.createdAt) }}</span>
          </div>
        </div>
        <div v-if="recentEvents.length > 0" class="sh-section__body">
          <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="goto('events')">
            进入事件日志 →
          </button>
        </div>
      </Section>

      <Section
        eyebrow="Reports"
        title="最近举报"
        description="举报单进入 AI 审核；高风险等级进入人工复核队列。"
        :meta="`${recentReports.length} 条`"
        flush
      >
        <div v-if="recentReports.length === 0" class="sh-empty">
          <p class="sh-empty__title">暂无举报</p>
          <p class="sh-empty__body">用户提交举报后会出现在这里。</p>
        </div>
        <div v-else class="sh-lane">
          <div
            v-for="report in recentReports.slice(0, 8)"
            :key="report.id"
            class="sh-lane__row"
          >
            <span class="sh-lane__dot" :class="dotClass(report.aiSeverity)"></span>
            <div>
              <div class="sh-lane__title">
                {{ report.reporterMemberId }} → {{ report.targetMemberId }}
              </div>
              <div class="sh-lane__subtitle">{{ report.aiSummary || report.reason }}</div>
            </div>
            <span class="sh-lane__time">{{ formatTimestamp(report.createdAt) }}</span>
          </div>
        </div>
        <div v-if="recentReports.length > 0" class="sh-section__body">
          <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="goto('events')">
            查看全部举报 →
          </button>
        </div>
      </Section>
    </div>

    <Section
      eyebrow="Shortcuts"
      title="快捷入口"
      description="按 spec 的 6 个一级分区直接跳转，不必在页面里找位置。"
    >
      <div class="sh-btn-row">
        <button class="sh-btn" @click="goto('gate')">认证准入 ↗</button>
        <button class="sh-btn" @click="goto('rules')">通用群规 ↗</button>
        <button class="sh-btn" @click="goto('templates')">模板与群绑定 ↗</button>
        <button class="sh-btn" @click="goto('enforcement')">处置中心 ↗</button>
        <button class="sh-btn" @click="goto('events')">事件日志 ↗</button>
      </div>
    </Section>
  </div>
</template>

<script setup lang="ts">
import Section from '../ConsolePanel.vue'
import type { StuhelperConsoleEvent, StuhelperConsoleReport } from '../../../src/console-types'
import { describeLevel, formatTimestamp, type ViewId } from '../../use-console-page'

defineProps<{
  recentEvents: readonly StuhelperConsoleEvent[]
  recentReports: readonly StuhelperConsoleReport[]
  generatedAt: string
  goto: (view: ViewId) => void
}>()

function dotClass(level?: string | null) {
  const intent = describeLevel(level || '')
  if (intent === 'neutral' || intent === 'muted') return ''
  return `sh-lane__dot--${intent}`
}
</script>
