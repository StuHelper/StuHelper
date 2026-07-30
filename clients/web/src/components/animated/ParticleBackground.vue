<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import gsap from 'gsap'

interface Props {
  particleCount?: number
  color?: string
}

const props = withDefaults(defineProps<Props>(), {
  particleCount: 50,
  color: '#60a5fa'
})

const prefersReducedMotion = typeof window !== 'undefined'
  && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const canvas = ref<HTMLCanvasElement>()
let ctx: CanvasRenderingContext2D | null = null
let particles: Particle[] = []
let animationId: number
let resizeFrame: number | null = null
const CONNECTION_DISTANCE = 120
const NEIGHBOR_OFFSETS = [
  [0, 0],
  [1, 0],
  [0, 1],
  [1, 1],
  [1, -1],
] as const

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  radius: number
}

const initCanvas = () => {
  if (!canvas.value) return

  ctx = canvas.value.getContext('2d')
  if (!ctx) return

  resizeCanvas()
  createParticles()
  if (!prefersReducedMotion) {
    animate()
  } else {
    // Reduced motion: draw static particles once
    drawFrame()
  }
}

const resizeCanvas = () => {
  if (!canvas.value) return
  canvas.value.width = window.innerWidth
  canvas.value.height = window.innerHeight
}

const createParticles = () => {
  gsap.killTweensOf(particles)
  particles = []
  for (let i = 0; i < props.particleCount; i++) {
    const particle: Particle = {
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      vx: 0,
      vy: 0,
      radius: Math.random() * 2 + 1
    }

    if (!prefersReducedMotion) {
      gsap.to(particle, {
        x: `+=${Math.random() * 200 - 100}`,
        y: `+=${Math.random() * 200 - 100}`,
        duration: Math.random() * 3 + 2,
        repeat: -1,
        yoyo: true,
        ease: 'sine.inOut'
      })
    }

    particles.push(particle)
  }
}

const drawFrame = () => {
  const context = ctx
  const element = canvas.value
  if (!context || !element) return

  context.clearRect(0, 0, element.width, element.height)

  particles.forEach(p => {
    context.beginPath()
    context.arc(p.x, p.y, p.radius, 0, Math.PI * 2)
    context.fillStyle = props.color
    context.fill()
  })

  const grid = new Map<string, Particle[]>()
  const cellSize = CONNECTION_DISTANCE
  for (const particle of particles) {
    const cellX = Math.floor(particle.x / cellSize)
    const cellY = Math.floor(particle.y / cellSize)
    const key = `${cellX}:${cellY}`
    const bucket = grid.get(key)
    if (bucket) bucket.push(particle)
    else grid.set(key, [particle])
  }

  for (const [key, bucket] of grid) {
    const [cellXRaw, cellYRaw] = key.split(':')
    const cellX = Number(cellXRaw)
    const cellY = Number(cellYRaw)

    for (const [offsetX, offsetY] of NEIGHBOR_OFFSETS) {
      const neighborBucket = grid.get(`${cellX + offsetX}:${cellY + offsetY}`)
      if (!neighborBucket) continue

      for (let i = 0; i < bucket.length; i++) {
        const startIndex = offsetX === 0 && offsetY === 0 ? i + 1 : 0
        for (let j = startIndex; j < neighborBucket.length; j++) {
          const left = bucket[i]
          const right = neighborBucket[j]
          const dx = left.x - right.x
          const dy = left.y - right.y
          const distance = Math.hypot(dx, dy)

          if (distance >= CONNECTION_DISTANCE) continue

          context.beginPath()
          context.moveTo(left.x, left.y)
          context.lineTo(right.x, right.y)
          context.strokeStyle = `${props.color}${Math.floor((1 - distance / CONNECTION_DISTANCE) * 50).toString(16).padStart(2, '0')}`
          context.lineWidth = 0.5
          context.stroke()
        }
      }
    }
  }
}

const animate = () => {
  drawFrame()
  animationId = requestAnimationFrame(animate)
}

const handleResize = () => {
  if (resizeFrame !== null) return
  resizeFrame = requestAnimationFrame(() => {
    resizeFrame = null
    resizeCanvas()
    createParticles()
  })
}

onMounted(() => {
  initCanvas()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  if (animationId) cancelAnimationFrame(animationId)
  if (resizeFrame !== null) cancelAnimationFrame(resizeFrame)
  gsap.killTweensOf(particles)
})
</script>

<template>
  <canvas ref="canvas" class="particle-canvas" />
</template>

<style scoped>
.particle-canvas {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}
</style>
