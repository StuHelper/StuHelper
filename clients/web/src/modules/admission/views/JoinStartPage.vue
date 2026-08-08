<template>
  <main class="min-h-dvh bg-slate-50 px-4 py-6 text-slate-950 sm:py-8">
    <section class="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <header class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <p class="m-0 text-sm font-medium text-slate-500">StuHelper</p>
        <h1 class="m-0 mt-2 text-2xl font-semibold">学生认证与 QQ 绑定</h1>
        <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
          此入口用于提前完成账号级学生认证和 QQ 绑定。没有入群验证码时，可以先在这里准备账号；它不会替代群内认证链接，也不会解除任何群聊禁言。
        </p>
      </header>

      <section
        class="rounded-lg border border-slate-200 bg-white p-5 shadow-sm"
        data-join-start
      >
        <div v-if="pageState === 'loading'" class="flex items-center gap-3 py-3" data-state="loading">
          <span
            class="size-5 shrink-0 rounded-full border-2 border-slate-200 border-t-slate-900 animate-spin"
            aria-hidden="true"
          />
          <p class="m-0 text-sm text-slate-600">正在检查账号状态...</p>
        </div>

        <div v-else-if="pageState === 'needsLogin'" data-state="needsLogin">
          <div class="flex items-start gap-3">
            <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-slate-100 text-slate-700">
              <LogIn class="size-5" aria-hidden="true" />
            </span>
            <div>
              <h2 class="m-0 text-lg font-semibold text-slate-950">登录 StuHelper</h2>
              <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
                登录或注册后会回到当前页面，继续完成学生认证和 QQ 绑定。
              </p>
            </div>
          </div>
          <div class="mt-5 grid gap-3 sm:grid-cols-2">
            <button
              class="join-start-primary-button"
              type="button"
              :disabled="auth.loading"
              @click="startLogin"
            >
              登录
            </button>
            <button
              class="join-start-secondary-button"
              type="button"
              :disabled="auth.loading"
              @click="startSignup"
            >
              注册账号
            </button>
          </div>
        </div>

        <div v-else-if="pageState === 'loadFailed'" data-state="loadFailed">
          <div class="flex items-start gap-3">
            <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-amber-50 text-amber-700">
              <RefreshCw class="size-5" aria-hidden="true" />
            </span>
            <div>
              <h2 class="m-0 text-lg font-semibold text-slate-950">状态加载失败</h2>
              <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
                当前无法读取账号认证状态，请检查网络后重试。
              </p>
            </div>
          </div>
          <button
            class="join-start-primary-button mt-5"
            type="button"
            @click="loadAccountReadiness"
          >
            重新加载
          </button>
        </div>

        <template v-else>
          <nav
            class="mb-5 grid gap-2 sm:grid-cols-3"
            aria-label="账号准备进度"
          >
            <div
              v-for="step in readinessSteps"
              :key="step.key"
              class="rounded-lg border p-3 text-sm"
              :class="step.active ? 'border-slate-900 bg-slate-50 text-slate-950' : 'border-slate-200 bg-white text-slate-500'"
            >
              <div class="flex items-center gap-2">
                <component :is="step.icon" class="size-4 shrink-0" aria-hidden="true" />
                <span class="font-medium">{{ step.label }}</span>
              </div>
              <p class="m-0 mt-1 text-xs leading-5" :class="step.done ? 'text-emerald-700' : 'text-slate-500'">
                {{ step.done ? '已完成' : step.hint }}
              </p>
            </div>
          </nav>

          <div v-if="pageState === 'needsStudentVerification'" data-state="needsStudentVerification">
            <div class="mb-4">
              <h2 class="m-0 text-lg font-semibold text-slate-950">完成学生认证</h2>
              <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
                学生认证中心是独立模块，不绑定任何 QQ 或入群会话。完成后返回此页即可继续。
              </p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <a
                class="join-start-primary-button"
                data-open-student-verification
                :href="studentVerificationURL"
              >
                前往学生认证中心
              </a>
              <button
                class="join-start-secondary-button"
                type="button"
                @click="loadAccountReadiness"
              >
                我已完成，刷新状态
              </button>
            </div>
          </div>

          <div v-else-if="pageState === 'needsQQBinding'" data-state="needsQQBinding">
            <div class="mb-4">
              <h2 class="m-0 text-lg font-semibold text-slate-950">绑定 QQ</h2>
              <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
                QQ 绑定同样是独立的账号能力。绑定后机器人可以识别你的 StuHelper 账号，但不会自动创建或放行入群申请。
              </p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
              <a
                class="join-start-primary-button"
                data-open-qq-binding
                :href="qqBindingURL"
              >
                前往 QQ 绑定
              </a>
              <button
                class="join-start-secondary-button"
                type="button"
                @click="loadAccountReadiness"
              >
                我已完成，刷新状态
              </button>
            </div>
          </div>

          <div v-else data-state="ready">
            <div class="flex items-start gap-3">
              <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-emerald-50 text-emerald-700">
                <CheckCircle2 class="size-5" aria-hidden="true" />
              </span>
              <div>
                <h2 class="m-0 text-lg font-semibold text-slate-950">账号已准备好</h2>
                <p class="m-0 mt-2 text-sm leading-6 text-slate-600">
                  当前账号已完成学生认证和 QQ 绑定。后续加入受控群时，请仍使用群内机器人或管理员提供的最新认证入口。
                </p>
              </div>
            </div>
          </div>
        </template>
      </section>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, markRaw, onBeforeUnmount, onMounted, ref } from 'vue'
