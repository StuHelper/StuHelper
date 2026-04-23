<template>
  <template v-for="node in nodes" :key="node.key">
    <span v-if="node.kind === 'text'">{{ node.text }}</span>
    <span v-else-if="node.kind === 'at'" class="msg-at">{{ node.text }}</span>
    <span v-else-if="node.kind === 'face'" class="msg-face">{{ node.text }}</span>
    <ChatMessageImage
      v-else-if="node.kind === 'image'"
      :src="node.src"
      :file="node.file"
      :open-url="node.openUrl"
    />
    <div v-else class="msg-quote">
      <span class="quote-user">{{ node.user ? `@${node.user}` : '' }}</span>
      <span class="quote-content">{{ node.content || '[引用消息]' }}</span>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed } from 'vue'

import type { GuildMember } from '../../api'
import type { ChatMessage } from '../../types'
import ChatMessageImage from './ChatMessageImage.vue'
import { buildMessageNodes } from './message-content-model'

const props = defineProps<{
  message: ChatMessage
  members: readonly GuildMember[]
  sessionMessages: readonly ChatMessage[]
}>()

const nodes = computed(() =>
  buildMessageNodes(props.message, props.sessionMessages, props.members),
)
</script>
