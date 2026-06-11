<template>
  <section class="old-flow" data-admission-old-student-flow>
    <div v-if="schools.length === 0" class="old-flow__notice join-chip">
      暂无可用学校认证配置。
    </div>

    <div v-else class="old-flow__stack">
      <label class="old-flow__field">
        <span class="join-label">学校</span>
        <span class="old-flow__select-wrap">
          <select
            :value="selectedSchoolCode"
            class="join-select"
            data-school-select
            @change="updateSchoolCode"
          >
            <option
              v-for="school in schools"
              :key="school.schoolCode"
              :value="school.schoolCode"
            >
              {{ school.schoolName }}（{{ school.schoolCode }}）
            </option>
          </select>
        </span>
      </label>

      <button
        v-if="selectedSchoolHasSSO"
        class="secondary-button old-flow__sso"
        data-school-sso-button
        type="button"
        @click="startSchoolSSO"
      >
        学校官方 SSO
      </button>

      <form
        v-if="linked && selectedSchoolHasEmailOTP"
        class="old-flow__form join-chip"
        data-school-email-otp-form
        @submit.prevent="verifyEmailOTP"
      >
        <div v-if="selectedSchoolRequiresAcademicEmail" class="old-flow__pair">
          <label class="old-flow__field">
            <span class="join-label">学号</span>
            <input
              :value="studentID"
              class="join-input"
              data-academic-student-id-input
              type="text"
              @input="updateStudentID"
            >
          </label>
          <label class="old-flow__field">
            <span class="join-label">姓名</span>
            <input
              :value="studentName"
              class="join-input"
              data-academic-student-name-input
              type="text"
              @input="updateStudentName"
            >
          </label>
        </div>
        <p
          v-if="selectedSchoolRequiresAcademicEmail && academicMatchMessage"
          class="old-flow__match"
          :class="academicMatchMessageClass"
          data-academic-match-status
        >
          {{ academicMatchMessage }}
        </p>
        <div class="old-flow__send-row">
          <label class="old-flow__field">
            <span class="join-label">学校邮箱</span>
            <input
              :value="email"
              class="join-input"
              data-academic-email-input
              type="email"
              :readonly="selectedSchoolRequiresAcademicEmail"
              :placeholder="selectedSchoolRequiresAcademicEmail ? '学号和姓名校验通过后自动填写' : ''"
              @input="updateEmail"
            >
          </label>
          <button
            class="secondary-button old-flow__send"
            :class="{ 'button-disabled': selectedSchoolRequiresAcademicEmail && !canRequestEmailOTP }"
            data-school-email-otp-request
            type="button"
            :aria-disabled="selectedSchoolRequiresAcademicEmail && !canRequestEmailOTP ? 'true' : undefined"
            :disabled="requestingOTP || (!selectedSchoolRequiresAcademicEmail && !canRequestEmailOTP)"
            @click="requestEmailOTP"
          >
            {{ selectedSchoolRequiresAcademicEmail ? '校验并发送验证码' : '发送验证码' }}
          </button>
        </div>
        <p v-if="successMessage" class="old-flow__feedback old-flow__feedback--success">{{ successMessage }}</p>
        <label class="old-flow__field old-flow__code-field">
          <span class="join-label">验证码</span>
          <input
            :value="code"
            class="join-input"
            inputmode="numeric"
            type="text"
            @input="updateCode"
          >
        </label>
        <button
          class="primary-button old-flow__verify"
          data-school-email-otp-verify
          type="submit"
          :disabled="!canVerifyEmailOTP"
        >
          {{ submitting ? '校验中...' : '验证邮箱' }}
        </button>
      </form>

      <div v-else-if="!linked" class="old-flow__notice join-chip">
        请先确认绑定当前 QQ 后再进行老生认证。
      </div>

      <div v-else class="old-flow__notice join-chip">
        当前学校暂未开通加群老生认证方式。
      </div>

      <p v-if="errorMessage" class="old-flow__feedback old-flow__feedback--danger">{{ errorMessage }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

import type { AdmissionMe } from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import { isAdmissionSessionExpiredError } from '../admissionToken'
import {
  buildSchoolSSOLoginPath,
  schoolHasAdmissionEmailOTP,
  schoolHasAdmissionSSO,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'

const props = defineProps<{
  currentReturnUrl: string
  admissionSessionId?: string
  linked: boolean
  schools: AdmissionSchoolOption[]
}>()

const emit = defineEmits<{
  expired: []
  verified: [admission: AdmissionMe]
}>()

const selectedSchoolCode = ref('')
const email = ref('')
const studentID = ref('')
const studentName = ref('')
const code = ref('')
const submitting = ref(false)
const requestingOTP = ref(false)
const errorMessage = ref('')
const successMessage = ref('')
const academicMatchState = ref<'idle' | 'waiting' | 'checking' | 'matched' | 'mismatch' | 'error'>('idle')
const academicMatchMessage = ref('')
let academicMatchTimer: ReturnType<typeof setTimeout> | undefined
let academicMatchRunID = 0

const selectedSchool = computed(() => {
  return props.schools.find((school) => school.schoolCode === selectedSchoolCode.value)
    ?? null
})
const selectedSchoolHasSSO = computed(() => schoolHasAdmissionSSO(selectedSchool.value))
const selectedSchoolHasEmailOTP = computed(() => schoolHasAdmissionEmailOTP(selectedSchool.value))
const selectedSchoolRequiresAcademicEmail = computed(() => {
  return selectedSchool.value?.schoolEmailIdentityPolicy?.type === 'academic_student_email'
})
const canRequestEmailOTP = computed(() => {
  if (!selectedSchoolHasEmailOTP.value) return false
  if (selectedSchoolRequiresAcademicEmail.value) {
    return academicMatchState.value === 'matched' && email.value.trim() !== ''
  }
  return email.value.trim() !== ''
})
const canVerifyEmailOTP = computed(() => {
  return !submitting.value && email.value.trim() !== '' && code.value.trim() !== ''
})
const academicMatchMessageClass = computed(() => {
  if (academicMatchState.value === 'matched') return 'text-sm text-green-700'
  if (academicMatchState.value === 'checking') return 'text-sm text-slate-600'
  return 'text-sm text-red-600'
})

watch(
  () => props.schools,
  (schools) => {
    if (schools.some((school) => school.schoolCode === selectedSchoolCode.value)) return
    selectedSchoolCode.value = schools[0]?.schoolCode ?? ''
  },
  { immediate: true },
)

function updateSchoolCode(event: Event): void {
  selectedSchoolCode.value = readControlValue(event).trim()
}

function updateEmail(event: Event): void {
  if (selectedSchoolRequiresAcademicEmail.value) return
  email.value = readControlValue(event).trim()
  successMessage.value = ''
}

function updateStudentID(event: Event): void {
  studentID.value = readControlValue(event).trim()
  code.value = ''
  successMessage.value = ''
}

function updateStudentName(event: Event): void {
  studentName.value = readControlValue(event).trim()
  code.value = ''
  successMessage.value = ''
}

function updateCode(event: Event): void {
  code.value = readControlValue(event).trim()
}

function startSchoolSSO(): void {
  const schoolCode = requireSelectedSchoolCode()
  window.location.href = buildSchoolSSOLoginPath(
    schoolCode,
    props.currentReturnUrl,
    props.admissionSessionId,
  )
}

async function requestEmailOTP(): Promise<void> {
  errorMessage.value = ''
  successMessage.value = ''
  if (requestingOTP.value) return
  if (!canRequestEmailOTP.value) {
    errorMessage.value = selectedSchoolRequiresAcademicEmail.value
      ? academicRequestBlockedMessage()
      : '请先填写学校邮箱。'
    return
  }
  requestingOTP.value = true
  try {
    const result = await admissionApi.requestSchoolEmailOTP({
      schoolCode: requireSelectedSchoolCode(),
      ...admissionSessionContext(),
      email: selectedSchoolRequiresAcademicEmail.value ? undefined : requireEmail(),
      studentID: selectedSchoolRequiresAcademicEmail.value ? requireStudentID() : undefined,
      studentName: selectedSchoolRequiresAcademicEmail.value ? requireStudentName() : undefined,
    })
    email.value = result.email
    successMessage.value = selectedSchoolRequiresAcademicEmail.value
      ? '学号和姓名已匹配，验证码已发送到学号邮箱。'
      : '验证码已发送。'
  } catch (error) {
    if (isAdmissionSessionExpiredError(error)) {
      emit('expired')
      return
    }
    errorMessage.value = readErrorMessage(error, '验证码发送失败。')
  } finally {
    requestingOTP.value = false
  }
}

async function verifyEmailOTP(): Promise<void> {
  if (submitting.value) return
  errorMessage.value = ''
  submitting.value = true
  try {
    const admission = await admissionApi.verifySchoolEmailOTP({
      schoolCode: requireSelectedSchoolCode(),
      ...admissionSessionContext(),
      email: requireEmail(),
      code: requireCode(),
    })
    emit('verified', admission)
  } catch (error) {
    if (isAdmissionSessionExpiredError(error)) {
      emit('expired')
      return
    }
    errorMessage.value = readErrorMessage(error, '邮箱验证失败。')
  } finally {
    submitting.value = false
  }
}

function requireSelectedSchoolCode(): string {
  if (!selectedSchoolCode.value) {
    throw new Error('请选择学校')
  }
  return selectedSchoolCode.value
}

function admissionSessionContext(): { admissionSessionID?: string } {
  return props.admissionSessionId ? { admissionSessionID: props.admissionSessionId } : {}
}

function requireEmail(): string {
  if (!email.value) {
    throw new Error('请填写学校邮箱')
  }
  return email.value
}

function requireStudentID(): string {
  if (!studentID.value) {
    throw new Error('请填写学号')
  }
  return studentID.value
}

function requireStudentName(): string {
  if (!studentName.value) {
    throw new Error('请填写姓名')
  }
  return studentName.value
}

function requireCode(): string {
  if (!code.value) {
    throw new Error('请填写验证码')
  }
  return code.value
}

watch(
  () => selectedSchool.value?.schoolCode,
  () => {
    resetAcademicMatch()
    email.value = ''
    studentID.value = ''
    studentName.value = ''
    code.value = ''
    successMessage.value = ''
  },
)

watch(
  [selectedSchoolRequiresAcademicEmail, selectedSchoolCode, studentID, studentName],
  () => {
    scheduleAcademicMatch()
  },
)

onBeforeUnmount(() => {
  clearAcademicMatchTimer()
  academicMatchRunID += 1
})

function scheduleAcademicMatch(): void {
  clearAcademicMatchTimer()
  academicMatchRunID += 1
  if (!selectedSchoolRequiresAcademicEmail.value) {
    resetAcademicMatch()
    return
  }
  email.value = ''
  const normalizedStudentID = studentID.value.trim()
  const normalizedStudentName = studentName.value.trim()
  if (!normalizedStudentID && !normalizedStudentName) {
    academicMatchState.value = 'idle'
    academicMatchMessage.value = ''
    return
  }
  if (!normalizedStudentID || !normalizedStudentName) {
    academicMatchState.value = 'waiting'
    academicMatchMessage.value = '请先输入学号和姓名。'
    return
  }
  academicMatchState.value = 'checking'
  academicMatchMessage.value = '正在匹配学号和姓名...'
  const runID = academicMatchRunID
  academicMatchTimer = setTimeout(() => {
    void runAcademicMatch(runID, normalizedStudentID, normalizedStudentName)
  }, 300)
}

async function runAcademicMatch(
  runID: number,
  normalizedStudentID: string,
  normalizedStudentName: string,
): Promise<void> {
  try {
    const result = await admissionApi.matchSchoolEmailAcademicStudent({
      schoolCode: requireSelectedSchoolCode(),
      ...admissionSessionContext(),
      studentID: normalizedStudentID,
      studentName: normalizedStudentName,
    })
    if (runID !== academicMatchRunID) return
    if (result.matched && result.email) {
      academicMatchState.value = 'matched'
      academicMatchMessage.value = result.message || '学号和姓名已匹配。'
      email.value = result.email
      return
    }
    academicMatchState.value = 'mismatch'
    academicMatchMessage.value = result.message || '学号和姓名不匹配，请核对后再发送验证码。'
    email.value = ''
  } catch (error) {
    if (runID !== academicMatchRunID) return
    if (isAdmissionSessionExpiredError(error)) {
      emit('expired')
      return
    }
    academicMatchState.value = 'error'
    academicMatchMessage.value = readErrorMessage(error, '学籍匹配暂时不可用，请稍后重试。')
    email.value = ''
  }
}

function resetAcademicMatch(): void {
  clearAcademicMatchTimer()
  academicMatchRunID += 1
  academicMatchState.value = 'idle'
  academicMatchMessage.value = ''
}

function clearAcademicMatchTimer(): void {
  if (academicMatchTimer === undefined) return
  clearTimeout(academicMatchTimer)
  academicMatchTimer = undefined
}

function academicRequestBlockedMessage(): string {
  if (!studentID.value.trim() || !studentName.value.trim()) {
    return '请先输入学号和姓名。'
  }
  if (academicMatchState.value === 'checking') {
    return '请等待学号和姓名匹配完成。'
  }
  return '请先通过学号和姓名匹配。'
}

function readControlValue(event: Event): string {
  const target = event.target
  if (
    target instanceof HTMLInputElement ||
    target instanceof HTMLSelectElement
  ) {
    return target.value
  }
  throw new Error('Admission form event target is invalid')
}

function readErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}
</script>

<style scoped>
/*
 * 视觉层为品牌玻璃风（join-theme.css 提供 .join-* 原语与按钮样式）。
 * 测试契约：.primary-button/.secondary-button/.button-disabled 类名、
 * data-* 选择器、输入框 DOM 顺序（验证码输入必须是最后一个 <input>）。
 */
.old-flow,
.old-flow__stack {
  display: grid;
  gap: 16px;
}

/* ── 空态/锁定提示：玻璃信息卡 + 警示圆点 ─────────── */
.old-flow__notice {
  align-items: flex-start;
  color: var(--join-ink-soft);
  display: flex;
  font-size: 14px;
  gap: 10px;
  line-height: 22px;
  padding: 14px 16px;
}

.old-flow__notice::before {
  background: var(--join-tone-warning-fg);
  border-radius: 999px;
  content: "";
  flex: none;
  height: 8px;
  margin-top: 7px;
  width: 8px;
}

/* ── 表单字段 ─────────────────────────────────────── */
.old-flow__field {
  display: block;
  min-width: 0;
}

.old-flow__select-wrap {
  display: block;
  position: relative;
}

.old-flow__select-wrap .join-select {
  appearance: none;
  padding-right: 42px;
}

.old-flow__select-wrap::after {
  border-bottom: 2px solid var(--join-ink-muted);
  border-right: 2px solid var(--join-ink-muted);
  content: "";
  height: 9px;
  pointer-events: none;
  position: absolute;
  right: 18px;
  top: 50%;
  transform: translateY(-65%) rotate(45deg);
  width: 9px;
}

/* ── SSO：官方入口升级为渐变 CTA（保留 .secondary-button 选择器） ── */
.old-flow__sso.secondary-button {
  background: var(--join-gradient-cta);
  border-color: transparent;
  box-shadow: var(--join-cta-glow);
  color: #ffffff;
  width: 100%;
}

.old-flow__sso.secondary-button:hover:not(:disabled) {
  background: var(--join-gradient-cta);
  border-color: transparent;
  box-shadow: var(--join-cta-glow-hover);
  filter: brightness(1.06);
}

/* ── 邮箱验证码表单：嵌套玻璃区块 ─────────────────── */
.old-flow__form {
  display: grid;
  gap: 16px;
  padding: 18px;
}

.old-flow__pair {
  display: grid;
  gap: 14px;
}

.old-flow__send-row {
  display: grid;
  gap: 12px;
}

.old-flow__send {
  white-space: nowrap;
}

/* 学术邮箱模式下按钮保持可点击，仅做视觉降级（测试契约） */
.button-disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.old-flow .secondary-button.button-disabled:hover:not(:disabled) {
  background: var(--join-chip-bg);
  border-color: var(--join-glass-border);
}

/* 验证码输入区：虚线分隔出“第二步”，等宽字距增强可读性 */
.old-flow__code-field {
  border-top: 1px dashed var(--join-glass-border);
  padding-top: 16px;
}

.old-flow__code-field .join-input {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.18em;
}

.old-flow__verify {
  width: 100%;
}

/* ── 学籍匹配状态行：语义色气泡（class 钩子来自冻结的脚本绑定） ── */
.old-flow__match {
  align-items: flex-start;
  border-radius: var(--radius-lg);
  display: flex;
  font-size: 13px;
  font-weight: 600;
  gap: 8px;
  line-height: 20px;
  margin: 0;
  padding: 10px 14px;
}

.old-flow__match::before {
  background: currentColor;
  border-radius: 999px;
  content: "";
  flex: none;
  height: 7px;
  margin-top: 6px;
  width: 7px;
}

/* checking（脚本绑定 text-slate-600）→ 信息蓝 + 呼吸圆点 */
.old-flow__match.text-slate-600 {
  background: var(--join-tone-info-bg);
  color: var(--join-tone-info-fg);
}

.old-flow__match.text-slate-600::before {
  animation: old-flow-checking-pulse 1.2s ease-in-out infinite;
}

/* matched（text-green-700）→ 成功绿 */
.old-flow__match.text-green-700 {
  background: var(--join-tone-success-bg);
  color: var(--join-tone-success-fg);
}

/* waiting/mismatch/error（text-red-600）→ 警示红 */
.old-flow__match.text-red-600 {
  background: var(--join-tone-danger-bg);
  color: var(--join-tone-danger-fg);
}

@keyframes old-flow-checking-pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.35;
  }
}

/* ── 成功/失败反馈气泡 ────────────────────────────── */
.old-flow__feedback {
  border-radius: var(--radius-lg);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  margin: 0;
  padding: 10px 14px;
}

.old-flow__feedback--success {
  background: var(--join-tone-success-bg);
  color: var(--join-tone-success-fg);
}

.old-flow__feedback--danger {
  background: var(--join-tone-danger-bg);
  color: var(--join-tone-danger-fg);
}

/* ── ≥640px：双列学号/姓名 + 邮箱与发送按钮同排 ───── */
@media (min-width: 640px) {
  .old-flow__form {
    padding: 20px;
  }

  .old-flow__pair {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .old-flow__send-row {
    align-items: end;
    grid-template-columns: minmax(0, 1fr) auto;
  }
}
</style>
