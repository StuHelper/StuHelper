<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useMediaQuery } from '@vueuse/core'

interface Props {
  text: string
  speed?: number
}

const props = withDefaults(defineProps<Props>(), {
  speed: 100
})

const prefersReducedMotion = useMediaQuery('(prefers-reduced-motion: reduce)')

const displayedText = ref('')
let intervalId: number | undefined
let currentIndex = 0

function startTyping(text: string) {
  // 用户偏好减少动态效果时，直接展示完整文本
  if (prefersReducedMotion.value) {
    displayedText.value = text
    return
  }

  currentIndex = 0
  displayedText.value = ''
  intervalId = window.setInterval(() => {
    if (currentIndex < text.length) {
      displayedText.value += text[currentIndex]
      currentIndex++
    } else {
      if (intervalId !== undefined) {
        clearInterval(intervalId)
        intervalId = undefined
      }
    }
  }, props.speed)
}

// 文本变化后重新开始逐字动画
watch(() => props.text, (newText) => {
  if (intervalId !== undefined) {
    clearInterval(intervalId)
    intervalId = undefined
  }
  startTyping(newText)
})

// 系统动态效果偏好变化后立即同步展示状态
watch(prefersReducedMotion, (reduced) => {
  if (reduced) {
    if (intervalId !== undefined) {
      clearInterval(intervalId)
      intervalId = undefined
    }
    displayedText.value = props.text
  }
})

onMounted(() => {
  startTyping(props.text)
})

onUnmounted(() => {
  if (intervalId !== undefined) {
    clearInterval(intervalId)
  }
})
</script>

<template>
  <span class="typewriter">{{ displayedText }}</span>
</template>

<style scoped>
.typewriter {
  color: inherit;
}
</style>
