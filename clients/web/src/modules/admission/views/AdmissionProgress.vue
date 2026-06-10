<template>
  <section
    v-if="session"
    class="admission-progress"
    aria-label="入群认证进度"
    data-admission-progress
  >
    <div class="admission-progress__summary">
      <div>
        <p class="admission-progress__eyebrow">当前阶段</p>
        <p class="admission-progress__title" data-admission-progress-current>
          {{ currentStep.label }}
        </p>
      </div>
      <p
        v-if="activeDeadline"
        class="admission-progress__deadline"
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
        class="admission-progress__deadline"
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
        <span class="admission-progress__dot" aria-hidden="true" />
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
.admission-progress {
  border-bottom: 1px solid #e2e8f0;
  display: grid;
  gap: 16px;
  margin: 0 0 20px;
  padding-bottom: 20px;
}

.admission-progress__summary {
  align-items: center;
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(0, 1fr) auto;
}

.admission-progress__eyebrow {
  color: #64748b;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  margin: 0;
}

.admission-progress__title {
  color: #0f172a;
  font-size: 18px;
  font-weight: 700;
  line-height: 26px;
  margin: 0;
}

.admission-progress__deadline,
.admission-progress__mute {
  color: #475569;
  font-size: 13px;
  line-height: 20px;
  margin: 0;
}

.admission-progress__deadline {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  padding: 7px 12px;
  white-space: nowrap;
}

.admission-progress__mute {
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  color: #92400e;
  padding: 10px 12px;
}

.admission-progress__steps {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  list-style: none;
  margin: 0;
  padding: 0;
}

.admission-progress__step {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.admission-progress__dot {
  background: #cbd5e1;
  border-radius: 999px;
  height: 8px;
  width: 100%;
}

.admission-progress__content {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.admission-progress__label {
  color: #334155;
  font-size: 13px;
  font-weight: 700;
  line-height: 18px;
}

.admission-progress__description {
  color: #64748b;
  font-size: 12px;
  line-height: 17px;
}

.admission-progress__step--complete .admission-progress__dot,
.admission-progress__step--current .admission-progress__dot {
  background: #0f766e;
}

.admission-progress__step--current .admission-progress__label {
  color: #0f766e;
}

.admission-progress__step--failed .admission-progress__dot {
  background: #dc2626;
}

.admission-progress__step--failed .admission-progress__label {
  color: #b91c1c;
}

@media (max-width: 640px) {
  .admission-progress__summary {
    align-items: start;
    grid-template-columns: 1fr;
  }

  .admission-progress__deadline {
    border-radius: 8px;
    white-space: normal;
  }

  .admission-progress__steps {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
