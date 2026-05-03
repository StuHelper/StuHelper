<template>
  <section class="mt-5 border-t border-slate-200 pt-5" data-admission-old-student-flow>
    <div v-if="schools.length === 0" class="rounded-lg bg-slate-50 p-4 text-sm text-slate-600">
      暂无可用学校认证配置。
    </div>

    <div v-else class="grid gap-4">
      <label class="field-label">
        学校
        <select
          :value="selectedSchoolID ?? ''"
          class="field-control"
          data-school-select
          @change="updateSchoolID"
        >
          <option
            v-for="school in schools"
            :key="school.schoolID"
            :value="String(school.schoolID)"
          >
            {{ school.schoolName }}
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
        v-if="linked"
        class="grid gap-3 rounded-lg border border-slate-200 bg-slate-50 p-4"
        data-school-email-otp-form
        @submit.prevent="verifyEmailOTP"
      >
        <label class="field-label">
          学校邮箱
          <input
            :value="email"
            class="field-control"
            type="email"
            @input="updateEmail"
          >
        </label>
        <div class="flex flex-wrap gap-3">
          <button class="secondary-button" type="button" @click="requestEmailOTP">
            发送验证码
          </button>
        </div>
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
        <button class="primary-button" type="submit" :disabled="submitting">
          {{ submitting ? '校验中...' : '验证邮箱' }}
        </button>
      </form>

      <div v-else class="rounded-lg bg-slate-50 p-4 text-sm text-slate-600">
        请先确认绑定当前 QQ 后再进行老生认证。
      </div>

      <p v-if="errorMessage" class="text-sm text-red-600">{{ errorMessage }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import type { AdmissionMe } from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import {
  buildSchoolSSOLoginPath,
  schoolHasAdmissionSSO,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'

const props = defineProps<{
  currentReturnUrl: string
  linked: boolean
  schools: AdmissionSchoolOption[]
}>()

const emit = defineEmits<{
  verified: [admission: AdmissionMe]
}>()

const selectedSchoolID = ref<number | null>(null)
const email = ref('')
const code = ref('')
const submitting = ref(false)
const errorMessage = ref('')

const selectedSchool = computed(() => {
  return props.schools.find((school) => school.schoolID === selectedSchoolID.value)
    ?? null
})
const selectedSchoolHasSSO = computed(() => schoolHasAdmissionSSO(selectedSchool.value))

watch(
  () => props.schools,
  (schools) => {
    if (selectedSchoolID.value !== null) return
    selectedSchoolID.value = schools[0]?.schoolID ?? null
  },
  { immediate: true },
)

function updateSchoolID(event: Event): void {
  const value = readControlValue(event)
  selectedSchoolID.value = value ? Number(value) : null
}

function updateEmail(event: Event): void {
  email.value = readControlValue(event).trim()
}

function updateCode(event: Event): void {
  code.value = readControlValue(event).trim()
}

function startSchoolSSO(): void {
  const schoolID = requireSelectedSchoolID()
  window.location.href = buildSchoolSSOLoginPath(schoolID, props.currentReturnUrl)
}

async function requestEmailOTP(): Promise<void> {
  errorMessage.value = ''
  try {
    await admissionApi.requestSchoolEmailOTP({
      schoolID: requireSelectedSchoolID(),
      email: requireEmail(),
    })
  } catch (error) {
    errorMessage.value = readErrorMessage(error, '验证码发送失败。')
  }
}

async function verifyEmailOTP(): Promise<void> {
  errorMessage.value = ''
  submitting.value = true
  try {
    const admission = await admissionApi.verifySchoolEmailOTP({
      schoolID: requireSelectedSchoolID(),
      email: requireEmail(),
      code: requireCode(),
    })
    emit('verified', admission)
  } catch (error) {
    errorMessage.value = readErrorMessage(error, '邮箱验证失败。')
  } finally {
    submitting.value = false
  }
}

function requireSelectedSchoolID(): number {
  if (!selectedSchoolID.value) {
    throw new Error('请选择学校')
  }
  return selectedSchoolID.value
}

function requireEmail(): string {
  if (!email.value) {
    throw new Error('请填写学校邮箱')
  }
  return email.value
}

function requireCode(): string {
  if (!code.value) {
    throw new Error('请填写验证码')
  }
  return code.value
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
</style>
