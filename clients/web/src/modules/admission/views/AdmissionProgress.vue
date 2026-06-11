<template>
  <section
    v-if="session"
    class="admission-progress"
    aria-label="入群认证进度"
    data-admission-progress
  >
    <div class="admission-progress__summary">
      <div class="admission-progress__now">
        <p class="admission-progress__eyebrow join-eyebrow">当前阶段</p>
        <p class="admission-progress__title" data-admission-progress-current>
          {{ currentStep.label }}
        </p>
      </div>
      <p
        v-if="activeDeadline"
        class="admission-progress__deadline join-chip"
        data-admission-active-deadline
      >
        {{ activeDeadline.label }}：
        <time :datetime="activeDeadline.value">
          {{ formatAdmissionDateTime(activeDeadline.value) }}
        </time>
        <span v-if="remainingText">（{{ remainingText }}）</span>
      </p>
      <p
        v-else-if="pageState === 'approved'"
        class="admission-progress__deadline join-chip"
        data-admission-active-deadline
      >
        已通过认证，等待机器人同步群内状态。
      </p>
    </div>

    <ol class="admission-progress__steps">
      <li
        v-for="step in renderedSteps"
        :key="step.key"
        class="admission-progress__step"
        :class="`admission-progress__step--${step.state}`"
        data-admission-progress-step
      >
        <span class="admission-progress__marker" aria-hidden="true" />
        <span class="admission-progress__content">
          <span class="admission-progress__label">{{ step.label }}</span>
          <span class="admission-progress__description">{{ step.description }}</span>
        </span>
      </li>
    </ol>

    <p
      v-if="muteUntilText"
      class="admission-progress__mute"
      data-admission-mute-deadline
    >
      群内临时禁言至 {{ muteUntilText }}；学生认证通过后会提前解除。
    </p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

import { formatDate } from '@/utils/date'
import type { AdmissionSession } from '@stuhelper/shared/api'

import type { AdmissionPageState } from '../admissionState'

type StepKey = 'account' | 'student' | 'review' | 'release'
type StepState = 'complete' | 'current' | 'failed' | 'pending'

type ProgressStep = {
  description: string
  key: StepKey
  label: string
}

type RenderedProgressStep = ProgressStep & {
  state: StepState
}

const props = defineProps<{
  pageState: AdmissionPageState
  session: AdmissionSession | null
}>()

const nowMs = ref(Date.now())
let timerID: number | undefined

const steps: ProgressStep[] = [
  {
    description: '登录 StuHelper 后绑定本次 QQ 入群会话',
    key: 'account',
    label: '账号绑定',
  },
  {
    description: '完成老生邮箱/统一认证，或提交新生材料',
    key: 'student',
    label: '学生认证',
  },
  {
    description: '等待审核或身份状态同步',
    key: 'review',
    label: '审核同步',
  },
  {
    description: '机器人确认后解除禁言',
    key: 'release',
    label: '解除禁言',
  },
]

const currentIndex = computed(() => {
  if (props.pageState === 'linked') return 1
  if (props.pageState === 'pendingReview') return 2
  if (props.pageState === 'projectionPending' || props.pageState === 'approved') return 3
  return 0
})

const failed = computed(() => {
  return ['accountMismatch', 'error', 'expired', 'qqMismatch'].includes(props.pageState)
})

const currentStep = computed(() => {
  return steps[currentIndex.value] ?? steps[0]
})

const renderedSteps = computed<RenderedProgressStep[]>(() => {
  return steps.map((step, index) => {
    let state: StepState = 'pending'
    if (props.pageState === 'approved') state = 'complete'
    else if (failed.value && index === currentIndex.value) state = 'failed'
    else if (index < currentIndex.value) state = 'complete'
    else if (index === currentIndex.value) state = 'current'
    return { ...step, state }
  })
})

const activeDeadline = computed(() => {
  const current = props.session
  if (!current) return null
  if (props.pageState === 'needsLogin' || props.pageState === 'ready') {
    return { label: '绑定账号截止', value: current.linkWaitDeadlineAt }
  }
  if (props.pageState === 'linked') {
    return { label: '提交认证截止', value: current.submissionWaitDeadlineAt }
  }
  if (props.pageState === 'pendingReview' && current.manualReviewDeadlineAt) {
    return { label: '审核处理截止', value: current.manualReviewDeadlineAt }
  }
  return null
})

const remainingText = computed(() => {
  const deadline = activeDeadline.value
  if (!deadline) return ''
  return formatRemaining(deadline.value, nowMs.value)
})

const muteUntilText = computed(() => {
  if (!props.session || props.pageState === 'approved' || props.pageState === 'expired') {
    return ''
  }
  return formatAdmissionDateTime(props.session.initialMuteUntil)
})

onMounted(() => {
  nowMs.value = Date.now()
  timerID = window.setInterval(() => {
    nowMs.value = Date.now()
  }, 60_000)
})

onBeforeUnmount(() => {
  if (timerID !== undefined) {
    window.clearInterval(timerID)
    timerID = undefined
  }
})

function formatAdmissionDateTime(value: string): string {
  return formatDate(value, 'YYYY-MM-DD HH:mm')
}

function formatRemaining(value: string, now: number): string {
  const deadlineMs = new Date(value).getTime()
  if (!Number.isFinite(deadlineMs)) return ''
  const remainingMs = deadlineMs - now
  if (remainingMs <= 0) return '已超时'
  const totalMinutes = Math.ceil(remainingMs / 60_000)
  if (totalMinutes < 60) return `剩余 ${totalMinutes} 分钟`
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (minutes === 0) return `剩余 ${hours} 小时`
  return `剩余 ${hours} 小时 ${minutes} 分钟`
}
</script>

