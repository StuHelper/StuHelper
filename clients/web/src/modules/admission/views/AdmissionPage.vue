<template>
  <main class="min-h-screen bg-slate-50 px-4 py-8 text-slate-950">
    <section class="mx-auto flex w-full max-w-2xl flex-col gap-5">
      <header class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <p class="text-sm font-medium text-slate-500">StuHelper</p>
        <h1 class="mt-2 text-2xl font-semibold">入群身份认证</h1>
        <p v-if="displayQQ" class="mt-2 text-sm text-slate-600">
          QQ：{{ displayQQ }}
        </p>
      </header>
      <section class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <div v-if="pageState === 'loading'" data-state="loading">
          <p class="text-sm text-slate-600">正在校验链接...</p>
        </div>

        <div v-else-if="pageState === 'needsLogin'" data-state="needsLogin">
          <h2 class="text-lg font-semibold">登录 StuHelper</h2>
          <p class="mt-2 text-sm text-slate-600">
            登录或注册后会回到当前认证链接。
          </p>
          <div class="mt-4 flex flex-wrap gap-3">
            <button class="primary-button" type="button" @click="startLogin">
              登录
            </button>
            <button class="secondary-button" type="button" @click="startSignup">
              注册
            </button>
          </div>
        </div>

        <div v-else-if="pageState === 'qqMismatch'" data-state="qqMismatch">
          <h2 class="text-lg font-semibold">链接被篡改</h2>
          <p class="mt-2 text-sm text-slate-600">
            链接被篡改，请联系管理员重新发放。
          </p>
        </div>

        <div v-else-if="pageState === 'ready'" data-state="ready">
          <h2 class="text-lg font-semibold">确认绑定当前 QQ</h2>
          <p class="mt-2 text-sm text-slate-600">
            确认后会把当前 StuHelper 账号与本次入群 QQ 认证会话绑定。
          </p>
          <button
            class="primary-button mt-4"
            type="button"
            :disabled="linking"
            @click="confirmLink"
          >
            {{ linking ? '正在确认...' : '开始认证' }}
          </button>
        </div>

        <div v-else-if="pageState === 'linked'" data-state="linked">
          <h2 class="text-lg font-semibold">选择认证方式</h2>
          <div class="mt-4 flex gap-2">
            <button
              class="flow-tab"
              :class="{ 'flow-tab--active': activeFlow === 'oldStudent' }"
              type="button"
              @click="activeFlow = 'oldStudent'"
            >
              老生认证
            </button>
            <button
              class="flow-tab"
              :class="{ 'flow-tab--active': activeFlow === 'freshman' }"
              type="button"
              @click="activeFlow = 'freshman'"
            >
              新生认证
            </button>
          </div>
          <OldStudentVerificationFlow
            v-if="activeFlow === 'oldStudent'"
            :current-return-url="currentAdmissionURL()"
            :linked="pageState === 'linked'"
            :schools="admissionSchools"
            @verified="handleOldStudentVerified"
          />
          <FreshmanCameraFlow
            v-else-if="showFreshmanSubmission"
            :max-material-bytes="session?.maxMaterialBytes"
            @submitted="markPendingReview"
          />
          <div
            v-else
            class="mt-5 rounded-lg border border-slate-200 bg-slate-50 p-4 text-sm text-slate-700"
            data-formal-student-credential
          >
            已完成老生认证。
          </div>
        </div>

        <div v-else-if="pageState === 'pendingReview'" data-state="pendingReview">
          <h2 class="text-lg font-semibold">等待管理员审核</h2>
          <p class="mt-2 text-sm text-slate-600">
            材料已提交，请等待管理员处理。
          </p>
        </div>

        <div v-else-if="pageState === 'approved'" data-state="approved">
          <h2 class="text-lg font-semibold">认证已通过</h2>
          <p class="mt-2 text-sm text-slate-600">
            群内禁言会由机器人自动解除。
          </p>
        </div>

        <div v-else-if="pageState === 'expired'" data-state="expired">
          <h2 class="text-lg font-semibold">链接已失效</h2>
          <p class="mt-2 text-sm text-slate-600">
            请回到 QQ 群联系管理员重新发放认证链接。
          </p>
        </div>

        <div v-else data-state="error">
          <h2 class="text-lg font-semibold">无法打开认证</h2>
          <p class="mt-2 text-sm text-slate-600">{{ errorMessage }}</p>
        </div>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import { buildAdmissionReturnURL, mapAdmissionApiError } from '../admissionToken'
