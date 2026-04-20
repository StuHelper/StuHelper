<template>
  <div id="sh-view-audit" class="sh-view" role="tabpanel">
    <header class="sh-view__header">
      <div class="sh-view__title-group">
        <span class="sh-view__eyebrow">AUDIT / 审计检索</span>
        <h1 class="sh-view__title">事件与举报检索</h1>
        <p class="sh-view__lead">按事件和举报两个方向检索最近记录，点击行可直接打开详情抽屉。</p>
      </div>
    </header>

    <div class="sh-split sh-split--1-1">
      <Section
        eyebrow="Event"
        title="事件日志"
        description="事件流用于回溯自动处罚、人工动作和系统异常。"
        :meta="`${filteredEvents.length} 条`"
        flush
      >
        <div class="sh-toolbar">
          <input v-model="eventSearch.value" class="sh-input sh-input--mono" placeholder="搜索成员 / 摘要 / 群号" />
        </div>

        <EmptyState
          v-if="filteredEvents.length === 0"
          title="没有匹配的事件"
          body="调整搜索词后重试。"
        />
        <div v-else class="sh-table-shell">
          <table class="sh-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>级别</th>
                <th>成员</th>
                <th>摘要</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="event in filteredEvents"
                :key="event.id"
                data-clickable="true"
                @click="inspectEvent(event)"
              >
                <td class="sh-table__mono">{{ formatTimestamp(event.createdAt) }}</td>
                <td>
                  <SeverityTag :label="event.level" :intent="describeLevel(event.level)" />
                </td>
                <td>{{ event.memberId }}</td>
                <td>{{ event.summary }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </Section>

      <Section
        eyebrow="Report"
        title="举报日志"
        description="按举报人、目标和 AI 摘要检索最近的举报记录。"
        :meta="`${filteredReports.length} 条`"
        flush
      >
        <div class="sh-toolbar">
          <input v-model="reportSearch.value" class="sh-input sh-input--mono" placeholder="搜索举报人 / 目标 / 原因" />
        </div>

        <EmptyState
          v-if="filteredReports.length === 0"
          title="没有匹配的举报"
          body="调整关键词后重试。"
        />
        <div v-else class="sh-table-shell">
          <table class="sh-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>举报人</th>
                <th>目标</th>
                <th>AI</th>
                <th>原因</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="report in filteredReports"
                :key="report.id"
                data-clickable="true"
                @click="inspectReport(report)"
              >
                <td class="sh-table__mono">{{ formatTimestamp(report.createdAt) }}</td>
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
import type { StuhelperConsoleEvent, StuhelperConsoleReport } from '../../../src/console-types'
import Section from '../ConsolePanel.vue'
import EmptyState from '../EmptyState.vue'
import SeverityTag from '../SeverityTag.vue'
import { describeLevel, formatTimestamp } from '../../use-console-page'

defineProps<{
  filteredEvents: readonly StuhelperConsoleEvent[]
  filteredReports: readonly StuhelperConsoleReport[]
  inspectEvent: (event: StuhelperConsoleEvent) => void
  inspectReport: (report: StuhelperConsoleReport) => void
}>()

const eventSearch = defineModel<string>('eventSearch', { required: true })
const reportSearch = defineModel<string>('reportSearch', { required: true })
</script>