<style scoped>
/*
 * 玻璃步骤轨道：四个状态化标记（完成=渐变对勾、当前=辉光圆环、
 * 失败=危险色、待办=弱化）+ 节点间连接线。
 * data-admission-* 属性与全部文案为测试契约；逻辑层未改动。
 */
.admission-progress {
  border-bottom: 1px solid var(--join-glass-border-soft);
  display: grid;
  gap: 18px;
  margin: 0 0 22px;
  padding-bottom: 22px;
}

/* ── 概要行：当前阶段 + 截止时间 chip ──────────────── */
.admission-progress__summary {
  align-items: center;
  display: grid;
  gap: 10px;
  grid-template-columns: minmax(0, 1fr) auto;
}

.admission-progress__now {
  display: grid;
  gap: 2px;
}

.admission-progress__eyebrow {
  margin: 0;
}

.admission-progress__title {
  color: var(--join-ink);
  font-size: 19px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 26px;
  margin: 0;
}

.admission-progress__deadline {
  border-radius: 999px;
  color: var(--join-ink-soft);
  font-size: 13px;
  line-height: 20px;
  margin: 0;
  padding: 9px 14px;
  white-space: nowrap;
}

.admission-progress__deadline time {
  color: var(--join-ink);
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

/* ── 步骤轨道 ─────────────────────────────────────── */
.admission-progress__steps {
  display: grid;
  gap: 14px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  list-style: none;
  margin: 0;
  padding: 0;
}

.admission-progress__step {
  display: grid;
  gap: 10px;
  justify-items: start;
  min-width: 0;
  position: relative;
}

/* 节点间连接线（横向） */
.admission-progress__step:not(:last-child)::after {
  background: var(--join-glass-border);
  border-radius: 999px;
  content: "";
  height: 2px;
  left: 34px;
  position: absolute;
  right: 6px;
  top: 12px;
}

.admission-progress__step--complete:not(:last-child)::after {
  background: linear-gradient(90deg, var(--color-primary), var(--color-accent));
  opacity: 0.6;
}

/* 标记基态（待办=弱化玻璃点） */
.admission-progress__marker {
  background: var(--join-chip-bg);
  border: 2px solid var(--join-glass-border);
  border-radius: 999px;
  box-shadow: inset 0 1px 0 var(--join-glass-highlight);
  display: grid;
  height: 26px;
  place-items: center;
  position: relative;
  width: 26px;
  z-index: 1;
}

/* 完成：品牌渐变填充 + 白色对勾 */
.admission-progress__step--complete .admission-progress__marker {
  background: var(--join-gradient-cta);
  border-color: transparent;
  box-shadow: 0 4px 12px rgba(91, 124, 247, 0.35);
}

.admission-progress__step--complete .admission-progress__marker::after {
  border-bottom: 2px solid #ffffff;
  border-left: 2px solid #ffffff;
  content: "";
  height: 5px;
  margin-top: -2px;
  transform: rotate(-45deg);
  width: 10px;
}

/* 当前：品牌描边 + 辉光圆环 + 实心内点 */
.admission-progress__step--current .admission-progress__marker {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 4px rgba(91, 124, 247, 0.22);
}

.admission-progress__step--current .admission-progress__marker::after {
  background: var(--color-primary);
  border-radius: 999px;
  content: "";
  height: 8px;
  width: 8px;
}

/* 失败：危险色调 */
.admission-progress__step--failed .admission-progress__marker {
  background: var(--join-tone-danger-bg);
  border-color: var(--join-tone-danger-fg);
  box-shadow: none;
}

.admission-progress__step--failed .admission-progress__marker::after {
  background: var(--join-tone-danger-fg);
  border-radius: 999px;
  content: "";
  height: 8px;
  width: 8px;
}

.admission-progress__content {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.admission-progress__label {
  color: var(--join-ink-soft);
  font-size: 13px;
  font-weight: 700;
  line-height: 18px;
}

.admission-progress__step--complete .admission-progress__label {
  color: var(--join-ink);
}

.admission-progress__step--current .admission-progress__label {
  color: var(--color-primary);
}

.admission-progress__step--failed .admission-progress__label {
  color: var(--join-tone-danger-fg);
}

.admission-progress__description {
  color: var(--join-ink-muted);
  font-size: 12px;
  line-height: 17px;
}

/* ── 禁言提示：警示色调 chip ───────────────────────── */
.admission-progress__mute {
  background: var(--join-tone-warning-bg);
  border: 1px solid var(--join-glass-border-soft);
  border-radius: var(--radius-lg);
  color: var(--join-tone-warning-fg);
  font-size: 13px;
  line-height: 20px;
  margin: 0;
  padding: 12px 14px;
}

/* ── 移动端：纵向轨道 ──────────────────────────────── */
@media (max-width: 640px) {
  .admission-progress__summary {
    align-items: start;
    grid-template-columns: 1fr;
    justify-items: start;
  }

  .admission-progress__deadline {
    border-radius: var(--radius-lg);
    white-space: normal;
  }

  .admission-progress__steps {
    gap: 14px;
    grid-template-columns: 1fr;
  }

  .admission-progress__step {
    align-items: start;
    grid-template-columns: auto minmax(0, 1fr);
  }

  .admission-progress__step:not(:last-child)::after {
    bottom: -10px;
    height: auto;
    left: 12px;
    right: auto;
    top: 32px;
    width: 2px;
  }

  .admission-progress__step--complete:not(:last-child)::after {
    background: linear-gradient(180deg, var(--color-primary), var(--color-accent));
  }
}
</style>
