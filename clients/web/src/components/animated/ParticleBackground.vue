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

const canvas = ref<HTMLCanvasElement>()
let ctx: CanvasRenderingContext2D | null = null
let particles: Particle[] = []
let animationId: number

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
  animate()
}

const resizeCanvas = () => {
  if (!canvas.value) return
  canvas.value.width = window.innerWidth
  canvas.value.height = window.innerHeight
}

const createParticles = () => {
  particles = []
  for (let i = 0; i < props.particleCount; i++) {
    const particle: Particle = {
      x: Math.random() * window.innerWidth,
      y: Math.random() * window.innerHeight,
      vx: 0,
      vy: 0,
      radius: Math.random() * 2 + 1
    }

    gsap.to(particle, {
      x: `+=${Math.random() * 200 - 100}`,
      y: `+=${Math.random() * 200 - 100}`,
      duration: Math.random() * 3 + 2,
      repeat: -1,
      yoyo: true,
      ease: 'sine.inOut'
    })

    particles.push(particle)
  }
}

const animate = () => {
  if (!ctx || !canvas.value) return

  ctx.clearRect(0, 0, canvas.value.width, canvas.value.height)

  // 绘制粒子
  particles.forEach(p => {
    ctx!.beginPath()
    ctx!.arc(p.x, p.y, p.radius, 0, Math.PI * 2)
    ctx!.fillStyle = props.color
    ctx!.fill()
  })

  // 绘制连接线
  for (let i = 0; i < particles.length; i++) {
    for (let j = i + 1; j < particles.length; j++) {
      const dx = particles[i].x - particles[j].x
      const dy = particles[i].y - particles[j].y
      const distance = Math.sqrt(dx * dx + dy * dy)

      if (distance < 120) {
        ctx!.beginPath()
        ctx!.moveTo(particles[i].x, particles[i].y)
        ctx!.lineTo(particles[j].x, particles[j].y)
        ctx!.strokeStyle = `${props.color}${Math.floor((1 - distance / 120) * 50).toString(16).padStart(2, '0')}`
        ctx!.lineWidth = 0.5
        ctx!.stroke()
      }
    }
  }

  animationId = requestAnimationFrame(animate)
}

const handleResize = () => {
  resizeCanvas()
  createParticles()
}

onMounted(() => {
  initCanvas()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  cancelAnimationFrame(animationId)
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
