<template>
  <div class="projection-pending" data-state="projectionPending">
    <div class="projection-pending__heading">
      <span
        class="projection-pending__icon"
        :class="timedOut ? 'join-tone-warning' : 'join-tone-info'"
        aria-hidden="true"
      >
        <RefreshCw
          class="projection-pending__icon-svg"
          :class="{ 'projection-pending__icon-svg--spinning': !timedOut }"
        />
      </span>
      <div class="projection-pending__copy">
        <h2>身份生效中</h2>
        <p
          v-if="timedOut"
          data-projection-timeout
        >
          身份仍在同步，请稍后手动刷新页面。
        </p>
        <p v-else>
          认证已通过，正在刷新身份能力。
        </p>
      </div>
    </div>
    <p
      v-if="timedOut"
      class="projection-pending__hint join-chip"
    >
      如果长时间未自动恢复，请回到 QQ 群确认机器人是否已经解除禁言。
    </p>
    <button
      v-if="timedOut"
      type="button"
      class="projection-pending__button"
      data-projection-retry
      @click="$emit('retry')"
    >
      重新检查状态
    </button>
  </div>
</template>

<script setup lang="ts">
import { RefreshCw } from 'lucide-vue-next'

defineProps<{
  timedOut: boolean
}>()

defineEmits<{
  retry: []
}>()
</script>

<style scoped>
/*
 * 身份同步通知：色调气泡里的 RefreshCw 同步中持续旋转（仅 transform，
 * 全局 prefers-reduced-motion 开关会禁用）；超时后切为警示色调并停止旋转。
 * data-state / data-projection-* 与全部文案为测试契约。
 */
.projection-pending {
  display: grid;
  gap: 18px;
}

.projection-pending__heading {
  align-items: start;
  display: grid;
  gap: 14px;
  grid-template-columns: auto minmax(0, 1fr);
}

/* 色调背景/文字色来自全局 .join-tone-info / .join-tone-warning */
.projection-pending__icon {
  align-items: center;
  border-radius: 14px;
  box-shadow: inset 0 1px 0 var(--join-glass-highlight);
  display: inline-flex;
  height: 46px;
  justify-content: center;
  width: 46px;
}

.projection-pending__icon-svg {
  height: 22px;
  width: 22px;
}

.projection-pending__icon-svg--spinning {
  animation: projection-pending-spin 1.8s linear infinite;
}

@keyframes projection-pending-spin {
  to {
    transform: rotate(360deg);
  }
}

.projection-pending__copy {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.projection-pending h2,
.projection-pending p {
  margin: 0;
}

.projection-pending h2 {
  color: var(--join-ink);
  font-size: 21px;
  font-weight: 800;
  letter-spacing: -0.01em;
  line-height: 28px;
}

.projection-pending p,
.projection-pending__hint {
  color: var(--join-ink-soft);
  font-size: 14px;
  line-height: 22px;
}

.projection-pending__hint {
  padding: 12px 14px;
}

/* 重试按钮：品牌渐变 CTA（触控目标 ≥44px） */
.projection-pending__button {
  align-items: center;
  background: var(--join-gradient-cta);
  border: 1px solid transparent;
  border-radius: var(--join-radius-control);
  box-shadow: var(--join-cta-glow);
  color: #ffffff;
  cursor: pointer;
  display: inline-flex;
  font-size: 14px;
  font-weight: 600;
  justify-content: center;
  line-height: 20px;
  min-height: 44px;
  padding: 10px 18px;
  transition:
    box-shadow var(--duration-base) var(--ease-smooth),
    filter var(--duration-base) var(--ease-smooth),
    transform var(--duration-fast) var(--ease-spring);
  width: fit-content;
}

.projection-pending__button:hover {
  box-shadow: var(--join-cta-glow-hover);
  filter: brightness(1.06);
}

.projection-pending__button:active {
  transform: scale(0.97);
}

.projection-pending__button:focus-visible {
  outline: 3px solid rgba(91, 124, 247, 0.45);
  outline-offset: 2px;
}

@media (max-width: 420px) {
  .projection-pending__heading {
    grid-template-columns: 1fr;
  }
}
</style>
