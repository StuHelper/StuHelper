<template>
  <div data-admission-page-root>
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
        <button
          class="primary-button"
          type="button"
          :disabled="authActionBusy"
          @click="startLogin"
        >
          {{ authActionBusy ? '正在跳转…' : '登录' }}
        </button>
        <button
          v-if="!consumedTokenNeedsLogin"
          class="secondary-button"
          type="button"
          :disabled="authActionBusy"
          @click="startSignup"
        >
          {{ authActionBusy ? '正在跳转…' : '注册' }}
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
        <button
          class="primary-button"
          type="button"
          :disabled="authActionBusy"
          @click="startAccountSwitch"
        >
          {{ authActionBusy ? '正在退出…' : '重新登录' }}
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
        <button
          class="primary-button"
          type="button"
          :disabled="authActionBusy"
          @click="startAccountSwitch"
        >
          {{ authActionBusy ? '正在退出…' : '重新登录' }}
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
        <span class="admission-flow__icon" aria-hidden="true">
          <ShieldCheck class="admission-flow__icon-svg" />
        </span>
        <div>
          <h2>完成账号级学生认证</h2>
          <p>
            当前 QQ 会话已经绑定。学生认证是独立的账号能力；认证中心通过后，本页会自动读取最新资格并继续入群流程。
          </p>
        </div>
      </div>
      <div
        v-if="linkedResourceErrorMessage"
        class="linked-resource-error"
        data-linked-resource-error
        role="status"
      >
        <p>{{ linkedResourceErrorMessage }}</p>
      </div>
      <div class="admission-flow__notice">
        <p>
          认证中心会根据学校配置展示可用方式，包括实名信息校验、学校统一身份认证、学号邮箱或人工材料审核。入群系统不会读取或复制认证材料。
        </p>
      </div>
      <div class="admission-flow__actions">
        <a
          class="primary-button"
          data-admission-open-student-verification
          :href="studentVerificationURL"
        >
          前往学生认证中心
        </a>
        <button
          class="secondary-button"
          data-admission-refresh-eligibility
          type="button"
          :disabled="admissionStateRefreshInFlight"
          @click="refreshAdmissionStateNow"
        >
          {{ admissionStateRefreshInFlight ? '正在检查…' : '我已完成，检查资格' }}
        </button>
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

    <Dialog :open="bindConfirmationDialogOpen" @update:open="handleBindConfirmationOpenChange">
      <DialogContent
        class="sm:max-w-[520px]"
        data-admission-bind-confirmation-dialog
      >
        <DialogHeader>
          <div class="mb-1 flex h-10 w-10 items-center justify-center rounded-md bg-amber-100 text-amber-700">
            <ShieldAlert class="h-5 w-5" aria-hidden="true" />
          </div>
          <DialogTitle>确认绑定 QQ</DialogTitle>
          <DialogDescription>
            您正在将 StuHelper 账号
            <span class="font-semibold text-slate-950">[{{ currentUserLabel }}]</span>
            绑定至 QQ：
            <span class="font-semibold text-slate-950">[{{ displayQQ || '当前入群 QQ' }}]</span>。
            绑定后无法变更。请确认是否继续？
          </DialogDescription>
        </DialogHeader>

        <div class="space-y-3">
          <div class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm leading-relaxed text-amber-900">
            绑定后该 QQ 将用于入群验证和机器人识别。若这不是你正在入群使用的 QQ，请取消并重新登录正确账号。
          </div>
          <div class="grid gap-2">
            <label
              class="text-sm font-medium text-slate-800"
              for="admission-bind-confirmation-qq"
            >
              手动输入需要绑定的 QQ 号
            </label>
            <Input
              id="admission-bind-confirmation-qq"
              v-model="bindConfirmationQQ"
              autocomplete="off"
              data-admission-bind-confirmation-input
              :disabled="linking"
              inputmode="numeric"
              :placeholder="displayQQ || 'QQ号'"
              @blur="bindConfirmationTouched = true"
              @keydown.enter.prevent="submitBindConfirmation"
            />
            <p
              v-if="bindConfirmationError"
              class="text-sm text-red-600"
              data-admission-bind-confirmation-error
            >
              {{ bindConfirmationError }}
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button
            :disabled="linking"
            type="button"
            variant="outline"
            @click="closeBindConfirmationDialog"
          >
            取消
          </Button>
          <Button
            data-admission-bind-confirmation-submit
            :disabled="!bindConfirmationMatches || linking"
            type="button"
            @click="submitBindConfirmation"
          >
            <ShieldCheck class="h-4 w-4" aria-hidden="true" />
            {{ linking ? '正在确认...' : '确认并开始认证' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
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
  ShieldAlert,
  ShieldCheck,
  UserX,
  XCircle,
} from 'lucide-vue-next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { getErrorStatus } from '@/api/errors'
import { Input } from '@/components/ui/input'
import { useToast } from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { consumeAdmissionAuthReturn } from '@/utils/auth'
import { hasStoredSessionHint } from '@/utils/sessionHint'
import { accountCenterURLWithRedirect } from '@/utils/redirect'
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
import { waitForAdmissionProjection } from '../projectionRefresh'
import AdmissionReissueHint from './AdmissionReissueHint.vue'
import AdmissionProgress from './AdmissionProgress.vue'
import AdmissionShell from './AdmissionShell.vue'
import AdmissionStatePanel from './AdmissionStatePanel.vue'
import ProjectionPendingNotice from './ProjectionPendingNotice.vue'

const ADMISSION_STATE_REFRESH_DELAYS_MS = [
  5000,
  10000,
  20000,
  30000,
] as const

const route = useRoute()
const auth = useAuthStore()
const verificationStore = useVerificationStore()
const toast = useToast()
const authRedirecting = ref(false)
const authActionBusy = computed(() => authRedirecting.value || auth.loading)
const pageState = ref<AdmissionPageState>('loading')
const session = ref<AdmissionSession | null>(null)
const errorMessage = ref('认证链接暂时无法打开，请稍后重试。')
const linkedResourceErrorMessage = ref('')
const consumedTokenNeedsLogin = ref(false)
const linking = ref(false)

const projectionRefreshTimedOut = ref(false)
const bindConfirmationDialogOpen = ref(false)
const bindConfirmationQQ = ref('')
const bindConfirmationTouched = ref(false)
let projectionRefreshAbort: AbortController | null = null
let admissionSessionLoad: Promise<void> | null = null
let admissionSessionReloadQueued = false
let activeAdmissionRouteKey = ''
let admissionStateRefreshTimer: number | null = null
let admissionStateRefreshAttempt = 0
const admissionStateRefreshInFlight = ref(false)

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
      return '等待学生资格'
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

const studentVerificationURL = computed(() => accountCenterURLWithRedirect(
  '/user/student-verification',
  currentAdmissionURL(),
))

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
  if (nextAdmission.session?.id) {
    rememberLinkedAdmissionSession(readAdmissionToken(), nextAdmission.session.id)
  }
}

