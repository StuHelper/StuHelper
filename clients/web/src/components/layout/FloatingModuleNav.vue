<template>
  <div
    class="fixed z-50 flex flex-col items-center gap-1.5 transition-opacity duration-base"
    :style="positionStyle"
    :class="isDragging ? 'cursor-grabbing' : 'cursor-grab'"
    @mouseenter="expanded = true"
    @mouseleave="expanded = false"
    @mousedown.prevent="startDrag"
    @touchstart.prevent="startDrag"
  >
    <!-- 当前模块图标（始终显示） -->
    <div class="relative group">
      <router-link
        :to="activeTab.to"
        class="w-10 h-10 rounded-full bg-bg-glass-heavy backdrop-blur-xl backdrop-saturate-150 border border-white/15 dark:border-white/8 shadow-md flex items-center justify-center no-underline transition-all duration-base hover:shadow-lg"
        @click.stop
      >
        <component :is="activeTab.icon" :size="20" class="text-primary" />
      </router-link>
      <span class="absolute right-full mr-2.5 top-1/2 -translate-y-1/2 px-2 py-1 text-xs font-medium text-text-primary bg-bg-glass-heavy backdrop-blur-lg rounded-md shadow-sm border border-white/15 dark:border-white/8 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity duration-fast pointer-events-none">
        {{ activeTab.label }}
      </span>
    </div>

    <!-- 展开的其他模块 -->
    <transition-group
      enter-active-class="transition-all duration-base ease-out"
      enter-from-class="opacity-0 scale-75"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition-all duration-fast ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-75"
    >
      <div
        v-for="tab in otherTabs"
        v-if="expanded"
        :key="tab.to"
        class="relative group"
      >
        <router-link
          :to="tab.to"
          class="w-9 h-9 rounded-full bg-bg-glass backdrop-blur-lg backdrop-saturate-150 border border-white/15 dark:border-white/8 shadow-sm flex items-center justify-center no-underline transition-colors duration-fast hover:bg-primary hover:text-white hover:border-primary"
          @click.stop
        >
          <component :is="tab.icon" :size="16" class="text-text-secondary" />
        </router-link>
        <span class="absolute right-full mr-2.5 top-1/2 -translate-y-1/2 px-2 py-1 text-xs font-medium text-text-primary bg-bg-glass-heavy backdrop-blur-lg rounded-md shadow-sm border border-white/15 dark:border-white/8 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity duration-fast pointer-events-none">
          {{ tab.label }}
        </span>
      </div>
    </transition-group>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, markRaw, type Component } from 'vue'
import { useRoute } from 'vue-router'
import { MessageSquare, GraduationCap, BookOpen, FolderOpen } from 'lucide-vue-next'

interface ModuleTab {
  to: string
  icon: Component
  label: string
}

const tabs: ModuleTab[] = [
  { to: '/review', icon: markRaw(MessageSquare), label: '评课' },
  { to: '/teacher', icon: markRaw(GraduationCap), label: '教师' },
  { to: '/spoc', icon: markRaw(BookOpen), label: 'SPOC' },
  { to: '/resource', icon: markRaw(FolderOpen), label: '资源' },
]

const route = useRoute()
const expanded = ref(false)
const isDragging = ref(false)

// 位置状态
const STORAGE_KEY = 'floating-nav-position'
const position = ref({ x: 16, y: Math.round(window.innerHeight / 2 - 20) })

function loadPosition() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as { x?: number; y?: number }
      position.value = { x: parsed.x ?? 16, y: parsed.y ?? position.value.y }
    }
  } catch {
    /* ignore malformed data */
  }
}

function savePosition() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(position.value))
}

const positionStyle = computed(() => ({
  right: `${position.value.x}px`,
  top: `${position.value.y}px`,
}))

const activeTab = computed(() => {
  return tabs.find((t) => route.path.startsWith(t.to)) || tabs[0]
})

const otherTabs = computed(() => {
  return tabs.filter((t) => t !== activeTab.value)
})

// 拖拽逻辑
let dragStartX = 0
let dragStartY = 0
let startPosX = 0
let startPosY = 0
let hasMoved = false

function startDrag(e: MouseEvent | TouchEvent) {
  isDragging.value = true
  hasMoved = false
  const point = 'touches' in e ? e.touches[0] : e
  dragStartX = point.clientX
  dragStartY = point.clientY
  startPosX = position.value.x
  startPosY = position.value.y

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.addEventListener('touchmove', onDrag)
  document.addEventListener('touchend', stopDrag)
}

function onDrag(e: MouseEvent | TouchEvent) {
  const point = 'touches' in e ? e.touches[0] : e
  const dx = dragStartX - point.clientX
  const dy = point.clientY - dragStartY

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) hasMoved = true

  const newX = Math.max(8, Math.min(window.innerWidth - 56, startPosX + dx))
  const newY = Math.max(8, Math.min(window.innerHeight - 56, startPosY + dy))
  position.value = { x: newX, y: newY }
}

function stopDrag() {
  isDragging.value = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDrag)
  document.removeEventListener('touchend', stopDrag)

  if (position.value.x < 40) {
    position.value.x = 8
  } else if (position.value.x > window.innerWidth - 80) {
    position.value.x = 8
  }

  if (hasMoved) {
    savePosition()
  }
}

function cleanup() {
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDrag)
  document.removeEventListener('touchend', stopDrag)
}

onMounted(loadPosition)
onUnmounted(cleanup)
</script>
