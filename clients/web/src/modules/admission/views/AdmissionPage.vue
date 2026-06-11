<template>
  <AdmissionShell
    :display-qq="displayQQ"
    :status-label="admissionStatusLabel"
  >
    <template #progress>
      <AdmissionProgress
        :page-state="pageState"
        :session="session"
      />
    </template>

    <AdmissionStatePanel
      v-if="pageState === 'loading'"
      :icon="CircleDashed"
      state="loading"
      title="正在校验链接"
      tone="info"
      description="正在读取本次入群认证会话，请保持页面打开。"
    />

    <AdmissionStatePanel
      v-else-if="pageState === 'needsLogin'"
      :description="consumedTokenNeedsLogin
        ? '该链接已绑定 StuHelper 账号，请使用首次绑定该链接的账号登录后继续认证。'
        : '登录或注册后会回到当前认证链接。'"
      :icon="LogIn"
      state="needsLogin"
      :title="consumedTokenNeedsLogin ? '继续入群认证' : '登录 StuHelper'"
      tone="info"
    >
      <template #actions>
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
      </template>
    </AdmissionStatePanel>

    <AdmissionStatePanel
      v-else-if="pageState === 'accountMismatch'"
      :icon="UserX"
      state="accountMismatch"
      title="账号不匹配"
      tone="danger"
      description="该认证链接已绑定首次打开时登录的 StuHelper 账号。请退出当前账号后使用原账号登录，或联系管理员重新生成认证链接。"
    >
      <template #help>
        <AdmissionReissueHint
          :command="reissueCommand"
          @copy="copyReissueCommand"
        />
      </template>
      <template #actions>
        <button class="primary-button" type="button" @click="startAccountSwitch">
          重新登录
        </button>
      </template>
    </AdmissionStatePanel>

    <AdmissionStatePanel
      v-else-if="pageState === 'qqMismatch'"
      :description="[
        `这条认证链接属于 QQ ${displayQQ || '当前入群 QQ'}。当前登录的 StuHelper 账号已绑定其他 QQ，不能用于认证这个入群申请。`,
        '请退出后登录或注册属于该 QQ 的 StuHelper 账号，或联系管理员重新生成认证链接。',
      ]"
      :icon="UserX"
      state="qqMismatch"
      title="QQ 账号不匹配"
      tone="danger"
    >
      <template #help>
        <AdmissionReissueHint
          :command="reissueCommand"
          @copy="copyReissueCommand"
        />
      </template>
      <template #actions>
        <button class="primary-button" type="button" @click="startAccountSwitch">
          重新登录
        </button>
      </template>
    </AdmissionStatePanel>

    <AdmissionStatePanel
      v-else-if="pageState === 'ready'"
      :icon="Link2"
      state="ready"
      title="确认绑定当前 QQ"
      tone="warning"
      description="确认后会把当前 StuHelper 账号与本次入群 QQ 认证会话绑定。"
    >
      <template #actions>
        <button
          class="primary-button"
          data-admission-open-bind-confirmation
          type="button"
          :disabled="linking"
          @click="openBindConfirmationDialog"
        >
          {{ linking ? '正在确认...' : '开始认证' }}
        </button>
      </template>
    </AdmissionStatePanel>

    <section v-else-if="pageState === 'linked'" class="admission-flow" data-state="linked">
      <div class="admission-flow__header">
        <span class="admission-flow__icon join-tone-success" aria-hidden="true">
          <ShieldCheck class="admission-flow__icon-svg" />
        </span>
        <div class="admission-flow__heading-copy">
          <h2>选择认证方式</h2>
          <p>当前 QQ 会话已绑定，请选择符合你身份的学生认证方式。</p>
        </div>
      </div>
      <div
        v-if="linkedResourceErrorMessage"
        class="linked-resource-error"
        data-linked-resource-error
      >
        <p>{{ linkedResourceErrorMessage }}</p>
        <button
          class="secondary-button"
          type="button"
          @click="retryLinkedResources"
        >
          重新加载
        </button>
      </div>
      <div class="admission-flow__tabs">
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
        :admission-session-id="session?.id"
        :current-return-url="currentAdmissionURL()"
        :linked="pageState === 'linked'"
        :schools="admissionSchools"
        @expired="handleAdmissionExpired"
        @verified="handleOldStudentVerified"
      />
      <FreshmanCameraFlow
        v-else-if="showFreshmanSubmission"
        :admission-session-id="session?.id"
        :max-material-bytes="session?.maxMaterialBytes"
        :schools="admissionSchools"
        @expired="handleAdmissionExpired"
        @submitted="markPendingReview"
      />
      <div
        v-else
        class="formal-student-credential join-chip"
        data-formal-student-credential
      >
        已完成老生认证。
      </div>
    </section>

    <AdmissionStatePanel
      v-else-if="pageState === 'pendingReview'"
      :icon="Hourglass"
      state="pendingReview"
      title="等待管理员审核"
      tone="warning"
      description="材料已提交，请等待管理员处理。页面会在可见时自动检查最新状态。"
    />

    <ProjectionPendingNotice
      v-else-if="pageState === 'projectionPending'"
      :timed-out="projectionRefreshTimedOut"
      @retry="retryProjectionRefresh"
    />

    <AdmissionStatePanel
      v-else-if="pageState === 'approved'"
      :icon="CheckCircle2"
      state="approved"
      title="认证已通过"
      tone="success"
      description="群内禁言会由机器人自动解除。"
    />

    <AdmissionStatePanel
      v-else-if="pageState === 'invalid'"
      :description="[
        '这个链接不存在、已经被替换，或不是群内机器人/管理员生成的最新认证链接。请回到 QQ 群使用最新链接。',
        '如果仍无法打开，请联系管理员重新生成认证链接。',
      ]"
      :icon="XCircle"
      state="invalid"
      title="认证链接无效"
      tone="danger"
    >
      <template #help>
        <AdmissionReissueHint
          :command="reissueCommand"
          @copy="copyReissueCommand"
        />
      </template>
    </AdmissionStatePanel>

    <AdmissionStatePanel
      v-else-if="pageState === 'expired'"
      :icon="XCircle"
      state="expired"
      title="链接已失效"
      tone="warning"
      description="请回到 QQ 群联系管理员重新生成认证链接。"
    >
      <template #help>
        <AdmissionReissueHint
          :command="reissueCommand"
          @copy="copyReissueCommand"
        />
      </template>
    </AdmissionStatePanel>

    <AdmissionStatePanel
      v-else
      :description="errorMessage"
      :icon="AlertTriangle"
      state="error"
      title="无法打开认证"
      tone="danger"
    />
  </AdmissionShell>

  <AdmissionBindConfirmationDialog
    :current-user-label="currentUserLabel"
    :display-qq="displayQQ"
    :error-message="bindConfirmationError"
    :linking="linking"
    :matches="bindConfirmationMatches"
    :open="bindConfirmationDialogOpen"
    :qq="bindConfirmationQQ"
    @cancel="closeBindConfirmationDialog"
    @submit="submitBindConfirmation"
    @touch="bindConfirmationTouched = true"
    @update:open="handleBindConfirmationOpenChange"
    @update:qq="bindConfirmationQQ = $event"
  />
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  AlertTriangle,
  CheckCircle2,
  CircleDashed,
  Hourglass,
  Link2,
  LogIn,
  ShieldCheck,
  UserX,
  XCircle,
} from 'lucide-vue-next'

