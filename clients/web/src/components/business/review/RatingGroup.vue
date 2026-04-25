<template>
  <div class="flex flex-col gap-2">
    <div v-if="loading" class="flex flex-col gap-3">
      <div v-for="i in 5" :key="i" class="flex items-center gap-3">
        <div class="w-[80px] h-4 rounded-full shimmer-bg animate-shimmer bg-[length:200%_100%]"></div>
        <div class="w-[160px] h-7 rounded-full shimmer-bg animate-shimmer bg-[length:200%_100%]"></div>
      </div>
    </div>
    <div v-else-if="loadFailed" class="px-2 py-3 text-sm text-danger">
      {{ t('review.post.ratingLoadFailed') }}
    </div>
    <div v-else-if="resolvedDimensions.length === 0" class="px-2 py-3 text-sm text-text-muted">
      {{ t('review.post.ratingUnavailable') }}
    </div>

    <template v-else>
      <div
        v-for="dim in resolvedDimensions"
        :key="dim.key"
        class="flex items-center justify-between py-1.5 px-2 rounded-lg transition-all duration-200 ease-out hover:bg-bg-secondary"
      >
        <span class="font-medium text-text-primary text-sm shrink-0">
          {{ dim.name }}<span v-if="dim.description" class="text-text-muted font-normal ml-1.5">{{ dim.description }}</span>
        </span>

        <div class="flex gap-1.5">
          <button
            v-for="face in faces"
            :key="face.value"
            type="button"
            class="p-0.5 transition-all duration-fast ease-out hover:scale-[1.2] rounded-full"
            :class="getFaceClass(dim.key, face.value)"
            :title="face.label"
            @click="handleSelect(dim.key, face.value as RatingValue)"
            @mouseenter="handleHover(dim.key, face.value)"
            @mouseleave="handleHoverEnd"
          >
            <svg class="w-7 h-7 block" viewBox="0 0 512 512" fill="currentColor">
              <path :d="face.path" />
            </svg>
          </button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { RatingDimension, RatingValue } from '@stuhelper/shared/course'
import type { ReviewRatings } from '@stuhelper/shared/review'

const props = withDefaults(defineProps<{
  modelValue: ReviewRatings
  dimensions?: RatingDimension[]
  loading?: boolean
  loadFailed?: boolean
}>(), {
  dimensions: () => [],
  loading: false,
  loadFailed: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: ReviewRatings]
}>()

const { t } = useI18n()

const resolvedDimensions = computed(() => props.dimensions)
const hoverKey = ref('')
const hoverValue = ref(0)

