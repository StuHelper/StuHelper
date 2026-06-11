<template>
  <main class="join-start join-surface">
    <div class="join-start__frame">
      <header class="join-start__header join-glass-heavy">
        <p class="join-start__eyebrow join-eyebrow">StuHelper</p>
        <h1 class="join-start__title">学生认证与 QQ 绑定</h1>
        <p class="join-start__description">
          此入口用于提前完成账号级学生认证和 QQ 绑定。没有入群验证码时，可以先在这里准备账号；它不会替代群内认证链接，也不会解除任何群聊禁言。
        </p>
      </header>

      <section class="join-start__panel join-glass" data-join-start>
        <div v-if="pageState === 'loading'" class="join-start__loading" data-state="loading">
          <span class="join-start__spinner" aria-hidden="true" />
          <p class="join-start__loading-text">正在检查账号状态...</p>
        </div>

        <div v-else-if="pageState === 'needsLogin'" data-state="needsLogin">
          <div class="join-start__state-head">
            <span class="join-start__state-icon join-tone-info">
              <LogIn class="size-5" aria-hidden="true" />
            </span>
            <div class="join-start__state-copy">
              <h2 class="join-start__state-title">登录 StuHelper</h2>
              <p class="join-start__state-text">
                登录或注册后会回到当前页面，继续完成学生认证和 QQ 绑定。
              </p>
            </div>
          </div>
          <div class="join-start__actions">
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
          <div class="join-start__state-head">
            <span class="join-start__state-icon join-tone-warning">
              <RefreshCw class="size-5" aria-hidden="true" />
            </span>
            <div class="join-start__state-copy">
              <h2 class="join-start__state-title">状态加载失败</h2>
              <p class="join-start__state-text">
                当前无法读取账号认证状态，请检查网络后重试。
              </p>
            </div>
          </div>
          <button
            class="join-start-primary-button join-start__retry"
            type="button"
            @click="loadAccountReadiness"
          >
            重新加载
          </button>
        </div>

        <template v-else>
          <nav class="join-start__rail" aria-label="账号准备进度">
            <div
              v-for="step in readinessSteps"
              :key="step.key"
              class="join-start__step join-chip"
              :class="{
                'join-start__step--active': step.active,
                'join-start__step--done': step.done,
              }"
            >
              <div class="join-start__step-head">
                <span class="join-start__step-icon">
                  <component :is="step.icon" class="size-4" aria-hidden="true" />
                </span>
                <span class="join-start__step-label">{{ step.label }}</span>
                <CheckCircle2
                  v-if="step.done"
                  class="join-start__step-check size-4"
                  aria-hidden="true"
                />
              </div>
              <p class="join-start__step-status">{{ step.done ? '已完成' : step.hint }}</p>
            </div>
          </nav>

          <div v-if="pageState === 'needsStudentVerification'" data-state="needsStudentVerification">
            <div class="join-start__section-head">
              <h2 class="join-start__state-title">完成学生认证</h2>
              <p class="join-start__state-text">
                学生认证通过后，继续绑定 QQ。此处提交的是账号级学生认证，不绑定任何入群会话。
              </p>
            </div>
            <div class="join-start__embed join-chip">
              <StudentVerificationPanel
                :standalone="false"
                :load-on-mount="false"
                :redirect-after-verification="false"
                @verified="loadAccountReadiness"
              />
            </div>
          </div>

          <div v-else-if="pageState === 'needsQQBinding'" data-state="needsQQBinding">
            <div class="join-start__section-head">
              <h2 class="join-start__state-title">绑定 QQ</h2>
              <p class="join-start__state-text">
                绑定后机器人可以识别你的 StuHelper 账号。若之后加入受控群，仍以群内最新认证链接或管理员策略为准。
              </p>
            </div>
            <div class="join-start__embed join-chip">
              <QQBindingPanel
                :standalone="false"
                :load-on-mount="false"
                @bound="loadAccountReadiness"
              />
            </div>
          </div>

          <div v-else data-state="ready">
            <div class="join-start__state-head">
              <span class="join-start__state-icon join-tone-success">
                <CheckCircle2 class="size-5" aria-hidden="true" />
              </span>
              <div class="join-start__state-copy">
                <h2 class="join-start__state-title">账号已准备好</h2>
                <p class="join-start__state-text">
                  当前账号已完成学生认证和 QQ 绑定。后续加入受控群时，请仍使用群内机器人或管理员提供的最新认证入口。
                </p>
              </div>
            </div>
          </div>
        </template>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, markRaw, onMounted, ref } from 'vue'
import type { Component } from 'vue'
import { Bot, CheckCircle2, GraduationCap, LogIn, RefreshCw, UserRound } from 'lucide-vue-next'

import { useToast } from '@/composables/useToast'
import { updatePageMeta } from '@/composables/usePageMeta'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import StudentVerificationPanel from '@/modules/user/views/StudentVerificationPanel.vue'
import QQBindingPanel from '@/modules/user/views/QQBindingPanel.vue'

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
      await verificationStore.fetchSchools()
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

