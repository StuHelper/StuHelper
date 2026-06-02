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
          <h2 class="text-lg font-semibold">
            {{ consumedTokenNeedsLogin ? '继续入群认证' : '登录 StuHelper' }}
          </h2>
          <p v-if="consumedTokenNeedsLogin" class="mt-2 text-sm text-slate-600">
            该链接已绑定 StuHelper 账号，请使用首次绑定该链接的账号登录后继续认证。
          </p>
          <p v-else class="mt-2 text-sm text-slate-600">
            登录或注册后会回到当前认证链接。
          </p>
          <div class="mt-4 flex flex-wrap gap-3">
            <button class="primary-button" type="button" @click="startLogin">
              登录
            </button>
            <button
              v-if="!consumedTokenNeedsLogin"
              class="secondary-button"
              type="button"
              @click="startSignup"
            >
              注册
            </button>
          </div>
        </div>

        <div v-else-if="pageState === 'accountMismatch'" data-state="accountMismatch">
          <h2 class="text-lg font-semibold">账号不匹配</h2>
          <p class="mt-2 text-sm text-slate-600">
            该认证链接已绑定首次打开时登录的 StuHelper 账号。请退出当前账号后使用原账号登录，或联系管理员重新生成认证链接。
          </p>
          <button class="primary-button mt-4" type="button" @click="startReauthentication">
            重新登录
          </button>
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
          <div
            v-if="linkedResourceErrorMessage"
            class="mt-3 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800"
            data-linked-resource-error
          >
            <p>{{ linkedResourceErrorMessage }}</p>
            <button
              class="secondary-button mt-3"
              type="button"
              @click="retryLinkedResources"
            >
              重新加载
            </button>
          </div>
          <div class="mt-4 flex gap-2">
            <button
              class="flow-tab"
              :class="{ 'flow-tab--active': activeFlow === 'oldStudent' }"
              type="button"
              @click="selectAdmissionFlow('oldStudent')"
            >
              老生认证
            </button>
            <button
              class="flow-tab"
              :class="{ 'flow-tab--active': activeFlow === 'freshman' }"
              type="button"
              @click="selectAdmissionFlow('freshman')"
            >
              新生认证
            </button>
          </div>
          <OldStudentVerificationFlow
            v-if="activeFlow === 'oldStudent'"
            :current-return-url="currentAdmissionURL()"
            :linked="pageState === 'linked'"
            :schools="admissionSchools"
            @expired="handleAdmissionExpired"
            @verified="handleOldStudentVerified"
          />
          <FreshmanCameraFlow
            v-else-if="showFreshmanSubmission"
            :max-material-bytes="session?.maxMaterialBytes"
            :schools="admissionSchools"
            @expired="handleAdmissionExpired"
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

        <ProjectionPendingNotice
          v-else-if="pageState === 'projectionPending'"
          :timed-out="projectionRefreshTimedOut"
        />

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
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import {
  stateFromAdmissionMe,
  stateFromAdmissionSession,
  type AdmissionPageState,
} from '../admissionState'
import {
  buildAdmissionReturnURL,
  isAdmissionTokenConsumedError,
  isAdmissionSessionExpiredError,
  mapAdmissionApiError,
} from '../admissionToken'
import {
  shouldShowFreshmanSubmission,
  schoolHasAdmissionEmailOTP,
  schoolHasAdmissionSSO,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'
import { waitForAdmissionProjection } from '../projectionRefresh'
import FreshmanCameraFlow from './FreshmanCameraFlow.vue'
import OldStudentVerificationFlow from './OldStudentVerificationFlow.vue'
import ProjectionPendingNotice from './ProjectionPendingNotice.vue'

const PENDING_REVIEW_REFRESH_DELAYS_MS = [
  5000,
  10000,
  20000,
  30000,
] as const

const route = useRoute()
const auth = useAuthStore()
const verificationStore = useVerificationStore()
const pageState = ref<AdmissionPageState>('loading')
const session = ref<AdmissionSession | null>(null)
const admissionMe = ref<AdmissionMe | null>(null)
const errorMessage = ref('认证链接暂时无法打开，请稍后重试。')
const linkedResourceErrorMessage = ref('')
const consumedTokenNeedsLogin = ref(false)
const linking = ref(false)
const activeFlow = ref<'freshman' | 'oldStudent'>('freshman')
const projectionRefreshTimedOut = ref(false)
const activeFlowManuallySelected = ref(false)
let projectionRefreshAbort: AbortController | null = null
let admissionSessionLoad: Promise<void> | null = null
let admissionSessionReloadQueued = false
let activeAdmissionRouteKey = ''
let pendingReviewRefreshTimer: number | null = null
let pendingReviewRefreshAttempt = 0
let pendingReviewRefreshInFlight = false

const displayQQ = computed(() => readAdmissionQQ() ?? '')

function readAdmissionToken(): string {
  return String(route.params.code ?? '')
}

function readAdmissionQQ(): string | undefined {
  const qq = route.query.qq
  return typeof qq === 'string' && qq !== '' ? qq : undefined
}
const admissionSchools = computed(() => {
  return verificationStore.schools as AdmissionSchoolOption[]
})
const showFreshmanSubmission = computed(() => {
  return shouldShowFreshmanSubmission(admissionMe.value)
})
const hasOldStudentAdmissionMethod = computed(() => {
  return admissionSchools.value.some((school) => (
    schoolHasAdmissionSSO(school) || schoolHasAdmissionEmailOTP(school)
  ))
})

function applyError(error: unknown) {
  pageState.value = mapAdmissionApiError(error)
  if (pageState.value === 'error') {
    errorMessage.value = '认证链接暂时无法打开，请稍后重试。'
  }
}

function currentAdmissionURL(): string {
  return buildAdmissionReturnURL(route.fullPath)
}

function setAdmissionMe(nextAdmission: AdmissionMe): void {
  admissionMe.value = nextAdmission
  if (!showFreshmanSubmission.value) activeFlow.value = 'oldStudent'
}

async function loadLinkedResources(options?: { refreshAdmission?: boolean }): Promise<void> {
  const refreshAdmission = options?.refreshAdmission !== false
  linkedResourceErrorMessage.value = ''
  const [, nextAdmission] = await Promise.all([
    verificationStore.fetchSchools(),
    refreshAdmission
      ? admissionApi.getAdmissionMe()
      : Promise.resolve<AdmissionMe | null>(null),
  ])
  if (nextAdmission) {
    setAdmissionMe(nextAdmission)
    if (nextAdmission.session) {
      session.value = nextAdmission.session
    }
    pageState.value = stateFromAdmissionMe(nextAdmission)
  }
  if (pageState.value === 'linked') syncDefaultAdmissionFlow()
  if (pageState.value === 'projectionPending') scheduleProjectionRefresh()
}

function syncDefaultAdmissionFlow(): void {
  if (!showFreshmanSubmission.value) {
    activeFlow.value = 'oldStudent'
    return
  }
  if (activeFlowManuallySelected.value) return
  activeFlow.value = hasOldStudentAdmissionMethod.value ? 'oldStudent' : 'freshman'
}

function selectAdmissionFlow(flow: 'freshman' | 'oldStudent'): void {
  activeFlowManuallySelected.value = true
  activeFlow.value = flow
}

function scheduleLinkedResourcesLoad(options?: { refreshAdmission?: boolean }): void {
  loadLinkedResources(options).catch(handleLinkedResourcesLoadError)
}

function handleLinkedResourcesLoadError(error: unknown): void {
  if (isAdmissionSessionExpiredError(error)) {
    handleAdmissionExpired()
    return
  }
  if (pageState.value === 'linked') {
    linkedResourceErrorMessage.value = readErrorMessage(
      error,
      '认证资料加载失败，请稍后重试。',
    )
    return
  }
  applyError(error)
}

function handleSessionState(nextSession: AdmissionSession): void {
  pageState.value = stateFromAdmissionSession(nextSession)
  if (pageState.value === 'linked') scheduleLinkedResourcesLoad()
  if (pageState.value === 'projectionPending') scheduleProjectionRefresh()
}

function handleAdmissionMeState(nextAdmission: AdmissionMe): void {
  setAdmissionMe(nextAdmission)
  if (nextAdmission.session) {
    session.value = nextAdmission.session
  }
  pageState.value = stateFromAdmissionMe(nextAdmission)
  if (pageState.value === 'linked') scheduleLinkedResourcesLoad({ refreshAdmission: false })
  if (pageState.value === 'projectionPending') scheduleProjectionRefresh()
}

async function resumeConsumedTokenSession(authenticated = auth.isAuthenticated): Promise<boolean> {
  if (!authenticated) {
    return false
  }
  const linked = await admissionApi.linkAdmissionSession(
    readAdmissionToken(),
    readAdmissionQQ(),
  )
  session.value = linked
  handleSessionState(linked)
  return true
}

async function refreshAdmissionAuthState(): Promise<boolean> {
  try {
    return await auth.bootstrapSession({ force: true })
  } catch {
    return auth.isAuthenticated
  }
}

function scheduleProjectionRefresh(): void {
  projectionRefreshTimedOut.value = false
  projectionRefreshAbort?.abort()
  projectionRefreshAbort = new AbortController()
  waitForAdmissionProjection({
    refreshAuth: auth.fetchUser,
    signal: projectionRefreshAbort.signal,
  })
    .then((ready) => {
      if (ready) pageState.value = 'approved'
      else projectionRefreshTimedOut.value = true
    })
    .catch((error) => {
      if (!isAbortError(error)) applyError(error)
    })
}

function schedulePendingReviewRefresh(reset = false): void {
  if (pageState.value !== 'pendingReview') return
  if (reset) pendingReviewRefreshAttempt = 0
  if (pendingReviewRefreshTimer !== null || pendingReviewRefreshInFlight) return
  if (document.visibilityState !== 'visible') return
  const delay = pendingReviewRefreshDelay()
  pendingReviewRefreshAttempt += 1
  pendingReviewRefreshTimer = window.setTimeout(() => {
    pendingReviewRefreshTimer = null
    void refreshPendingReviewState()
  }, delay)
}

function pendingReviewRefreshDelay(): number {
  const index = Math.min(
    pendingReviewRefreshAttempt,
    PENDING_REVIEW_REFRESH_DELAYS_MS.length - 1,
  )
  return PENDING_REVIEW_REFRESH_DELAYS_MS[index]
}

function clearPendingReviewRefresh(): void {
  if (pendingReviewRefreshTimer === null) return
  window.clearTimeout(pendingReviewRefreshTimer)
  pendingReviewRefreshTimer = null
}

function refreshPendingReviewAfterBrowserReturn(): void {
  clearPendingReviewRefresh()
  void refreshPendingReviewState()
}

async function refreshPendingReviewState(): Promise<void> {
  if (pageState.value !== 'pendingReview' || pendingReviewRefreshInFlight) return
  pendingReviewRefreshInFlight = true
  try {
    const nextAdmission = await admissionApi.getAdmissionMe()
    if (pageState.value === 'pendingReview') {
      handleAdmissionMeState(nextAdmission)
    }
  } catch (error) {
    if (pageState.value === 'pendingReview' && isAdmissionSessionExpiredError(error)) {
      handleAdmissionExpired()
    }
  } finally {
    pendingReviewRefreshInFlight = false
    if (pageState.value === 'pendingReview') {
      schedulePendingReviewRefresh()
    }
  }
}

function isCurrentAdmissionRoute(requestToken: string, requestQQ: string | undefined): boolean {
  return currentAdmissionRouteKey() === admissionRouteKey(requestToken, requestQQ)
}

function admissionRouteKey(routeToken: string, qq: string | undefined): string {
  return `${routeToken}\u0000${qq ?? ''}`
}

function currentAdmissionRouteKey(): string {
  return admissionRouteKey(readAdmissionToken(), readAdmissionQQ())
}

async function loadAdmissionSession() {
  const requestToken = readAdmissionToken()
  const requestQQ = readAdmissionQQ()
  activeAdmissionRouteKey = admissionRouteKey(requestToken, requestQQ)
  pageState.value = 'loading'
  consumedTokenNeedsLogin.value = false
  try {
    const preview = await admissionApi.getAdmissionSession(
      requestToken,
      requestQQ,
    )
    if (!isCurrentAdmissionRoute(requestToken, requestQQ)) return
    session.value = preview
    const authenticated = await refreshAdmissionAuthState()
    if (!isCurrentAdmissionRoute(requestToken, requestQQ)) return
    if (authenticated) handleSessionState(preview)
    else pageState.value = 'needsLogin'
  } catch (error) {
    if (!isCurrentAdmissionRoute(requestToken, requestQQ)) return
    if (isAdmissionTokenConsumedError(error)) {
      const authenticated = await refreshAdmissionAuthState()
      if (!isCurrentAdmissionRoute(requestToken, requestQQ)) return
      if (!authenticated) {
        consumedTokenNeedsLogin.value = true
        pageState.value = 'needsLogin'
        return
      }
      try {
        if (await resumeConsumedTokenSession(authenticated)) {
          return
        }
      } catch (resumeError) {
        if (isAdmissionTokenConsumedError(resumeError)) {
          pageState.value = 'accountMismatch'
          return
        }
        applyError(resumeError)
        return
      }
    }
    applyError(error)
  }
}

function queueAdmissionSessionLoad(): void {
  if (admissionSessionLoad) {
    admissionSessionReloadQueued = true
    return
  }
  admissionSessionLoad = loadAdmissionSession().finally(() => {
    admissionSessionLoad = null
    if (admissionSessionReloadQueued) {
      admissionSessionReloadQueued = false
      queueAdmissionSessionLoad()
    }
  })
}

function shouldRefreshAfterBrowserReturn(): boolean {
  return (
    (pageState.value === 'loading' && activeAdmissionRouteKey !== currentAdmissionRouteKey()) ||
    pageState.value === 'needsLogin' ||
    pageState.value === 'accountMismatch' ||
    pageState.value === 'error'
  )
}

function refreshAfterBrowserReturn(): void {
  if (pageState.value === 'pendingReview') {
    refreshPendingReviewAfterBrowserReturn()
    return
  }
  if (shouldRefreshAfterBrowserReturn()) {
    queueAdmissionSessionLoad()
  }
}

function handlePageShow(event: PageTransitionEvent): void {
  if (event.persisted || shouldRefreshAfterBrowserReturn()) {
    refreshAfterBrowserReturn()
  }
}

function handleVisibilityChange(): void {
  if (document.visibilityState === 'visible') {
    refreshAfterBrowserReturn()
  } else {
    clearPendingReviewRefresh()
  }
}

function handleWindowFocus(): void {
  refreshAfterBrowserReturn()
}

async function confirmLink() {
  linking.value = true
  try {
    const linked = await admissionApi.linkAdmissionSession(
      readAdmissionToken(),
      readAdmissionQQ(),
    )
    session.value = linked
    handleSessionState(linked)
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

function startReauthentication() {
  void auth.reauthenticate(currentAdmissionURL())
}

function markPendingReview() {
  pageState.value = 'pendingReview'
}

function handleAdmissionExpired() {
  pageState.value = 'expired'
}

function handleOldStudentVerified(nextAdmission: AdmissionMe) {
  handleAdmissionMeState(nextAdmission)
}

function retryLinkedResources() {
  scheduleLinkedResourcesLoad()
}

function readErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}

onMounted(() => {
  window.addEventListener('pageshow', handlePageShow)
  window.addEventListener('focus', handleWindowFocus)
  document.addEventListener('visibilitychange', handleVisibilityChange)
  queueAdmissionSessionLoad()
})

watch(
  () => route.fullPath,
  () => {
    activeFlowManuallySelected.value = false
    queueAdmissionSessionLoad()
  },
)

watch(pageState, (state) => {
  if (state === 'pendingReview') {
    schedulePendingReviewRefresh(true)
    return
  }
  clearPendingReviewRefresh()
})

onBeforeUnmount(() => {
  projectionRefreshAbort?.abort()
  clearPendingReviewRefresh()
  window.removeEventListener('pageshow', handlePageShow)
  window.removeEventListener('focus', handleWindowFocus)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError'
}
</script>

<style scoped src="./AdmissionPage.css"></style>