// 5 → 爱 (grin-hearts), 4 → 笑 (smile), 3 → 平 (meh), 2 → 皱眉 (frown), 1 → 哭 (sad-cry)
const faces = [
  {
    value: 5,
    get label() { return t('review.rating.face5') },
    color: 'text-success',
    path: 'M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zM388.1 312.8c12.3-3.8 24.3 6.9 19.3 18.7C382.4 390.6 324.2 432 256.3 432s-126.2-41.4-151.1-100.5c-5-11.8 7-22.5 19.3-18.7c39.7 12.2 84.5 19 131.8 19s92.1-6.8 131.8-19zM199.3 129.1c17.8 4.8 28.4 23.1 23.6 40.8l-17.4 65c-2.3 8.5-11.1 13.6-19.6 11.3l-65.1-17.4c-17.8-4.8-28.4-23.1-23.6-40.8s23.1-28.4 40.8-23.6l16.1 4.3 4.3-16.1c4.8-17.8 23.1-28.4 40.8-23.6zm154.3 23.6l4.3 16.1 16.1-4.3c17.8-4.8 36.1 5.8 40.8 23.6s-5.8 36.1-23.6 40.8l-65.1 17.4c-8.5 2.3-17.3-2.8-19.6-11.3l-17.4-65c-4.8-17.8 5.8-36.1 23.6-40.8s36.1 5.8 40.9 23.6z'
  },
  {
    value: 4,
    get label() { return t('review.rating.face4') },
    color: 'text-success',
    path: 'M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zM164.1 325.5C182 346.2 212.6 368 256 368s74-21.8 91.9-42.5c5.8-6.7 15.9-7.4 22.6-1.6s7.4 15.9 1.6 22.6C349.8 372.1 311.1 400 256 400s-93.8-27.9-116.1-53.5c-5.8-6.7-5.1-16.8 1.6-22.6s16.8-5.1 22.6 1.6zM144.4 208a32 32 0 1 1 64 0 32 32 0 1 1 -64 0zm192-32a32 32 0 1 1 0 64 32 32 0 1 1 0-64z'
  },
  {
    value: 3,
    get label() { return t('review.rating.face3') },
    color: 'text-text-muted',
    path: 'M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zM176.4 176a32 32 0 1 1 0 64 32 32 0 1 1 0-64zm128 32a32 32 0 1 1 64 0 32 32 0 1 1 -64 0zM160 336l192 0c8.8 0 16 7.2 16 16s-7.2 16-16 16l-192 0c-8.8 0-16-7.2-16-16s7.2-16 16-16z'
  },
  {
    value: 2,
    get label() { return t('review.rating.face2') },
    color: 'text-warning',
    path: 'M256 512A256 256 0 1 0 256 0a256 256 0 1 0 0 512zM159.3 388.7c-2.6 8.4-11.6 13.2-20 10.5s-13.2-11.6-10.5-20C145.2 326.1 196.3 288 256 288s110.8 38.1 127.3 91.3c2.6 8.4-2.1 17.4-10.5 20s-17.4-2.1-20-10.5C340.5 349.4 302.1 320 256 320s-84.5 29.4-96.7 68.7zM144.4 208a32 32 0 1 1 64 0 32 32 0 1 1 -64 0zm192-32a32 32 0 1 1 0 64 32 32 0 1 1 0-64z'
  },
  {
    value: 1,
    get label() { return t('review.rating.face1') },
    color: 'text-danger',
    path: 'M352 493.4c-29.6 12-62.1 18.6-96 18.6s-66.4-6.6-96-18.6L160 288c0-8.8-7.2-16-16-16s-16 7.2-16 16l0 189.8C51.5 433.5 0 350.8 0 256C0 114.6 114.6 0 256 0S512 114.6 512 256c0 94.8-51.5 177.5-128 221.8L384 288c0-8.8-7.2-16-16-16s-16 7.2-16 16l0 205.4zM195.2 233.6c5.3 7.1 15.3 8.5 22.4 3.2s8.5-15.3 3.2-22.4c-30.4-40.5-91.2-40.5-121.6 0c-5.3 7.1-3.9 17.1 3.2 22.4s17.1 3.9 22.4-3.2c17.6-23.5 52.8-23.5 70.4 0zm121.6 0c17.6-23.5 52.8-23.5 70.4 0c5.3 7.1 15.3 8.5 22.4 3.2s8.5-15.3 3.2-22.4c-30.4-40.5-91.2-40.5-121.6 0c-5.3 7.1-3.9 17.1 3.2 22.4s17.1 3.9 22.4-3.2zM208 336l0 32c0 26.5 21.5 48 48 48s48-21.5 48-48l0-32c0-26.5-21.5-48-48-48s-48 21.5-48 48z'
  }
]

const faceColorMap: Record<number, string> = {
  1: 'text-danger',
  2: 'text-warning',
  3: 'text-text-muted',
  4: 'text-success',
  5: 'text-success'
}

const getFaceClass = (key: string, faceValue: number) => {
  const selected = props.modelValue[key]
  const hovering = hoverKey.value === key
  const isHovered = hovering && hoverValue.value === faceValue
  const isSelected = selected === faceValue

  if (isSelected || isHovered) {
    return [faceColorMap[faceValue], 'scale-[1.1]']
  }
  // 未选中且未 hover：灰色
  return 'text-text-muted/40'
}

const handleSelect = (key: string, value: RatingValue) => {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}

const handleHover = (key: string, value: number) => {
  hoverKey.value = key
  hoverValue.value = value
}

const handleHoverEnd = () => {
  hoverKey.value = ''
  hoverValue.value = 0
}

defineExpose({ dimensions: resolvedDimensions })
</script>

<style scoped>
/* Shimmer 渐变背景 — 三色渐变 + background-size 无法用纯 utility 表达 */
.shimmer-bg {
  background-image: linear-gradient(90deg, var(--color-bg-secondary) 25%, var(--color-bg-hover) 50%, var(--color-bg-secondary) 75%);
}

/* Vue Transition hooks */
.fade-enter-active,
.fade-leave-active {
  transition: opacity var(--duration-fast);
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
