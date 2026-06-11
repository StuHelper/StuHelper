<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent
      class="admission-bind-dialog sm:max-w-[520px]"
      data-admission-bind-confirmation-dialog
    >
      <DialogHeader>
        <div class="admission-bind-dialog__badge">
          <ShieldAlert class="h-5 w-5" aria-hidden="true" />
        </div>
        <DialogTitle class="admission-bind-dialog__title">确认绑定 QQ</DialogTitle>
        <DialogDescription class="admission-bind-dialog__description">
          您正在将 StuHelper 账号
          <span class="admission-bind-dialog__strong">[{{ currentUserLabel }}]</span>
          绑定至 QQ：
          <span class="admission-bind-dialog__strong">[{{ displayQq || '当前入群 QQ' }}]</span>。
          绑定后无法变更。请确认是否继续？
        </DialogDescription>
      </DialogHeader>

      <div class="admission-bind-dialog__body">
        <div class="admission-bind-dialog__warning">
          绑定后该 QQ 将用于入群验证和机器人识别。若这不是你正在入群使用的 QQ，请取消并重新登录正确账号。
        </div>
        <div class="admission-bind-dialog__field">
          <label
            class="admission-bind-dialog__label"
            for="admission-bind-confirmation-qq"
          >
            手动输入需要绑定的 QQ 号
          </label>
          <Input
            id="admission-bind-confirmation-qq"
            autocomplete="off"
            class="admission-bind-dialog__input"
            data-admission-bind-confirmation-input
            :disabled="linking"
            inputmode="numeric"
            :model-value="qq"
            :placeholder="displayQq || 'QQ号'"
            @blur="emit('touch')"
            @keydown.enter.prevent="emit('submit')"
            @update:model-value="emit('update:qq', $event)"
          />
          <p
            v-if="errorMessage"
            class="admission-bind-dialog__error"
            data-admission-bind-confirmation-error
          >
            {{ errorMessage }}
          </p>
        </div>
      </div>

      <DialogFooter>
        <Button
          class="admission-bind-dialog__cancel"
          :disabled="linking"
          type="button"
          variant="outline"
          @click="emit('cancel')"
        >
          取消
        </Button>
        <Button
          class="admission-bind-dialog__submit"
          data-admission-bind-confirmation-submit
          :disabled="!matches || linking"
          type="button"
          @click="emit('submit')"
        >
          <ShieldCheck class="h-4 w-4" aria-hidden="true" />
          {{ linking ? '正在确认...' : '确认并开始认证' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { ShieldAlert, ShieldCheck } from 'lucide-vue-next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'

defineProps<{
  currentUserLabel: string
  displayQq: string
  errorMessage: string
  linking: boolean
  matches: boolean
  open: boolean
  qq: string
}>()

const emit = defineEmits<{
  cancel: []
  submit: []
  touch: []
  'update:open': [open: boolean]
  'update:qq': [qq: string]
}>()
</script>

<!--
  非 scoped：DialogContent 会 teleport 到 body，脱离 .join-surface 作用域，
  既拿不到 --join-* 变量，也接不到 scoped data 属性。这里以
  .admission-bind-dialog 为命名空间自带一套玻璃风变量（含暗色覆盖），
  只使用全局 --color-* / --radius-* design tokens。
-->
<style>
.admission-bind-dialog {
  --bind-glass-bg: rgba(255, 255, 255, 0.9);
  --bind-glass-border: rgba(255, 255, 255, 0.68);
  --bind-glass-highlight: rgba(255, 255, 255, 0.9);
  --bind-chip-bg: rgba(255, 255, 255, 0.55);
  --bind-warning-bg: rgba(232, 168, 64, 0.16);
  --bind-warning-fg: #9a6a14;
  --bind-danger-fg: #b54040;
  --bind-focus-ring: rgba(91, 124, 247, 0.45);
}

[data-theme="dark"] .admission-bind-dialog {
  --bind-glass-bg: rgba(38, 31, 60, 0.92);
  --bind-glass-border: rgba(255, 255, 255, 0.14);
  --bind-glass-highlight: rgba(255, 255, 255, 0.16);
  --bind-chip-bg: rgba(255, 255, 255, 0.07);
  --bind-warning-bg: rgba(232, 168, 64, 0.14);
  --bind-warning-fg: #f0c060;
  --bind-danger-fg: #e88888;
}

@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) .admission-bind-dialog {
    --bind-glass-bg: rgba(38, 31, 60, 0.92);
    --bind-glass-border: rgba(255, 255, 255, 0.14);
    --bind-glass-highlight: rgba(255, 255, 255, 0.16);
    --bind-chip-bg: rgba(255, 255, 255, 0.07);
    --bind-warning-bg: rgba(232, 168, 64, 0.14);
    --bind-warning-fg: #f0c060;
    --bind-danger-fg: #e88888;
  }
}

