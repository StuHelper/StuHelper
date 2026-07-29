<template>
  <img
    :src="resolvedSrc"
    class="msg-img"
    :class="{ loading: loading, error: failed, clickable: Boolean(openUrl) }"
    :alt="failed ? '图片已过期或无法加载' : '聊天图片'"
    :role="openUrl ? 'button' : undefined"
    :tabindex="openUrl ? 0 : undefined"
    :aria-label="openUrl ? '打开聊天图片' : undefined"
    @click="openImage"
    @keydown.enter.prevent="openImage"
    @keydown.space.prevent="openImage"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

import { loadChatImage, needsImageProxy } from './image-proxy'

const props = defineProps<{
  src: string
  file?: string
  openUrl?: string | null
}>()

const failed = ref(false)
const loading = ref(false)
const resolvedSrc = ref('')

watch(
  () => [props.src, props.file],
  () => {
    void refreshImage()
  },
  { immediate: true },
)

async function refreshImage() {
  failed.value = false
  if (!props.src) {
    resolvedSrc.value = ''
    failed.value = true
    return
  }

  if (!needsImageProxy(props.src)) {
    resolvedSrc.value = props.src
    return
  }

  loading.value = true
  const proxied = await loadChatImage(props.src, props.file)
  loading.value = false

  if (!proxied) {
    resolvedSrc.value = ''
    failed.value = true
    return
  }

  resolvedSrc.value = proxied
}

function openImage() {
  if (!props.openUrl) {
    return
  }
  window.open(props.openUrl, '_blank', 'noopener,noreferrer')
}
</script>
