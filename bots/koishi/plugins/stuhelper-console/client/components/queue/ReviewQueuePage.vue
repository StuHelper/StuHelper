<template>
  <div id="sh-view-enforcement" class="sh-view" role="tabpanel">
    <header class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">ENFORCEMENT / 处置中心</span>
        <h1 class="sh-view__title">人工复核与举报处理</h1>
        <p class="sh-view__lead">复核队列和最近举报保持同屏，点行即看详情，执行后自动切到下一条。</p>
      </div>
    </header>

    <div class="sh-split sh-split--7-5">
      <Section
        eyebrow="Review Queue"
        title="人工复核队列"
        description="踢人、拉黑等高风险动作只在这里执行或驳回。"
        :meta="`${pendingReviews.length} 条`"
      >
        <QueueToolbar :stats="reviewStats">
          <label class="sh-field sh-review-queue__field">
            <span class="sh-field__label">检索</span>
            <input v-model="search" class="sh-input" placeholder="成员 ID / 原因 / 动作" />
          </label>
          <label class="sh-field sh-review-queue__field sh-review-queue__field--wide">
            <span class="sh-field__label">复核备注</span>
            <input v-model="reviewForm.note" class="sh-input" placeholder="记录这次决策依据" />
          </label>
        </QueueToolbar>

        <QueueTable
          :columns="reviewColumns"
          :rows="reviewRows"
          :selected-id="selectedId"
          empty-title="当前没有待复核动作"
          empty-body="高风险动作会自动流转到这里。"
          @select="handleReviewSelect"
          @action="handleReviewAction"
        />
      </Section>

      <Section
        eyebrow="Reports"
        title="最近举报"
        description="保留侧栏表格，减少在多个入口之间切换。"
        :meta="`${reportRows.length} 条`"
      >
        <QueueTable
          :columns="reportColumns"
          :rows="reportRows"
          empty-title="暂无举报"
          empty-body="用户举报会显示在这里。"
          @select="handleReportSelect"
        />
      </Section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

import type { StuhelperConsoleReport, StuhelperConsoleReview } from '../../../src/console-types'
import Section from '../ConsolePanel.vue'
import QueueTable from './QueueTable.vue'
import QueueToolbar from './QueueToolbar.vue'
import { describeAction, describeLevel, formatTimestamp } from '../../use-console-page'

const REVIEW_COLUMNS = [
  { key: 'member', label: '成员' },
  { key: 'action', label: '动作' },
  { key: 'reason', label: '原因' },
  { key: 'createdAt', label: '时间' },
] as const

const REPORT_COLUMNS = [
  { key: 'reporter', label: '举报人' },
  { key: 'target', label: '目标' },
  { key: 'ai', label: 'AI' },
  { key: 'createdAt', label: '时间' },
] as const

const props = defineProps<{
  pendingReviews: readonly StuhelperConsoleReview[]
  recentReports: readonly StuhelperConsoleReport[]
  selectedId: string
  reviewForm: { note: string }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitReviewAndFocus: (reviewId: string, action: 'execute' | 'reject') => Promise<unknown>
  inspectReview: (review: StuhelperConsoleReview) => void
  inspectReport: (report: StuhelperConsoleReport) => void
}>()

const search = ref('')

const filteredReviews = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  if (!keyword) return props.pendingReviews
  return props.pendingReviews.filter((review) =>
    [review.memberId, review.reason, review.actionType]
      .filter(Boolean)
      .some((field) => String(field).toLowerCase().includes(keyword)),
  )
})

const reviewStats = computed(() => [
  { label: '待处理', value: props.pendingReviews.length },
  { label: '当前过滤', value: filteredReviews.value.length },
  { label: '选中', value: props.selectedId ? 1 : 0 },
])

const reviewColumns = REVIEW_COLUMNS
const reportColumns = REPORT_COLUMNS

const reviewRows = computed(() =>
  filteredReviews.value.map((review) => ({
    id: review.id,
    cells: {
      member: {
        text: review.memberId,
        secondary: review.id,
      },
      action: {
        text: reviewAction(review.actionType).label,
        tone: reviewAction(review.actionType).intent,
      },
      reason: review.reason || '—',
      createdAt: {
        text: formatTimestamp(review.createdAt),
        mono: true,
      },
    },
    actions: [
      { key: 'execute', label: '执行', tone: 'primary' as const },
      { key: 'reject', label: '驳回', tone: 'ghost' as const },
    ],
  })),
)

const reportRows = computed(() =>
  props.recentReports.slice(0, 10).map((report) => ({
    id: report.id,
    cells: {
      reporter: report.reporterMemberId,
      target: report.targetMemberId,
      ai: {
        text: `${report.aiStatus}/${report.aiSeverity}`,
        tone: describeLevel(report.aiSeverity),
      },
      createdAt: {
        text: formatTimestamp(report.createdAt),
        mono: true,
      },
    },
  })),
)

function reviewAction(action: string) {
  return describeAction(action === 'kick_and_block' ? 'kick-permanent' : action)
}

function handleReviewSelect(reviewId: string) {
  const review = props.pendingReviews.find((item) => item.id === reviewId)
  if (!review) return
  props.inspectReview(review)
}

function handleReportSelect(reportId: string) {
  const report = props.recentReports.find((item) => item.id === reportId)
  if (!report) return
  props.inspectReport(report)
}

function handleReviewAction(payload: { rowId: string; action: string }) {
  if (payload.action !== 'execute' && payload.action !== 'reject') return
  return props.runTask(() => props.submitReviewAndFocus(payload.rowId, payload.action))
}
</script>

<style scoped>
.sh-review-queue__field {
  min-width: 180px;
  flex: 0 1 220px;
}

.sh-review-queue__field--wide {
  flex-basis: 320px;
}
</style>