/* 玻璃容器：双类名提升优先级，覆盖 DialogContent 内置的浅色工具类 */
.admission-bind-dialog.admission-bind-dialog {
  -webkit-backdrop-filter: blur(20px) saturate(170%);
  backdrop-filter: blur(20px) saturate(170%);
  background: var(--bind-glass-bg);
  border: 1px solid var(--bind-glass-border);
  border-radius: var(--radius-2xl);
  box-shadow:
    inset 0 1px 0 var(--bind-glass-highlight),
    0 24px 64px rgba(50, 40, 70, 0.3);
  color: var(--color-text-primary);
  padding: 26px;
}

@supports not (backdrop-filter: blur(1px)) {
  .admission-bind-dialog.admission-bind-dialog {
    background: var(--color-bg-card);
  }
}

.admission-bind-dialog__badge {
  align-items: center;
  background: var(--bind-warning-bg);
  border-radius: 12px;
  box-shadow: inset 0 1px 0 var(--bind-glass-highlight);
  color: var(--bind-warning-fg);
  display: inline-flex;
  height: 42px;
  justify-content: center;
  margin-bottom: 6px;
  width: 42px;
}

.admission-bind-dialog__title.admission-bind-dialog__title {
  color: var(--color-text-primary);
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 28px;
}

.admission-bind-dialog__description.admission-bind-dialog__description {
  color: var(--color-text-secondary);
}

.admission-bind-dialog__strong {
  color: var(--color-text-primary);
  font-weight: 700;
}

.admission-bind-dialog__body {
  display: grid;
  gap: 14px;
}

.admission-bind-dialog__warning {
  background: var(--bind-warning-bg);
  border: 1px solid var(--bind-glass-border);
  border-radius: var(--radius-lg);
  color: var(--bind-warning-fg);
  font-size: 13px;
  line-height: 21px;
  padding: 12px 14px;
}

.admission-bind-dialog__field {
  display: grid;
  gap: 8px;
}

.admission-bind-dialog__label {
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
}

.admission-bind-dialog__input.admission-bind-dialog__input {
  background: var(--bind-chip-bg);
  border: 1px solid var(--bind-glass-border);
  border-radius: 12px;
  color: var(--color-text-primary);
  font-size: 15px;
  height: auto;
  line-height: 22px;
  min-height: 44px;
  padding: 10px 14px;
}

.admission-bind-dialog__input.admission-bind-dialog__input::placeholder {
  color: var(--color-text-muted);
}

.admission-bind-dialog__input.admission-bind-dialog__input:focus-visible {
  --tw-ring-color: transparent;
  --tw-ring-offset-width: 0px;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(91, 124, 247, 0.3);
  outline: none;
}

.admission-bind-dialog__error {
  color: var(--bind-danger-fg);
  font-size: 13px;
  line-height: 19px;
  margin: 0;
}

/* 按钮：取消为玻璃 chip，确认为品牌渐变 CTA（触控目标 ≥44px） */
.admission-bind-dialog__cancel.admission-bind-dialog__cancel,
.admission-bind-dialog__submit.admission-bind-dialog__submit {
  border-radius: 12px;
  height: auto;
  min-height: 44px;
  padding: 10px 18px;
  transition:
    box-shadow 200ms cubic-bezier(0.4, 0, 0.2, 1),
    background-color 200ms cubic-bezier(0.4, 0, 0.2, 1),
    filter 200ms cubic-bezier(0.4, 0, 0.2, 1),
    transform 120ms cubic-bezier(0.34, 1.56, 0.64, 1);
}

.admission-bind-dialog__cancel.admission-bind-dialog__cancel {
  background: var(--bind-chip-bg);
  border: 1px solid var(--bind-glass-border);
  color: var(--color-text-primary);
}

.admission-bind-dialog__cancel.admission-bind-dialog__cancel:hover:not(:disabled) {
  background: var(--bind-glass-bg);
  color: var(--color-text-primary);
}

.admission-bind-dialog__submit.admission-bind-dialog__submit {
  background: linear-gradient(135deg, #4563d8 0%, #6f58d8 50%, #b8467e 100%);
  border: 1px solid transparent;
  box-shadow: 0 8px 24px rgba(91, 124, 247, 0.35);
  color: #ffffff;
}

.admission-bind-dialog__submit.admission-bind-dialog__submit:hover:not(:disabled) {
  background: linear-gradient(135deg, #4563d8 0%, #6f58d8 50%, #b8467e 100%);
  box-shadow: 0 10px 32px rgba(111, 88, 216, 0.45);
  filter: brightness(1.06);
}

.admission-bind-dialog__cancel.admission-bind-dialog__cancel:active:not(:disabled),
.admission-bind-dialog__submit.admission-bind-dialog__submit:active:not(:disabled) {
  transform: scale(0.97);
}

.admission-bind-dialog__cancel.admission-bind-dialog__cancel:focus-visible,
.admission-bind-dialog__submit.admission-bind-dialog__submit:focus-visible {
  --tw-ring-color: transparent;
  --tw-ring-offset-width: 0px;
  outline: 3px solid var(--bind-focus-ring);
  outline-offset: 2px;
}
</style>