import {
  shouldShowFreshmanSubmission,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'
import FreshmanCameraFlow from './FreshmanCameraFlow.vue'
import OldStudentVerificationFlow from './OldStudentVerificationFlow.vue'

type AdmissionPageState =
  | 'loading'
  | 'needsLogin'
  | 'qqMismatch'
  | 'ready'
  | 'linked'
  | 'pendingReview'
  | 'approved'
  | 'expired'
  | 'error'

const route = useRoute()
const auth = useAuthStore()
const verificationStore = useVerificationStore()
const pageState = ref<AdmissionPageState>('loading')
const session = ref<AdmissionSession | null>(null)
const admissionMe = ref<AdmissionMe | null>(null)
const errorMessage = ref('认证链接暂时无法打开，请稍后重试。')
const linking = ref(false)
const activeFlow = ref<'freshman' | 'oldStudent'>('freshman')

const token = computed(() => String(route.params.code ?? ''))
const displayQQ = computed(() => {
  const qq = route.query.qq
  return typeof qq === 'string' ? qq : ''
})
const admissionSchools = computed(() => {
  return verificationStore.schools as AdmissionSchoolOption[]
})
const showFreshmanSubmission = computed(() => {
  return shouldShowFreshmanSubmission(admissionMe.value)
})

function stateFromSession(nextSession: AdmissionSession): AdmissionPageState {
  if (nextSession.status === 'joined_muted') return 'ready'
  if (nextSession.status === 'linked') return 'linked'
  if (nextSession.status === 'material_submitted') return 'pendingReview'
  if (nextSession.status === 'verified') return 'approved'
  if (nextSession.status === 'expired_kicked') return 'expired'
  return 'error'
}

function applyError(error: unknown) {
  pageState.value = mapAdmissionApiError(error)
  if (pageState.value === 'error') {
    errorMessage.value = '认证链接暂时无法打开，请稍后重试。'
  }
}

function currentAdmissionURL(): string {
  return buildAdmissionReturnURL(route.fullPath)
}

async function loadLinkedResources(): Promise<void> {
  await Promise.all([
    verificationStore.fetchSchools(),
    admissionApi.getAdmissionMe().then((admission) => {
      admissionMe.value = admission
      if (!showFreshmanSubmission.value) activeFlow.value = 'oldStudent'
    }),
  ])
}

function scheduleLinkedResourcesLoad(): void {
  loadLinkedResources().catch(applyError)
}

async function loadAdmissionSession() {
  pageState.value = 'loading'
  try {
    const preview = await admissionApi.getAdmissionSession(
      token.value,
      displayQQ.value || undefined,
    )
    session.value = preview
    pageState.value = auth.isAuthenticated ? stateFromSession(preview) : 'needsLogin'
    if (pageState.value === 'linked') scheduleLinkedResourcesLoad()
  } catch (error) {
    applyError(error)
  }
}

async function confirmLink() {
  linking.value = true
  try {
    const linked = await admissionApi.linkAdmissionSession(
      token.value,
      displayQQ.value || undefined,
    )
    session.value = linked
    pageState.value = stateFromSession(linked)
    if (pageState.value === 'linked') scheduleLinkedResourcesLoad()
  } catch (error) {
    applyError(error)
  } finally {
    linking.value = false
  }
}

function startLogin() {
  void auth.login(currentAdmissionURL())
}

function startSignup() {
  void auth.signup(currentAdmissionURL())
}

function markPendingReview() {
  pageState.value = 'pendingReview'
}

function handleOldStudentVerified(nextAdmission: AdmissionMe) {
  admissionMe.value = nextAdmission
  pageState.value = nextAdmission.status === 'verified' ? 'approved' : 'linked'
}

onMounted(() => {
  void loadAdmissionSession()
})
</script>

<style scoped>
.primary-button,
.secondary-button {
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  min-height: 40px;
  padding: 10px 16px;
}

.primary-button {
  background: #0f172a;
  color: #ffffff;
}

.primary-button:disabled { cursor: not-allowed; opacity: 0.6; }

.secondary-button {
  background: #e2e8f0;
  color: #0f172a;
}

.flow-tab {
  border: 1px solid #cbd5e1;
  border-radius: 8px;
  color: #334155;
  font-size: 14px;
  font-weight: 600;
  min-height: 36px;
  padding: 8px 12px;
}

.flow-tab--active {
  background: #0f172a;
  border-color: #0f172a;
  color: #ffffff;
}
</style>