import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { consumeAdmissionAuthReturn } from '@/utils/auth'
import { hasStoredSessionHint } from '@/utils/sessionHint'
import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

import { admissionApi } from '../api'
import {
  stateFromAdmissionMe,
  stateFromAdmissionSession,
  type AdmissionPageState,
} from '../admissionState'
import {
  buildAdmissionReturnURL,
  forgetLinkedAdmissionSession,
  isAdmissionTokenConsumedError,
  isAdmissionSessionExpiredError,
  mapAdmissionApiError,
  readLinkedAdmissionSessionID,
  rememberLinkedAdmissionSession,
} from '../admissionToken'
import {
  shouldShowFreshmanSubmission,
  schoolHasAdmissionEmailOTP,
  schoolHasAdmissionSSO,
  type AdmissionSchoolOption,
} from '../oldStudentAdmission'
import { waitForAdmissionProjection } from '../projectionRefresh'
import FreshmanCameraFlow from './FreshmanCameraFlow.vue'
import AdmissionBindConfirmationDialog from './AdmissionBindConfirmationDialog.vue'
import AdmissionReissueHint from './AdmissionReissueHint.vue'
import AdmissionProgress from './AdmissionProgress.vue'
import AdmissionShell from './AdmissionShell.vue'
import AdmissionStatePanel from './AdmissionStatePanel.vue'
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
const toast = useToast()
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
const bindConfirmationDialogOpen = ref(false)
const bindConfirmationQQ = ref('')
const bindConfirmationTouched = ref(false)
let projectionRefreshAbort: AbortController | null = null
let admissionSessionLoad: Promise<void> | null = null
let admissionSessionReloadQueued = false
let activeAdmissionRouteKey = ''
let pendingReviewRefreshTimer: number | null = null
let pendingReviewRefreshAttempt = 0
let pendingReviewRefreshInFlight = false