import type { Component } from 'vue'
import { Bot, CheckCircle2, GraduationCap, LogIn, RefreshCw, UserRound } from 'lucide-vue-next'

import { useToast } from '@/composables/useToast'
import { updatePageMeta } from '@/composables/usePageMeta'
import { accountCenterURLWithRedirect } from '@/utils/redirect'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'

type PageState =
  | 'loading'
  | 'needsLogin'
  | 'needsStudentVerification'
  | 'needsQQBinding'
  | 'ready'
  | 'loadFailed'

interface ReadinessStep {
  key: string
  label: string
  hint: string
  done: boolean
  active: boolean
  icon: Component
}

const auth = useAuthStore()
const verificationStore = useVerificationStore()
const toast = useToast()
const pageState = ref<PageState>('loading')

const currentReturnURL = computed(() => {
  if (typeof window === 'undefined') return undefined
  return window.location.href
})
const joinStartReturnURL = computed(() => {
  if (typeof window === 'undefined') return '/start'
  return new URL('/start', window.location.origin).toString()
})
const studentVerificationURL = computed(() => accountCenterURLWithRedirect(
  '/user/student-verification',
  joinStartReturnURL.value,
))
const qqBindingURL = computed(() => accountCenterURLWithRedirect(
  '/user/qq-binding',
  joinStartReturnURL.value,
))

const readinessSteps = computed<ReadinessStep[]>(() => [
  {
    key: 'account',
    label: '登录账号',
    hint: '需要登录',
    done: auth.isAuthenticated,
    active: pageState.value === 'needsLogin',
    icon: markRaw(UserRound),
  },
  {
    key: 'student',
    label: '学生认证',
    hint: '待认证',
    done: verificationStore.studentVerified,
    active: pageState.value === 'needsStudentVerification',
    icon: markRaw(GraduationCap),
  },
  {
    key: 'qq',
    label: 'QQ 绑定',
    hint: '待绑定',
    done: verificationStore.qqBound,
    active: pageState.value === 'needsQQBinding' || pageState.value === 'ready',
    icon: markRaw(Bot),
  },
])

async function bootstrapJoinStart() {
  pageState.value = 'loading'
  const authenticated = await auth.bootstrapSession({ force: true })
  if (!authenticated) {
    pageState.value = 'needsLogin'
    return
  }
  await loadAccountReadiness()
}

async function loadAccountReadiness() {
  pageState.value = 'loading'
  try {
    await verificationStore.fetchStatus()
    if (!verificationStore.studentVerified) {
      pageState.value = 'needsStudentVerification'
      return
    }
    if (!verificationStore.qqBound) {
      pageState.value = 'needsQQBinding'
      return
    }
    pageState.value = 'ready'
  } catch {
    pageState.value = 'loadFailed'
    toast.error('账号认证状态加载失败，请稍后重试')
  }
}

function startLogin() {
  void auth.login(currentReturnURL.value)
}

function startSignup() {
  void auth.signup(currentReturnURL.value)
}

function refreshAfterReturn(): void {
  if (auth.isAuthenticated && document.visibilityState === 'visible') {
    void loadAccountReadiness()
  }
}

updatePageMeta({
  title: '学生认证与 QQ 绑定',
  description: '在 join.stuhelper.com 快捷完成 StuHelper 学生认证和 QQ 绑定。',
})

onMounted(() => {
  window.addEventListener('focus', refreshAfterReturn)
  window.addEventListener('pageshow', refreshAfterReturn)
  void bootstrapJoinStart()
})

onBeforeUnmount(() => {
  window.removeEventListener('focus', refreshAfterReturn)
  window.removeEventListener('pageshow', refreshAfterReturn)
})
</script>

<style scoped>
.join-start-primary-button,
.join-start-secondary-button {
  align-items: center;
  border-radius: 8px;
  cursor: pointer;
  display: inline-flex;
  font-size: 14px;
  font-weight: 600;
  justify-content: center;
  line-height: 20px;
  min-height: 44px;
  padding: 10px 16px;
  transition:
    background-color 150ms ease,
    border-color 150ms ease,
    color 150ms ease,
    opacity 150ms ease;
}

.join-start-primary-button {
  background: #0f172a;
  border: 1px solid #0f172a;
  color: #ffffff;
  text-decoration: none;
}

.join-start-secondary-button {
  background: #ffffff;
  border: 1px solid #cbd5e1;
  color: #0f172a;
}

.join-start-primary-button:disabled,
.join-start-secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
</style>
