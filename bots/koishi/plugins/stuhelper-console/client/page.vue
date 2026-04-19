<template>
  <k-layout main="sh-console">
    <el-scrollbar class="sh-console__scroll" view-class="sh-console__view">
      <div class="sh-shell">
        <ConsoleHero :title="title" :generated-at="generatedAt" :loading="loading" @refresh="runTask(refresh)" />
        <section class="sh-stat-strip"><MetricCard v-for="item in metrics" :key="item.label" :label="item.label" :value="item.value" :note="item.note" :tone="item.tone" /></section>
        <Tabs v-model="activeTab" :items="tabs" />

        <section class="sh-view">
          <header class="sh-view__header">
            <div class="sh-view__title-group">
              <span class="sh-view__eyebrow">{{ viewMeta.eyebrow }}</span>
              <h1 class="sh-view__title">{{ viewMeta.title }}</h1>
              <p class="sh-view__lead">{{ viewMeta.lead }}</p>
            </div>
            <div class="sh-view__toolbar">
              <span class="sh-toolbar__count">更新时间 {{ generatedAt ? formatTimestamp(generatedAt) : '—' }}</span>
              <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="runTask(refresh)">刷新</button>
            </div>
          </header>

          <EmptyState v-if="!data" title="控制台数据尚未就绪" body="点击刷新，或确认 stuhelper-console 数据服务已经正确加载。" />

          <template v-else-if="activeTab === 'overview'">
            <div class="sh-split sh-split--7-5">
              <ConsolePanel eyebrow="Queue" title="待处理工作流" description="把准入、复核和举报入口压缩成一屏可扫描的工作台。">
                <div class="sh-lane">
                  <div v-for="item in overviewQueues" :key="item.tab" class="sh-lane__row">
                    <span class="sh-lane__dot" :class="`sh-lane__dot--${item.intent}`"></span>
                    <div><div class="sh-lane__title">{{ item.title }}</div><div class="sh-lane__subtitle">{{ item.subtitle }}</div></div>
                    <button class="sh-btn sh-btn--ghost sh-btn--sm" @click="goto(item.tab)">{{ item.count }}</button>
                  </div>
                </div>
              </ConsolePanel>
              <ConsolePanel eyebrow="Signal" title="最近高风险事件" description="高风险级别事件优先暴露，便于快速定位异常。">
                <div v-if="recentEvents.length" class="sh-lane">
                  <div v-for="event in recentEvents.slice(0, 6)" :key="event.id" class="sh-lane__row" @click="inspectEvent(event)">
                    <span class="sh-lane__dot" :class="`sh-lane__dot--${describeLevel(event.level)}`"></span>
                    <div><div class="sh-lane__title">{{ event.summary }}</div><div class="sh-lane__subtitle">{{ event.memberId }} / {{ event.guildId }}</div></div>
                    <span class="sh-lane__time">{{ formatTimestamp(event.createdAt) }}</span>
                  </div>
                </div>
                <EmptyState v-else title="最近没有异常事件" body="自动处罚、人工复核和系统事件会出现在这里。" />
              </ConsolePanel>
            </div>

            <div class="sh-split sh-split--1-1">
              <ConsolePanel eyebrow="Report" title="最近举报" description="AI 审核结果与人工处置入口在同一列表里汇总。">
                <div class="sh-table-shell">
                  <table class="sh-table">
                    <thead><tr><th>目标</th><th>AI</th><th>原因</th><th>时间</th></tr></thead>
                    <tbody>
                      <tr v-if="!recentReports.length"><td colspan="4" class="sh-table__empty"><strong>暂无举报</strong>用户提交的新举报会显示在这里。</td></tr>
                      <tr v-for="report in recentReports.slice(0, 6)" :key="report.id" data-clickable="true" @click="inspectReport(report)">
                        <td>{{ report.targetMemberId }}</td>
                        <td><SeverityTag :label="`${report.aiStatus}/${report.aiSeverity}`" :intent="describeLevel(report.aiSeverity)" /></td>
                        <td>{{ report.reason }}</td>
                        <td class="sh-table__mono">{{ formatTimestamp(report.createdAt) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </ConsolePanel>
              <ConsolePanel eyebrow="Snapshot" title="治理配置快照" description="从总览直接看到规则、模板和权限配置的规模。">
                <div class="sh-lane">
                  <div class="sh-lane__row"><span class="sh-lane__dot sh-lane__dot--primary"></span><div><div class="sh-lane__title">关键词规则</div><div class="sh-lane__subtitle">自动处罚与人工复核的入口规则。</div></div><span class="sh-lane__time">{{ keywordRules.length }}</span></div>
                  <div class="sh-lane__row"><span class="sh-lane__dot sh-lane__dot--success"></span><div><div class="sh-lane__title">群模板</div><div class="sh-lane__subtitle">模板统一禁言、提醒与踢出策略。</div></div><span class="sh-lane__time">{{ guardTemplates.length }}</span></div>
                  <div class="sh-lane__row"><span class="sh-lane__dot sh-lane__dot--warning"></span><div><div class="sh-lane__title">群绑定</div><div class="sh-lane__subtitle">群绑定优先于静态 guard 配置。</div></div><span class="sh-lane__time">{{ guardBindings.length }}</span></div>
                  <div class="sh-lane__row"><span class="sh-lane__dot sh-lane__dot--danger"></span><div><div class="sh-lane__title">命令权限</div><div class="sh-lane__subtitle">角色和 authority 共同控制命令入口。</div></div><span class="sh-lane__time">{{ commandPolicies.length }}</span></div>
                </div>
              </ConsolePanel>
            </div>
          </template>

          <template v-else-if="activeTab === 'gate'">
          <div class="sh-split sh-split--7-5">
            <ConsolePanel eyebrow="Gate" title="待认证成员" description="入群准入、解禁和超时踢出都围绕这张表运转。" flush>
              <div class="sh-toolbar"><span class="sh-toolbar__count">已选 {{ selectedGuardIds.length }}</span></div>
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th></th><th>成员</th><th>状态</th><th>群</th><th>截止</th><th>错误</th></tr></thead><tbody><tr v-if="!pendingMembers.length"><td colspan="6" class="sh-table__empty"><strong>当前没有待认证成员</strong>新入群未认证成员会出现在这里。</td></tr><tr v-for="member in pendingMembers" :key="member.id" data-clickable="true" :aria-selected="inspector.kind === 'member' && inspector.id === member.id" @click="inspectMember(member)"><td><input v-model="selectedGuardIds" type="checkbox" :value="member.id" @click.stop /></td><td>{{ member.memberName }}<div class="sh-table__id">{{ member.memberId }}</div></td><td><SeverityTag :label="member.verificationState" intent="warning" /></td><td>{{ member.guildId }}</td><td class="sh-table__mono">{{ formatTimestamp(member.deadlineAt) }}</td><td>{{ member.lastError || '—' }}</td></tr></tbody></table></div>
            </ConsolePanel>
            <ConsolePanel eyebrow="Action" title="批量操作" description="禁言、解禁和角色设置可以直接执行；踢人仍统一走复核。">
              <div class="sh-form-grid">
                <label class="sh-field"><span class="sh-field__label">动作</span><select v-model="guardForm.action" class="sh-select"><option value="mute">批量禁言</option><option value="unmute">批量解除禁言</option><option value="kick">提交踢出复核</option><option value="set-role">批量设置角色</option><option value="unset-role">批量移除角色</option></select></label>
                <label class="sh-field"><span class="sh-field__label">禁言秒数</span><input v-model.number="guardForm.seconds" class="sh-input sh-input--mono" type="number" min="0" /></label>
                <label class="sh-field"><span class="sh-field__label">角色 ID</span><input v-model="guardForm.roleId" class="sh-input sh-input--mono" placeholder="role-id" /></label>
                <label class="sh-field"><span class="sh-field__label">原因</span><input v-model="guardForm.reason" class="sh-input" placeholder="控制台批量操作" /></label>
                <label class="sh-check"><input v-model="guardForm.permanent" type="checkbox" /><span>同时拉黑</span></label>
                <button class="sh-btn sh-btn--primary" :disabled="!selectedGuardIds.length || loading" @click="runTask(submitGuardAction)">执行操作</button>
              </div>
              <p class="sh-field__hint">踢人和踢人并拉黑属于高风险动作，提交后只会进入复核队列，不会直接对群成员执行。</p>
            </ConsolePanel>
          </div>
        </template>

        <template v-else-if="activeTab === 'rules'">
          <div class="sh-split sh-split--7-5">
            <ConsolePanel eyebrow="Rule" title="关键词规则" description="规则表管理自动处罚与转人工复核的命中条件。">
              <div class="sh-form-grid">
                <label class="sh-field"><span class="sh-field__label">规则 ID</span><input v-model="ruleForm.id" class="sh-input sh-input--mono" placeholder="spam-link" /></label>
                <label class="sh-field"><span class="sh-field__label">群号或 *</span><input v-model="ruleForm.guildId" class="sh-input sh-input--mono" placeholder="*" /></label>
                <label class="sh-field"><span class="sh-field__label">关键词 / 正则</span><input v-model="ruleForm.pattern" class="sh-input" placeholder="输入规则内容" /></label>
                <label class="sh-field"><span class="sh-field__label">匹配模式</span><select v-model="ruleForm.matchMode" class="sh-select"><option value="includes">includes</option><option value="regex">regex</option></select></label>
                <label class="sh-field"><span class="sh-field__label">动作</span><select v-model="ruleForm.action" class="sh-select"><option value="warn">warn</option><option value="delete">delete</option><option value="mute">mute</option><option value="review">review</option></select></label>
                <label class="sh-field"><span class="sh-field__label">禁言秒数</span><input v-model.number="ruleForm.muteSeconds" class="sh-input sh-input--mono" type="number" min="0" /></label>
                <label class="sh-field"><span class="sh-field__label">备注</span><input v-model="ruleForm.note" class="sh-input" placeholder="记录规则来源" /></label>
                <label class="sh-check"><input v-model="ruleForm.enabled" type="checkbox" /><span>规则启用</span></label>
                <button class="sh-btn sh-btn--primary" @click="runTask(submitRule)">保存规则</button>
              </div>
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>规则</th><th>匹配</th><th>动作</th><th>群</th><th>操作</th></tr></thead><tbody><tr v-if="!keywordRules.length"><td colspan="5" class="sh-table__empty"><strong>暂无规则</strong>创建第一条自动处罚规则。</td></tr><tr v-for="rule in keywordRules" :key="rule.id" data-clickable="true" @click="openInspector('rule', rule.id, rule)"><td>{{ rule.id }}</td><td>{{ rule.pattern }}</td><td><SeverityTag :label="describeAction(rule.action).label" :intent="describeAction(rule.action).intent" /></td><td>{{ rule.guildId }}</td><td class="sh-table__actions"><button class="sh-btn sh-btn--ghost sh-btn--sm" @click.stop="loadRule(rule)">载入</button></td></tr></tbody></table></div>
            </ConsolePanel>
            <div class="sh-split">
              <ConsolePanel eyebrow="Command" title="命令权限" description="authority 与角色白名单一起决定命令是否可执行。">
                <div class="sh-form-grid sh-form-grid--narrow">
                  <label class="sh-field"><span class="sh-field__label">命令</span><select v-model="policyForm.commandId" class="sh-select"><option v-for="commandId in supportedCommandIds" :key="commandId" :value="commandId">{{ commandId }}</option></select></label>
                  <label class="sh-field"><span class="sh-field__label">最小 authority</span><input v-model.number="policyForm.minAuthority" class="sh-input sh-input--mono" type="number" min="0" /></label>
                  <label class="sh-field"><span class="sh-field__label">允许角色</span><input v-model="policyForm.rolesText" class="sh-input" placeholder="reviewer, admin" /></label>
                  <button class="sh-btn sh-btn--primary" @click="runTask(submitPolicy)">保存</button>
                </div>
                <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>命令</th><th>Authority</th><th>角色</th></tr></thead><tbody><tr v-if="!commandPolicies.length"><td colspan="3" class="sh-table__empty"><strong>暂无命令策略</strong>默认命令权限仍可用。</td></tr><tr v-for="policy in commandPolicies" :key="policy.commandId"><td>{{ policy.commandId }}</td><td>{{ policy.minAuthority }}</td><td>{{ policy.roles.join(', ') || '—' }}</td></tr></tbody></table></div>
              </ConsolePanel>
              <ConsolePanel eyebrow="Role" title="成员角色" description="群内角色是命令放行、审核分权和运营协作的基础。">
                <div class="sh-form-grid sh-form-grid--narrow">
                  <label class="sh-field"><span class="sh-field__label">群号</span><input v-model="roleForm.guildId" class="sh-input sh-input--mono" placeholder="guild-id" /></label>
                  <label class="sh-field"><span class="sh-field__label">成员 ID</span><input v-model="roleForm.memberId" class="sh-input sh-input--mono" placeholder="member-id" /></label>
                  <label class="sh-field"><span class="sh-field__label">角色</span><input v-model="roleForm.rolesText" class="sh-input" placeholder="admin, reviewer" /></label>
                  <button class="sh-btn sh-btn--primary" @click="runTask(submitRoles)">保存</button>
                </div>
                <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>群</th><th>成员</th><th>角色</th></tr></thead><tbody><tr v-if="!memberRoles.length"><td colspan="3" class="sh-table__empty"><strong>暂无成员角色</strong>需要时再为特定成员绑定角色。</td></tr><tr v-for="entry in memberRoles" :key="entry.id"><td>{{ entry.guildId }}</td><td>{{ entry.memberId }}</td><td>{{ entry.roles.join(', ') || '—' }}</td></tr></tbody></table></div>
              </ConsolePanel>
            </div>
          </div>
        </template>

        <GuardPolicyPanel v-else-if="activeTab === 'templates'" :templates="guardTemplates" :bindings="guardBindings" :template-form="templateForm" :binding-form="bindingForm" :run-task="runTask" :submit-template="submitTemplate" :submit-binding="submitBinding" :load-template="loadTemplate" :load-binding="loadBinding" :inspect-template="(item) => openInspector('template', item.id, item)" :inspect-binding="(item) => openInspector('binding', item.id, item)" />

        <template v-else-if="activeTab === 'enforcement'">
          <div class="sh-split sh-split--7-5">
            <ConsolePanel eyebrow="Review" title="人工复核队列" description="踢人、拉黑等高风险动作只在这里执行或驳回。">
              <div class="sh-form-grid sh-form-grid--narrow"><label class="sh-field"><span class="sh-field__label">复核备注</span><input v-model="reviewForm.note" class="sh-input" placeholder="记录这次决策依据" /></label></div>
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>成员</th><th>动作</th><th>原因</th><th>时间</th><th>处理</th></tr></thead><tbody><tr v-if="!pendingReviews.length"><td colspan="5" class="sh-table__empty"><strong>当前没有待复核动作</strong>高风险动作会自动流转到这里。</td></tr><tr v-for="review in pendingReviews" :key="review.id" data-clickable="true" @click="inspectReview(review)"><td>{{ review.memberId }}</td><td><SeverityTag :label="describeAction(review.actionType === 'kick_and_block' ? 'kick-permanent' : review.actionType).label" :intent="describeAction(review.actionType === 'kick_and_block' ? 'kick-permanent' : review.actionType).intent" /></td><td>{{ review.reason }}</td><td class="sh-table__mono">{{ formatTimestamp(review.createdAt) }}</td><td class="sh-table__actions"><button class="sh-btn sh-btn--primary sh-btn--sm" @click.stop="runTask(() => submitReviewAction(review.id, 'execute'))">执行</button><button class="sh-btn sh-btn--ghost sh-btn--sm" @click.stop="runTask(() => submitReviewAction(review.id, 'reject'))">驳回</button></td></tr></tbody></table></div>
            </ConsolePanel>
            <ConsolePanel eyebrow="Report" title="最近举报" description="举报流可以和复核队列并排处理，减少在多个入口间切换。">
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>举报人</th><th>目标</th><th>AI</th><th>原因</th></tr></thead><tbody><tr v-if="!recentReports.length"><td colspan="4" class="sh-table__empty"><strong>暂无举报</strong>用户举报会显示在这里。</td></tr><tr v-for="report in recentReports.slice(0, 10)" :key="report.id" data-clickable="true" @click="inspectReport(report)"><td>{{ report.reporterMemberId }}</td><td>{{ report.targetMemberId }}</td><td><SeverityTag :label="`${report.aiStatus}/${report.aiSeverity}`" :intent="describeLevel(report.aiSeverity)" /></td><td>{{ report.reason }}</td></tr></tbody></table></div>
            </ConsolePanel>
          </div>
        </template>

        <template v-else>
          <div class="sh-split sh-split--1-1">
            <ConsolePanel eyebrow="Event" title="事件日志" description="事件流用于回溯自动处罚、人工动作和系统异常。">
              <div class="sh-toolbar"><input v-model="eventSearch" class="sh-input sh-input--mono" placeholder="搜索成员 / 摘要 / 群号" /></div>
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>时间</th><th>级别</th><th>成员</th><th>摘要</th></tr></thead><tbody><tr v-if="!filteredEvents.length"><td colspan="4" class="sh-table__empty"><strong>没有匹配的事件</strong>调整搜索词后重试。</td></tr><tr v-for="event in filteredEvents" :key="event.id" data-clickable="true" @click="inspectEvent(event)"><td class="sh-table__mono">{{ formatTimestamp(event.createdAt) }}</td><td><SeverityTag :label="event.level" :intent="describeLevel(event.level)" /></td><td>{{ event.memberId }}</td><td>{{ event.summary }}</td></tr></tbody></table></div>
            </ConsolePanel>
            <ConsolePanel eyebrow="Report" title="举报日志" description="按举报人、目标和 AI 摘要检索最近的举报记录。">
              <div class="sh-toolbar"><input v-model="reportSearch" class="sh-input sh-input--mono" placeholder="搜索举报人 / 目标 / 原因" /></div>
              <div class="sh-table-shell"><table class="sh-table"><thead><tr><th>时间</th><th>举报人</th><th>目标</th><th>AI</th><th>原因</th></tr></thead><tbody><tr v-if="!filteredReports.length"><td colspan="5" class="sh-table__empty"><strong>没有匹配的举报</strong>调整关键词后重试。</td></tr><tr v-for="report in filteredReports" :key="report.id" data-clickable="true" @click="inspectReport(report)"><td class="sh-table__mono">{{ formatTimestamp(report.createdAt) }}</td><td>{{ report.reporterMemberId }}</td><td>{{ report.targetMemberId }}</td><td><SeverityTag :label="`${report.aiStatus}/${report.aiSeverity}`" :intent="describeLevel(report.aiSeverity)" /></td><td>{{ report.reason }}</td></tr></tbody></table></div>
            </ConsolePanel>
          </div>
          </template>
        </section>

        <NoticeStack :items="notices" @dismiss="dismissNotice" />
        <Drawer :open="inspector.open" :title="inspectorTitle" :subtitle="inspector.id" @close="closeInspector">
          <dl class="sh-keylist"><template v-for="item in inspectorDetails" :key="item.label"><dt>{{ item.label }}</dt><dd :class="{ 'sh-mono': item.mono }">{{ item.value }}</dd></template></dl>
          <template #footer v-if="reviewPending"><button class="sh-btn sh-btn--ghost" @click="runTask(() => submitReviewAction(inspector.id, 'reject'))">驳回</button><button class="sh-btn sh-btn--primary" @click="runTask(() => submitReviewAction(inspector.id, 'execute'))">执行</button></template>
        </Drawer>
      </div>
    </el-scrollbar>
  </k-layout>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { StuhelperConsoleEvent, StuhelperConsoleGuardBinding, StuhelperConsoleGuardMember, StuhelperConsoleGuardTemplate, StuhelperConsoleKeywordRule, StuhelperConsoleReport, StuhelperConsoleReview } from '../src/console-types'
import ConsoleHero from './components/ConsoleHero.vue'
import ConsolePanel from './components/ConsolePanel.vue'
import Drawer from './components/Drawer.vue'
import EmptyState from './components/EmptyState.vue'
import GuardPolicyPanel from './components/GuardPolicyPanel.vue'
import MetricCard from './components/MetricCard.vue'
import NoticeStack from './components/NoticeStack.vue'
import SeverityTag from './components/SeverityTag.vue'
import Tabs from './components/Tabs.vue'
import { describeAction, describeLevel, formatTimestamp, useConsolePage } from './use-console-page'

const { data, title, generatedAt, loading, activeTab, tabs, goto, inspector, openInspector, closeInspector, inspectMember, inspectReview, inspectEvent, inspectReport, notices, dismissNotice, pendingMembers, pendingReviews, keywordRules, commandPolicies, memberRoles, guardTemplates, guardBindings, recentEvents, recentReports, supportedCommandIds, filteredEvents, filteredReports, metrics, eventSearch, reportSearch, selectedGuardIds, guardForm, reviewForm, ruleForm, templateForm, bindingForm, roleForm, policyForm, runTask, refresh, submitGuardAction, submitReviewAction, submitRule, submitTemplate, submitBinding, submitRoles, submitPolicy, loadRule, loadTemplate, loadBinding } = useConsolePage()

const viewMeta = computed(() => ({ overview: { eyebrow: 'OVERVIEW', title: '总览', lead: '从一屏里判断今天要优先处理什么。' }, gate: { eyebrow: 'MEMBER GATE', title: '认证准入', lead: '入群后的禁言、解禁、超时踢出都围绕这一页。' }, rules: { eyebrow: 'RULE ENGINE', title: '通用群规', lead: '把自动处罚、命令权限和成员角色收敛到统一规则面。' }, templates: { eyebrow: 'POLICY TEMPLATE', title: '模板与群绑定', lead: '模板统一策略，绑定决定哪些群优先读取数据库规则。' }, enforcement: { eyebrow: 'REVIEW CENTER', title: '处置中心', lead: '高风险动作必须经人工复核，举报流也从这里汇入。' }, events: { eyebrow: 'AUDIT LOG', title: '事件日志', lead: '按事件与举报两个方向检索最近的治理记录。' } }[activeTab.value]))
const overviewQueues = computed(() => [{ tab: 'gate', title: '待认证成员', subtitle: '尚未完成注册、绑定或认证的入群成员。', count: pendingMembers.value.length, intent: 'warning' }, { tab: 'enforcement', title: '人工复核', subtitle: '踢人和拉黑等高风险动作统一回流此处。', count: pendingReviews.value.length, intent: 'danger' }, { tab: 'events', title: '开放举报', subtitle: '近期新增的举报单与 AI 摘要。', count: recentReports.value.length, intent: 'primary' }])
const reviewPending = computed(() => inspector.kind === 'review' && (inspector.payload as StuhelperConsoleReview | null)?.status === 'pending')
const inspectorTitle = computed(() => ({ member: '成员详情', review: '复核详情', event: '事件详情', report: '举报详情', template: '模板详情', binding: '绑定详情', rule: '规则详情' }[inspector.kind || 'event']))
const inspectorDetails = computed(() => {
  const payload = inspector.payload as Record<string, unknown> | null
  if (!payload) return []
  if (inspector.kind === 'member') return detailList(payload as unknown as StuhelperConsoleGuardMember, [['成员', 'memberName'], ['成员 ID', 'memberId', true], ['群号', 'guildId', true], ['状态', 'verificationState'], ['截止', 'deadlineAt', true], ['最后错误', 'lastError']])
  if (inspector.kind === 'review') return detailList(payload as unknown as StuhelperConsoleReview, [['成员', 'memberId', true], ['动作', 'actionType'], ['状态', 'status'], ['原因', 'reason'], ['提交时间', 'createdAt', true], ['备注', 'resolutionNote']])
  if (inspector.kind === 'event') return detailList(payload as unknown as StuhelperConsoleEvent, [['类型', 'type'], ['级别', 'level'], ['成员', 'memberId', true], ['群号', 'guildId', true], ['摘要', 'summary'], ['时间', 'createdAt', true]])
  if (inspector.kind === 'report') return detailList(payload as unknown as StuhelperConsoleReport, [['举报人', 'reporterMemberId', true], ['目标', 'targetMemberId', true], ['AI 状态', 'aiStatus'], ['AI 等级', 'aiSeverity'], ['AI 摘要', 'aiSummary'], ['原因', 'reason'], ['时间', 'createdAt', true]])
  if (inspector.kind === 'template') return detailList(payload as unknown as StuhelperConsoleGuardTemplate, [['模板', 'name'], ['模板 ID', 'id', true], ['禁言秒数', 'muteDurationSeconds', true], ['踢出分钟数', 'kickAfterMinutes', true], ['提醒文案', 'reminderTemplate'], ['启用', 'enabled']])
  if (inspector.kind === 'binding') return detailList(payload as unknown as StuhelperConsoleGuardBinding, [['平台', 'platform'], ['群号', 'guildId', true], ['模板', 'templateId', true], ['启用', 'enabled'], ['备注', 'note']])
  return detailList(payload as unknown as StuhelperConsoleKeywordRule, [['规则', 'id', true], ['群号', 'guildId', true], ['模式', 'matchMode'], ['动作', 'action'], ['表达式', 'pattern'], ['备注', 'note']])
})

function detailList(record: Record<string, unknown>, fields: Array<[string, string, boolean?]>) {
  return fields.map(([label, key, mono]) => ({ label, value: normalizeValue(record[key]), mono: Boolean(mono) }))
}

function normalizeValue(value: unknown) {
  if (value === null || value === undefined || value === '') return '—'
  if (Array.isArray(value)) return value.join(', ') || '—'
  if (typeof value === 'boolean') return value ? '是' : '否'
  return String(value)
}
</script>