async function loadLinkedResources(): Promise<void> {
  linkedResourceErrorMessage.value = ''
  const nextAdmission = await admissionApi.getAdmissionMe(session.value?.id)
  setAdmissionMe(nextAdmission)
  if (nextAdmission.session) {
    session.value = nextAdmission.session
  }
  pageState.value = stateFromAdmissionMe(nextAdmission)
  if (pageState.value === 'projectionPending') scheduleProjectionRefresh()
}

function scheduleLinkedResourcesLoad(): void {
  loadLinkedResources().catch(handleLinkedResourcesLoadError)
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
  const status = String(nextSession.status)
  if ((status !== 'awaiting_account_link' && status !== 'joined_muted') || !nextSession.qqID) {
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
      if (isAbortError(error)) return
      if (getErrorStatus(error) === 401) {
        pageState.value = 'needsLogin'
        return
      }
      projectionRefreshTimedOut.value = true
    })
}

function retryProjectionRefresh(): void {
  if (pageState.value !== 'projectionPending') return
  scheduleProjectionRefresh()
}

function isAwaitingEligibilityState(): boolean {
  return pageState.value === 'linked' || pageState.value === 'pendingReview'
}

function scheduleAdmissionStateRefresh(reset = false): void {
  if (!isAwaitingEligibilityState()) return
  if (reset) admissionStateRefreshAttempt = 0
  if (admissionStateRefreshTimer !== null || admissionStateRefreshInFlight.value) return
  if (document.visibilityState !== 'visible') return
  const delay = admissionStateRefreshDelay()
  admissionStateRefreshAttempt += 1
  admissionStateRefreshTimer = window.setTimeout(() => {
    admissionStateRefreshTimer = null
    void refreshAdmissionState()
  }, delay)
}

