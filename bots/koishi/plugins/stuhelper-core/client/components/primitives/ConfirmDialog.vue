<template>
  <el-dialog
    class="sh-confirm"
    modal-class="stuhelperGroupCenter-portal"
    :model-value="open"
    :append-to-body="true"
    :close-on-click-modal="false"
    :width="DIALOG_WIDTH"
    @close="emit('cancel')"
  >
    <template #header>
      <div class="sh-confirm__head" :data-tone="tone">
        <span class="sh-confirm__marker" aria-hidden="true"></span>
        <h3 class="sh-confirm__title">{{ title }}</h3>
      </div>
    </template>

    <p class="sh-confirm__message">{{ message }}</p>

    <template #footer>
      <div class="sh-confirm__actions">
        <el-button class="sh-button sh-button--ghost" @click="emit('cancel')">
          {{ cancelText }}
        </el-button>
        <el-button
          :type="tone === 'danger' ? 'danger' : 'primary'"
          class="sh-button"
          :class="tone === 'danger' ? 'sh-button--danger' : 'sh-button--primary'"
          @click="emit('confirm')"
        >
          {{ confirmText }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import type { ConfirmTone } from '../../composables/use-confirm'

const DIALOG_WIDTH = 'min(400px, calc(100vw - 32px))'

withDefaults(
  defineProps<{
    open: boolean
    title: string
    message: string
    tone?: ConfirmTone
    confirmText?: string
    cancelText?: string
  }>(),
  {
    tone: 'normal',
    confirmText: '确认',
    cancelText: '取消',
  },
)

const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>

<style scoped>
:deep(.el-dialog.sh-confirm),
:deep(.sh-confirm .el-dialog) {
  border-radius: var(--sh-r-3);
  background: var(--sh-surface-0);
  box-shadow: var(--sh-shadow-overlay);
}

:deep(.el-dialog.sh-confirm .el-dialog__header),
:deep(.el-dialog.sh-confirm .el-dialog__body),
:deep(.el-dialog.sh-confirm .el-dialog__footer),
:deep(.sh-confirm .el-dialog__header),
:deep(.sh-confirm .el-dialog__body),
:deep(.sh-confirm .el-dialog__footer) {
  padding-left: var(--sh-s-5);
  padding-right: var(--sh-s-5);
}

.sh-confirm__head {
  display: flex;
  align-items: center;
  gap: var(--sh-s-3);
}

.sh-confirm__marker {
  width: 8px;
  height: 8px;
  border-radius: var(--sh-r-full);
  background: var(--sh-primary);
  flex: 0 0 auto;
}

.sh-confirm__head[data-tone='danger'] .sh-confirm__marker {
  background: var(--sh-danger);
}

.sh-confirm__title {
  margin: 0;
  font-size: var(--sh-t-title);
  font-weight: var(--sh-w-semibold);
  color: var(--sh-fg);
  line-height: var(--sh-l-tight);
}

.sh-confirm__message {
  margin: 0;
  color: var(--sh-fg-1);
  font-size: var(--sh-t-body);
  line-height: var(--sh-l-normal);
}

.sh-confirm__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--sh-s-2);
}
</style>
