<template>
  <div class="projection-pending" data-state="projectionPending">
    <div class="projection-pending__heading">
      <span class="projection-pending__icon" aria-hidden="true">
        <RefreshCw class="projection-pending__icon-svg" />
      </span>
      <div>
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
      class="projection-pending__hint"
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
.projection-pending {
  border: 1px solid #dbe3ee;
  border-radius: 8px;
  display: grid;
  gap: 16px;
}

.projection-pending__heading {
  display: grid;
  gap: 12px;
  grid-template-columns: auto minmax(0, 1fr);
}

.projection-pending__icon {
  align-items: center;
  background: #eff6ff;
  border-radius: 8px;
  color: #1d4ed8;
  display: inline-flex;
  height: 44px;
  justify-content: center;
  width: 44px;
}

.projection-pending__icon-svg {
  height: 22px;
  width: 22px;
}

.projection-pending h2,
.projection-pending p {
  margin: 0;
}

.projection-pending h2 {
  color: #0f172a;
  font-size: 20px;
  font-weight: 700;
  line-height: 28px;
}

.projection-pending p,
.projection-pending__hint {
  color: #475569;
  font-size: 14px;
  line-height: 22px;
}

.projection-pending__hint {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px;
}

.projection-pending__button {
  align-items: center;
  background: #0f172a;
  border: 1px solid transparent;
  border-radius: 8px;
  color: #ffffff;
  cursor: pointer;
  display: inline-flex;
  font-size: 14px;
  font-weight: 600;
  justify-content: center;
  line-height: 20px;
  min-height: 44px;
  padding: 10px 16px;
  transition: background-color 160ms ease, box-shadow 160ms ease;
  width: fit-content;
}

.projection-pending__button:hover {
  background: #111827;
}

.projection-pending__button:focus-visible {
  outline: 3px solid rgb(15 118 110 / 0.28);
  outline-offset: 2px;
}

@media (max-width: 520px) {
  .projection-pending__heading {
    grid-template-columns: 1fr;
  }
}
</style>