const displayQQ = computed(() => session.value?.qqID ?? '')
const currentUserLabel = computed(() => (
  auth.user?.displayName || auth.user?.name || '当前 StuHelper 账号'
))
const bindConfirmationExpectedQQ = computed(() => displayQQ.value.trim())
const normalizedBindConfirmationQQ = computed(() => bindConfirmationQQ.value.trim())
const bindConfirmationMatches = computed(() => (
  bindConfirmationExpectedQQ.value !== '' &&
  normalizedBindConfirmationQQ.value === bindConfirmationExpectedQQ.value
))
const bindConfirmationError = computed(() => {
  if (!bindConfirmationTouched.value || normalizedBindConfirmationQQ.value === '') {
    return ''
  }
  return bindConfirmationMatches.value ? '' : '输入的 QQ 号与本次入群 QQ 不一致。'
})
const reissueCommand = computed(() => {
  return displayQQ.value
    ? `重新生成认证链接 ${displayQQ.value}`
    : '重新生成认证链接 <QQ号>'
})
const admissionStatusLabel = computed(() => {
  switch (pageState.value) {
    case 'loading':
      return '校验链接'
    case 'needsLogin':
      return consumedTokenNeedsLogin.value ? '等待原账号登录' : '等待登录'
    case 'accountMismatch':
      return '账号不匹配'
    case 'qqMismatch':
      return 'QQ 不匹配'
    case 'ready':
      return '等待确认 QQ'
    case 'linked':
      return '选择认证方式'
    case 'pendingReview':
      return '等待审核'
    case 'projectionPending':
      return '身份同步中'
    case 'approved':
      return '认证通过'
    case 'invalid':
      return '链接无效'
    case 'expired':
      return '链接失效'
    case 'error':
      return '加载失败'
    default:
      return '处理中'
  }
})

function readAdmissionToken(): string {
  return String(route.params.code ?? '')
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
  if (nextAdmission.session?.id) {
    rememberLinkedAdmissionSession(readAdmissionToken(), nextAdmission.session.id)
  }
  if (!showFreshmanSubmission.value) activeFlow.value = 'oldStudent'
}

