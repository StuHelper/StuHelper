<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'

interface Props {
  text: string
  speed?: number
}

const props = withDefaults(defineProps<Props>(), {
  speed: 100
})

const displayedText = ref('')
let intervalId: number | undefined
let currentIndex = 0

function startTyping(text: string) {
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

// React to text prop changes by restarting the typing animation
watch(() => props.text, (newText) => {
  if (intervalId !== undefined) {
    clearInterval(intervalId)
    intervalId = undefined
  }
  startTyping(newText)
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
