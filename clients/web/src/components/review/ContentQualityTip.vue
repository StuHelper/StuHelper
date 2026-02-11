<template>
  <div class="quality-tip" :class="level">
    <div class="tip-header">
      <svg v-if="level === 'error'" class="icon" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
      </svg>
      <svg v-else class="icon" viewBox="0 0 24 24" fill="currentColor">
        <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
      </svg>
      <span class="title">{{ title }}</span>
    </div>
    <ul class="tip-list">
      <li v-for="(item, index) in issues" :key="index">
        {{ item }}
      </li>
    </ul>
    <button v-if="dismissible" class="dismiss-btn" @click="$emit('dismiss')">
      {{ t('review.quality.dismiss') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  issues: string[]
  level?: 'warning' | 'error'
  dismissible?: boolean
}>(), {
  level: 'warning',
  dismissible: true
})

defineEmits<{
  dismiss: []
}>()

const title = computed(() =>
  props.level === 'error' ? t('review.quality.errorTitle') : t('review.quality.warningTitle')
)
</script>

<style scoped>
.quality-tip {
  padding: var(--space-3);
  border-radius: var(--radius-sm);
  margin-bottom: var(--space-3);
  animation: fadeIn 0.3s var(--ease-out);
}

.quality-tip.warning {
  background: transparent;
  border: 1px solid var(--border);
}

.quality-tip.error {
  background: transparent;
  border: 1px solid var(--text-primary);
}

.tip-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}

.icon {
  width: 18px;
  height: 18px;
}

.warning .icon {
  color: var(--accent);
}

.error .icon {
  color: var(--text-primary);
}

.title {
  font-weight: var(--weight-medium);
  font-size: var(--text-sm);
}

.warning .title {
  color: var(--accent);
}

.error .title {
  color: var(--text-primary);
}

.tip-list {
  margin: 0;
  padding-left: var(--space-6);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.tip-list li {
  margin-bottom: var(--space-1);
}

.dismiss-btn {
  margin-top: var(--space-2);
  padding: var(--space-1) var(--space-2);
  font-size: var(--text-xs);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  cursor: pointer;
}

.dismiss-btn:hover {
  border-color: var(--text-muted);
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
