<template>
  <section class="mt-5 border-t border-slate-200 pt-5" data-admission-old-student-flow>
    <div v-if="schools.length === 0" class="rounded-lg bg-slate-50 p-4 text-sm text-slate-600">
      暂无可用学校认证配置。
    </div>

    <div v-else class="grid gap-4">
      <label class="field-label">
        学校
        <select
          :value="selectedSchoolCode"
          class="field-control"
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
      </label>

      <button
        v-if="selectedSchoolHasSSO"
        class="secondary-button"
        data-school-sso-button
        type="button"
        @click="startSchoolSSO"
      >
        学校官方 SSO
      </button>

      <form
        v-if="linked && selectedSchoolHasEmailOTP"
        class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4"
        data-school-email-otp-form
        @submit.prevent="verifyEmailOTP"
      >
        <div v-if="selectedSchoolRequiresAcademicEmail" class="grid gap-3 sm:grid-cols-2">
          <label class="field-label">
            学号
            <input
              :value="studentID"
              class="field-control"
              data-academic-student-id-input
              type="text"
              @input="updateStudentID"
            >
          </label>
          <label class="field-label">
            姓名
            <input
              :value="studentName"
              class="field-control"
              data-academic-student-name-input
              type="text"
              @input="updateStudentName"
            >
          </label>
        </div>
        <p
          v-if="selectedSchoolRequiresAcademicEmail && academicMatchMessage"
          :class="academicMatchMessageClass"
          data-academic-match-status
        >
          {{ academicMatchMessage }}
        </p>
        <label class="field-label">
          学校邮箱
          <input
            :value="email"
            class="field-control"
            data-academic-email-input
            type="email"
            :readonly="selectedSchoolRequiresAcademicEmail"
            :placeholder="selectedSchoolRequiresAcademicEmail ? '学号和姓名校验通过后自动填写' : ''"
            @input="updateEmail"
          >
        </label>
        <div class="flex flex-wrap gap-3">
          <button
            class="secondary-button"
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
        <p v-if="successMessage" class="text-sm text-green-700">{{ successMessage }}</p>
        <label class="field-label">
          验证码
          <input
            :value="code"
            class="field-control"
            inputmode="numeric"
            type="text"
            @input="updateCode"
          >
        </label>
        <button
          class="primary-button"
          data-school-email-otp-verify
          type="submit"
          :disabled="!canVerifyEmailOTP"
        >
          {{ submitting ? '校验中...' : '验证邮箱' }}
        </button>
      </form>

      <div v-else-if="!linked" class="rounded-lg bg-slate-50 p-4 text-sm text-slate-600">
        请先确认绑定当前 QQ 后再进行老生认证。
      </div>

      <div v-else class="rounded-lg bg-slate-50 p-4 text-sm text-slate-600">
        当前学校暂未开通加群老生认证方式。
      </div>

      <p v-if="errorMessage" class="text-sm text-red-600">{{ errorMessage }}</p>
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
.field-label {
  color: #334155;
  display: grid;
  font-size: 14px;
  font-weight: 600;
  gap: 6px;
}

.field-control {
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  color: #0f172a;
  min-height: 40px;
  padding: 8px 10px;
}

.button-disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.primary-button:disabled,
.secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
</style>