function admissionStateRefreshDelay(): number {
  const index = Math.min(
    admissionStateRefreshAttempt,
    ADMISSION_STATE_REFRESH_DELAYS_MS.length - 1,
  )
  return ADMISSION_STATE_REFRESH_DELAYS_MS[index]
}

function clearAdmissionStateRefresh(): void {
  if (admissionStateRefreshTimer === null) return
  window.clearTimeout(admissionStateRefreshTimer)
  admissionStateRefreshTimer = null
}

function refreshAdmissionStateAfterBrowserReturn(): void {
  clearAdmissionStateRefresh()
  void refreshAdmissionState()
}

async function refreshAdmissionState(): Promise<void> {
  if (!isAwaitingEligibilityState() || admissionStateRefreshInFlight.value) return
  admissionStateRefreshInFlight.value = true
  linkedResourceErrorMessage.value = ''
  try {
    const nextAdmission = await admissionApi.getAdmissionMe(session.value?.id)
    if (isAwaitingEligibilityState()) {
      handleAdmissionMeState(nextAdmission)
    }
  } catch (error) {
    if (isAdmissionSessionExpiredError(error)) {
      handleAdmissionExpired()
    } else if (pageState.value === 'linked') {
      linkedResourceErrorMessage.value = '暂时无法检查最新学生资格；系统会继续自动重试。'
    }
  } finally {
    admissionStateRefreshInFlight.value = false
    if (isAwaitingEligibilityState()) {
      scheduleAdmissionStateRefresh()
    }
  }
}

function refreshAdmissionStateNow(): void {
  clearAdmissionStateRefresh()
  void refreshAdmissionState()
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
  if (isAwaitingEligibilityState()) {
    refreshAdmissionStateAfterBrowserReturn()
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
    clearAdmissionStateRefresh()
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

async function runAuthRedirect(
  start: (returnURL: string) => Promise<void>,
): Promise<void> {
  if (authActionBusy.value) return
  authRedirecting.value = true
  try {
    await start(currentAdmissionURL())
    // 成功后保持锁定；浏览器即将离开当前页面。
  } catch {
    authRedirecting.value = false
  }
}

function startLogin() {
  void runAuthRedirect(auth.login)
}

function startSignup() {
  void runAuthRedirect(auth.signup)
}

function startAccountSwitch() {
  void runAuthRedirect(auth.switchAccount)
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

function handleAdmissionExpired() {
  pageState.value = 'expired'
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
    bindConfirmationDialogOpen.value = false
    bindConfirmationQQ.value = ''
    bindConfirmationTouched.value = false
    queueAdmissionSessionLoad()
  },
)

watch(pageState, (state) => {
  if (state === 'linked' || state === 'pendingReview') {
    scheduleAdmissionStateRefresh(true)
    return
  }
  clearAdmissionStateRefresh()
})

onBeforeUnmount(() => {
  projectionRefreshAbort?.abort()
  clearAdmissionStateRefresh()
  window.removeEventListener('pageshow', handlePageShow)
  window.removeEventListener('focus', handleWindowFocus)
  document.removeEventListener('visibilitychange', handleVisibilityChange)
})

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError'
}
</script>

<style scoped src="./AdmissionPage.css"></style>
