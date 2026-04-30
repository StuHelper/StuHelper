<template>
  <Teleport to="body">
    <div
      class="stuhelperGroupCenter-portal sh-dock"
      role="dialog"
      aria-label="实时聊天"
      :data-open="shell.chatOpen.value ? 'true' : 'false'"
      :data-minimized="shell.chatMinimized.value ? 'true' : 'false'"
    >
      <header class="sh-dock__head">
        <span class="sh-dock__title">实时聊天</span>
        <button
          type="button"
          class="sh-dock__action"
          :title="shell.chatMinimized.value ? '展开' : '最小化'"
          @click="shell.toggleChatMinimized()"
        >
          {{ shell.chatMinimized.value ? '⤢' : '—' }}
        </button>
        <button
          type="button"
          class="sh-dock__action"
          title="关闭"
          @click="shell.closeChat()"
        >
          ✕
        </button>
      </header>
      <div class="sh-dock__body">
        <ChatView v-if="mounted" />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

import ChatView from '../ChatView.vue'
import { useAppShell } from '../../composables/use-app-shell'

const shell = useAppShell()

// Mount ChatView lazily on first open so dashboards that never use chat
// don't pay its 2k LOC cost upfront.
const mounted = ref(false)

watch(
  () => shell.chatOpen.value,
  (open) => {
    if (open && !mounted.value) {
      mounted.value = true
    }
  },
  { immediate: true },
)
</script>