async function loadLinkedResources(options?: { refreshAdmission?: boolean }): Promise<void> {
  const refreshAdmission = options?.refreshAdmission !== false
  linkedResourceErrorMessage.value = ''
  const [, nextAdmission] = await Promise.all([
    verificationStore.fetchSchools(),
    refreshAdmission
      ? admissionApi.getAdmissionMe(session.value?.id)
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

async function applyKnownQQMismatch(nextSession: AdmissionSession): Promise<boolean> {
  if (nextSession.status !== 'joined_muted' || !nextSession.qqID) {
    return false
  }
  try {
    const currentBinding = await verificationStore.fetchQQBinding()
    if (currentBinding && currentBinding.qqID !== nextSession.qqID) {
      pageState.value = 'qqMismatch'
      return true
    }
  } catch {
    return false
  }
  return false
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
  )
  session.value = linked
  rememberLinkedAdmissionSession(readAdmissionToken(), linked.id)
  handleSessionState(linked)
  return true
}

async function resumeRememberedAdmissionSession(requestToken: string): Promise<boolean> {
  const rememberedSessionID = readLinkedAdmissionSessionID(requestToken)
  if (!rememberedSessionID) {
    return false
  }

  const authenticated = await refreshAdmissionAuthState()
  if (!isCurrentAdmissionRoute(requestToken)) return true
  if (!authenticated) {
    consumedTokenNeedsLogin.value = true
    pageState.value = 'needsLogin'
    return true
  }

  try {
    const nextAdmission = await admissionApi.getAdmissionMe(rememberedSessionID)
    if (!isCurrentAdmissionRoute(requestToken)) return true
    handleAdmissionMeState(nextAdmission)
    return true
  } catch (error) {
    forgetLinkedAdmissionSession(requestToken)
    if (isAdmissionSessionExpiredError(error)) {
      handleAdmissionExpired()
      return true
    }
    return false
  }
}

async function refreshAdmissionAuthState(): Promise<boolean> {
  const shouldProbeAuth = (
    auth.isAuthenticated ||
    hasStoredSessionHint() ||
    consumeAdmissionAuthReturn(currentAdmissionURL())
  )
  if (!shouldProbeAuth) {
    return false
  }
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

function retryProjectionRefresh(): void {
  if (pageState.value !== 'projectionPending') return
  scheduleProjectionRefresh()
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
    const nextAdmission = await admissionApi.getAdmissionMe(session.value?.id)
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

function isCurrentAdmissionRoute(requestToken: string): boolean {
  return currentAdmissionRouteKey() === admissionRouteKey(requestToken)
}

function admissionRouteKey(routeToken: string): string {
  return routeToken
}

function currentAdmissionRouteKey(): string {
  return admissionRouteKey(readAdmissionToken())
}

async function loadAdmissionSession() {
  const requestToken = readAdmissionToken()
  activeAdmissionRouteKey = admissionRouteKey(requestToken)
  pageState.value = 'loading'
  consumedTokenNeedsLogin.value = false
  try {
    if (await resumeRememberedAdmissionSession(requestToken)) {
      return
    }

    const preview = await admissionApi.getAdmissionSession(requestToken)
    if (!isCurrentAdmissionRoute(requestToken)) return
    session.value = preview
    const authenticated = await refreshAdmissionAuthState()
    if (!isCurrentAdmissionRoute(requestToken)) return
    if (!authenticated) {
      pageState.value = 'needsLogin'
      return
    }
    if (await applyKnownQQMismatch(preview)) return
    handleSessionState(preview)
  } catch (error) {
    if (!isCurrentAdmissionRoute(requestToken)) return
    if (isAdmissionTokenConsumedError(error)) {
      const authenticated = await refreshAdmissionAuthState()
      if (!isCurrentAdmissionRoute(requestToken)) return
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
    )
    bindConfirmationDialogOpen.value = false
    session.value = linked
    rememberLinkedAdmissionSession(readAdmissionToken(), linked.id)
    handleSessionState(linked)
  } catch (error) {
    bindConfirmationDialogOpen.value = false
    if (isAdmissionTokenConsumedError(error)) {
      pageState.value = 'accountMismatch'
      return
    }
    applyError(error)
  } finally {
    linking.value = false
  }
}

function openBindConfirmationDialog(): void {
  bindConfirmationQQ.value = ''
  bindConfirmationTouched.value = false
  bindConfirmationDialogOpen.value = true
}

function closeBindConfirmationDialog(): void {
  if (linking.value) return
  bindConfirmationDialogOpen.value = false
}

function handleBindConfirmationOpenChange(open: boolean): void {
  if (!open && linking.value) return
  bindConfirmationDialogOpen.value = open
  if (open) {
    bindConfirmationQQ.value = ''
    bindConfirmationTouched.value = false
  }
}

async function submitBindConfirmation(): Promise<void> {
  bindConfirmationTouched.value = true
  if (!bindConfirmationMatches.value || linking.value) return
  await confirmLink()
}

function startLogin() {
  void auth.login(currentAdmissionURL())
}

function startSignup() {
  void auth.signup(currentAdmissionURL())
}

function startAccountSwitch() {
  void auth.switchAccount(currentAdmissionURL())
}

async function copyReissueCommand(): Promise<void> {
  try {
    if (!navigator.clipboard?.writeText) {
      throw new Error('clipboard unavailable')
    }
    await navigator.clipboard.writeText(reissueCommand.value)
    toast.success('重新生成指令已复制')
  } catch {
    toast.error('复制失败，请手动复制')
  }
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
    bindConfirmationDialogOpen.value = false
    bindConfirmationQQ.value = ''
    bindConfirmationTouched.value = false
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
