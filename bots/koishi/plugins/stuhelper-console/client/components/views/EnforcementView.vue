<template>
  <div id="sh-view-enforcement" class="sh-view" role="tabpanel">
    <header class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">ENFORCEMENT / 处置中心</span>
        <h1 class="sh-view__title">人工复核与举报处理</h1>
        <p class="sh-view__lead">
          高风险动作在这里执行或驳回，最近举报也会一起汇总，避免在多个入口之间切换。
        </p>
      </div>
    </header>

    <div class="sh-split sh-split--7-5">
      <Section
        eyebrow="Review"
        title="人工复核队列"
        description="踢人、拉黑等高风险动作只在这里执行或驳回。"
        :meta="`${pendingReviews.length} 条`"
        flush
      >
        <div class="sh-section__body">
          <label class="sh-field">
            <span class="sh-field__label">复核备注</span>
            <input v-model="reviewForm.note" class="sh-input" placeholder="记录这次决策依据" />
          </label>
        </div>

        <EmptyState
          v-if="pendingReviews.length === 0"
          title="当前没有待复核动作"
          body="高风险动作会自动流转到这里。"
        />
        <div v-else class="sh-table-shell">
          <table class="sh-table">
            <thead>
              <tr>
                <th>成员</th>
                <th>动作</th>
                <th>原因</th>
                <th>时间</th>
                <th>处理</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="review in pendingReviews"
                :key="review.id"
                data-clickable="true"
                @click="inspectReview(review)"
              >
                <td>{{ review.memberId }}</td>
                <td>
                  <SeverityTag
                    :label="reviewAction(review.actionType).label"
                    :intent="reviewAction(review.actionType).intent"
                  />
                </td>
                <td>{{ review.reason }}</td>
                <td class="sh-table__mono">{{ formatTimestamp(review.createdAt) }}</td>
                <td class="sh-table__actions">
                  <button
                    class="sh-btn sh-btn--primary sh-btn--sm"
                    @click.stop="runTask(() => submitReviewAction(review.id, 'execute'))"
                  >
                    执行
                  </button>
                  <button
                    class="sh-btn sh-btn--ghost sh-btn--sm"
                    @click.stop="runTask(() => submitReviewAction(review.id, 'reject'))"
                  >
                    驳回
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Section>

      <Section
        eyebrow="Reports"
        title="最近举报"
        description="举报流可以和复核队列并排处理，减少在多个入口间切换。"
        :meta="`${recentReports.length} 条`"
        flush
      >
        <EmptyState v-if="recentReports.length === 0" title="暂无举报" body="用户举报会显示在这里。" />
        <div v-else class="sh-table-shell">
          <table class="sh-table">
            <thead>
              <tr>
                <th>举报人</th>
                <th>目标</th>
                <th>AI</th>
                <th>原因</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="report in recentReports.slice(0, 10)"
                :key="report.id"
                data-clickable="true"
                @click="inspectReport(report)"
              >
                <td>{{ report.reporterMemberId }}</td>
                <td>{{ report.targetMemberId }}</td>
                <td>
                  <SeverityTag
                    :label="`${report.aiStatus}/${report.aiSeverity}`"
                    :intent="describeLevel(report.aiSeverity)"
                  />
                </td>
                <td>{{ report.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </Section>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { StuhelperConsoleReport, StuhelperConsoleReview } from '../../../src/console-types'
import Section from '../ConsolePanel.vue'
import EmptyState from '../EmptyState.vue'
import SeverityTag from '../SeverityTag.vue'
import { describeAction, describeLevel, formatTimestamp } from '../../use-console-page'

defineProps<{
  pendingReviews: readonly StuhelperConsoleReview[]
  recentReports: readonly StuhelperConsoleReport[]
  reviewForm: { note: string }
  runTask: (task: () => Promise<unknown>) => Promise<unknown>
  submitReviewAction: (reviewId: string, action: 'execute' | 'reject') => Promise<unknown>
  inspectReview: (review: StuhelperConsoleReview) => void
  inspectReport: (report: StuhelperConsoleReport) => void
}>()

function reviewAction(action: string) {
  return describeAction(action === 'kick_and_block' ? 'kick-permanent' : action)
}
</script>
