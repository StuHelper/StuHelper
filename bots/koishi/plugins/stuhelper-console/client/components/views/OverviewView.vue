<template>
  <div id="sh-view-dashboard" class="sh-view" role="tabpanel">
    <div class="sh-split sh-split--7-5">
      <Section title="当前概览" description="待认证与待复核会从这里分流到对应工作区。">
        <div class="sh-lane">
          <div class="sh-lane__row">
            <span class="sh-lane__dot" :class="pendingMemberDotClass"></span>
            <div>
              <div class="sh-lane__title">待认证成员</div>
              <div class="sh-lane__subtitle">{{ pendingMemberSummary }}</div>
            </div>
            <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="goToSection('identity')">
              进入
            </button>
          </div>
          <div class="sh-lane__row">
            <span class="sh-lane__dot" :class="pendingReviewDotClass"></span>
            <div>
              <div class="sh-lane__title">待复核动作</div>
              <div class="sh-lane__subtitle">{{ pendingReviewSummary }}</div>
            </div>
            <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="goToSection('enforcement')">
              进入
            </button>
          </div>
        </div>
      </Section>

      <Section title="快捷入口" description="按工作流进入身份、处置、策略与审计分区。">
        <div class="sh-btn-row">
          <button class="sh-btn" @click="goToSection('identity')">
            身份认证
          </button>
          <button class="sh-btn" @click="goToSection('enforcement')">
            处置中心
          </button>
          <button class="sh-btn" @click="goToSection('policy')">
            策略中心
          </button>
          <button class="sh-btn" @click="goToSection('audit')">
            审计检索
          </button>
        </div>
      </Section>
    </div>

    <div class="sh-split sh-split--1-1">
      <Section title="最近事件" :meta="`${recentEvents.length} 条`" flush>
        <EmptyState
          v-if="visibleEvents.length === 0"
          title="暂无事件"
          body="事件流会在命中规则、执行动作或人工决策后更新。"
        />
        <div v-else class="sh-lane">
          <div v-for="event in visibleEvents" :key="event.id" class="sh-lane__row">
            <span class="sh-lane__dot" :class="dotClass(event.level)"></span>
            <div>
              <div class="sh-lane__title">{{ event.summary || event.type || '未命名事件' }}</div>
              <div class="sh-lane__subtitle sh-mono">
                {{ event.memberId || '—' }} · {{ event.guildId || '系统' }}
              </div>
            </div>
            <button
              class="sh-btn sh-btn--ghost sh-btn--sm"
              @click="goToSection('audit')"
            >
              进入
            </button>
          </div>
        </div>
      </Section>

      <Section title="最近举报" :meta="`${recentReports.length} 条`" flush>
        <EmptyState
          v-if="visibleReports.length === 0"
          title="暂无举报"
          body="用户提交举报后会出现在这里，并同步进入审计检索。"
        />
        <div v-else class="sh-lane">
          <div v-for="report in visibleReports" :key="report.id" class="sh-lane__row">
            <span class="sh-lane__dot" :class="dotClass(report.aiSeverity)"></span>
            <div>
              <div class="sh-lane__title">
                {{ report.reporterMemberId || '匿名举报' }} → {{ report.targetMemberId || '未知目标' }}
              </div>
              <div class="sh-lane__subtitle">{{ report.aiSummary || report.reason || '无摘要' }}</div>
            </div>
            <button
              class="sh-btn sh-btn--ghost sh-btn--sm"
              @click="goToSection('audit')"
            >
              进入
            </button>
          </div>
        </div>
      </Section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type {
  StuhelperConsoleEvent,
  StuhelperConsoleGuardMember,
  StuhelperConsoleReport,
  StuhelperConsoleReview,
} from '../../../src/console-types'
import type { ConsoleSectionId } from '../../sections'
import { describeLevel, formatTimestamp } from '../../use-console-page'
import Section from '../ConsolePanel.vue'
import EmptyState from '../EmptyState.vue'

const ITEM_LIMIT = 6

const props = defineProps<{
  pendingMembers: readonly StuhelperConsoleGuardMember[]
  pendingReviews: readonly StuhelperConsoleReview[]
  recentEvents: readonly StuhelperConsoleEvent[]
  recentReports: readonly StuhelperConsoleReport[]
  openSection: (section: ConsoleSectionId) => void
}>()

const visibleEvents = computed(() => props.recentEvents.slice(0, ITEM_LIMIT))
const visibleReports = computed(() => props.recentReports.slice(0, ITEM_LIMIT))
const pendingMemberSummary = computed(() => describePendingMembers(props.pendingMembers))
const pendingReviewSummary = computed(() => describePendingReviews(props.pendingReviews))
const pendingMemberDotClass = computed(() => statusDotClass(props.pendingMembers.length, countOverdueMembers(props.pendingMembers)))
const pendingReviewDotClass = computed(() => statusDotClass(props.pendingReviews.length, props.pendingReviews.length))

function goToSection(section: ConsoleSectionId) {
  props.openSection(section)
}

function dotClass(level?: string | null) {
  const intent = describeLevel(level || '')
  if (intent === 'neutral' || intent === 'muted' || intent === 'info') return ''
  return `sh-lane__dot--${intent}`
}

function describePendingMembers(items: readonly StuhelperConsoleGuardMember[]) {
  if (items.length === 0) return '当前没有待认证成员。'

  const overdueCount = countOverdueMembers(items)
  if (overdueCount > 0) {
    return `当前 ${items.length} 条待处理，其中 ${overdueCount} 条已超时。`
  }

  const nextDeadline = findNearestDeadline(items)
  if (!nextDeadline) return `当前 ${items.length} 条待处理。`
  return `当前 ${items.length} 条待处理，最近截止 ${formatTimestamp(nextDeadline)}。`
}

function describePendingReviews(items: readonly StuhelperConsoleReview[]) {
  if (items.length === 0) return '当前没有待复核动作。'
  return `当前 ${items.length} 条待复核，最近提交 ${formatTimestamp(items[0]?.createdAt)}。`
}

function countOverdueMembers(items: readonly StuhelperConsoleGuardMember[]) {
  return items.filter((item) => item.verificationState === 'overdue').length
}

function findNearestDeadline(items: readonly StuhelperConsoleGuardMember[]) {
  let nearest: string | null = null
  let nearestTime = Number.POSITIVE_INFINITY

  for (const item of items) {
    const time = new Date(item.deadlineAt).getTime()
    if (Number.isNaN(time) || time >= nearestTime) continue
    nearest = item.deadlineAt
    nearestTime = time
  }

  return nearest
}

function statusDotClass(total: number, urgent: number) {
  if (urgent > 0) return 'sh-lane__dot--danger'
  if (total > 0) return 'sh-lane__dot--warning'
  return 'sh-lane__dot--success'
}
</script>