updatePageMeta({
  title: '学生认证与 QQ 绑定',
  description: '在 join.stuhelper.com 快捷完成 StuHelper 学生认证和 QQ 绑定。',
})

onMounted(() => {
  void bootstrapJoinStart()
})
</script>

<style src="./join-theme.css"></style>

<style scoped>
/* ── 页面框架（移动优先）─────────────────────────────── */
.join-start {
  min-height: 100dvh;
  padding: 18px 12px max(28px, env(safe-area-inset-bottom));
}

.join-start__frame {
  display: grid;
  gap: 14px;
  margin: 0 auto;
  max-width: 720px;
  width: 100%;
}

/* ── 玻璃头部卡 ───────────────────────────────────── */
.join-start__header {
  display: grid;
  gap: 8px;
  padding: 20px;
}

.join-start__eyebrow {
  margin: 0;
}

.join-start__title {
  color: var(--join-ink);
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 32px;
  margin: 0;
}

.join-start__description {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
  margin: 0;
  max-width: 62ch;
}

/* ── 主玻璃面板 ───────────────────────────────────── */
.join-start__panel {
  padding: 20px;
}

/* ── 加载态 ───────────────────────────────────────── */
.join-start__loading {
  align-items: center;
  display: flex;
  gap: 12px;
  padding: 10px 0;
}

.join-start__spinner {
  animation: join-start-spin 0.9s linear infinite;
  border: 3px solid var(--join-glass-border);
  border-radius: 999px;
  border-top-color: var(--color-primary);
  flex-shrink: 0;
  height: 22px;
  width: 22px;
}

@keyframes join-start-spin {
  to {
    transform: rotate(360deg);
  }
}

.join-start__loading-text {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
  margin: 0;
}

/* ── 状态卡头部（图标气泡 + 标题/说明）──────────────── */
.join-start__state-head {
  align-items: flex-start;
  display: flex;
  gap: 14px;
}

.join-start__state-icon {
  border-radius: var(--join-radius-control);
  display: grid;
  flex-shrink: 0;
  height: 44px;
  place-items: center;
  width: 44px;
}

.join-start__state-copy {
  min-width: 0;
}

.join-start__state-title {
  color: var(--join-ink);
  font-size: 19px;
  font-weight: 700;
  line-height: 26px;
  margin: 0;
}

.join-start__state-text {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
  margin: 6px 0 0;
}

.join-start__actions {
  display: grid;
  gap: 12px;
  margin-top: 20px;
}

.join-start__retry {
  margin-top: 20px;
}

/* ── 三步准备进度（玻璃 chips + 渐变激活描边）────────── */
.join-start__rail {
  display: grid;
  gap: 10px;
  margin-bottom: 20px;
}

.join-start__step {
  display: grid;
  gap: 6px;
  padding: 12px 14px;
  transition:
    background-color var(--duration-base) var(--ease-smooth),
    border-color var(--duration-base) var(--ease-smooth),
    box-shadow var(--duration-base) var(--ease-smooth);
}

.join-start__step--active {
  background:
    linear-gradient(var(--join-glass-bg-heavy), var(--join-glass-bg-heavy)) padding-box,
    var(--join-gradient-cta) border-box;
  border-color: transparent;
  box-shadow: var(--join-cta-glow);
}

.join-start__step-head {
  align-items: center;
  display: flex;
  gap: 8px;
}

.join-start__step-icon {
  background: var(--join-chip-bg);
  border-radius: 8px;
  color: var(--join-ink-muted);
  display: grid;
  flex-shrink: 0;
  height: 28px;
  place-items: center;
  width: 28px;
}

.join-start__step--active .join-start__step-icon {
  background: var(--join-tone-info-bg);
  color: var(--join-tone-info-fg);
}

.join-start__step--done .join-start__step-icon {
  background: var(--join-tone-success-bg);
  color: var(--join-tone-success-fg);
}

.join-start__step-label {
  color: var(--join-ink-soft);
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
}

.join-start__step--active .join-start__step-label,
.join-start__step--done .join-start__step-label {
  color: var(--join-ink);
}

.join-start__step-check {
  color: var(--join-tone-success-fg);
  margin-left: auto;
}

.join-start__step-status {
  color: var(--join-ink-muted);
  font-size: 12px;
  line-height: 18px;
  margin: 0;
}

.join-start__step--done .join-start__step-status {
  color: var(--join-tone-success-fg);
}

/* ── 内嵌共享面板容器 ─────────────────────────────── */
.join-start__section-head {
  margin-bottom: 16px;
}

.join-start__embed {
  padding: 14px;
}

/* ── ≥640px：放宽留白、并排布局 ───────────────────── */
@media (min-width: 640px) {
  .join-start {
    padding: 28px 16px max(36px, env(safe-area-inset-bottom));
  }

  .join-start__frame {
    gap: 18px;
  }

  .join-start__header,
  .join-start__panel {
    padding: 26px;
  }

  .join-start__title {
    font-size: 28px;
    line-height: 36px;
  }

  .join-start__rail {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .join-start__actions {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .join-start__embed {
    padding: 18px;
  }
}
</style>
