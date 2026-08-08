<template>
  <div class="grid grid-cols-6 gap-[6px] sm:gap-2" role="group" :aria-label="ariaLabel">
    <input
      v-for="index in length"
      :key="index"
      :value="displayValue(index - 1)"
      class="aspect-square w-full rounded-lg border bg-bg-card text-center font-mono text-lg font-semibold text-text-primary outline-none transition-colors duration-fast focus:border-primary focus:ring-1 focus:ring-primary/30 disabled:cursor-not-allowed disabled:opacity-50"
      :class="cellClass(index - 1)"
      :name="`${name}_${index}`"
      :type="inputType"
      inputmode="numeric"
      autocapitalize="none"
      spellcheck="false"
      :autocomplete="index === 1 ? 'one-time-code' : 'off'"
      maxlength="1"
      :disabled="disabled"
      :aria-label="cellAriaLabel(index - 1)"
      @focus="activeIndex = index - 1"
      @input="handleInput(index - 1, $event)"
      @keydown="handleKeydown(index - 1, $event)"
      @paste="handlePaste(index - 1, $event)"
    >
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: string
  length?: number
  disabled?: boolean
  ariaLabel?: string
  name?: string
  inputType?: 'text' | 'tel'
}>(), {
  length: 6,
  disabled: false,
  ariaLabel: 'verification code',
  name: 'verification_code',
  inputType: 'tel',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  complete: [value: string]
}>()

const activeIndex = ref(0)
const cells = ref<string[]>(fillCells(props.modelValue, props.length))

watch(
  () => props.modelValue,
  (value) => {
    cells.value = fillCells(value, props.length)
    const firstEmpty = cells.value.findIndex((cell) => cell === '')
    activeIndex.value = firstEmpty < 0 ? props.length - 1 : firstEmpty
  },
  { immediate: true },
)

function normalizeDigits(value: string, length: number): string[] {
  return value.replace(/\D/g, '').slice(0, length).split('')
}

function fillCells(value: string, length: number): string[] {
  const digits = normalizeDigits(value, length)
  return Array.from({ length }, (_, index) => digits[index] ?? '')
}

function displayValue(index: number): string {
  return cells.value[index] ?? ''
}

function cellClass(index: number): string {
  return index === activeIndex.value
    ? 'border-primary shadow-[0_0_0_1px_rgba(91,124,247,0.25)]'
    : 'border-border'
}

function cellAriaLabel(index: number): string {
  return `${props.ariaLabel} ${index + 1}`
}

function emitValue(nextDigits: string[]): void {
  cells.value = Array.from({ length: props.length }, (_, index) => nextDigits[index] ?? '')
  const code = cells.value.join('')
  emit('update:modelValue', code)
  if (code.length === props.length) {
    emit('complete', code)
  }
}

function focusCell(current: HTMLInputElement, index: number): void {
  const inputs = current.parentElement?.querySelectorAll('input')
  const target = inputs?.item(Math.max(0, Math.min(index, props.length - 1))) as HTMLInputElement | null
  target?.focus()
  target?.select()
}

function handleInput(index: number, event: Event): void {
  const target = event.target
  if (!(target instanceof HTMLInputElement)) return

  const incoming = normalizeDigits(target.value, props.length - index)
  const nextDigits = cells.value.slice()

  if (incoming.length === 0) {
    nextDigits[index] = ''
    emitValue(nextDigits)
    activeIndex.value = index
    focusCell(target, activeIndex.value)
    return
  }

  for (let offset = 0; offset < incoming.length; offset += 1) {
    nextDigits[index + offset] = incoming[offset] ?? ''
  }

  emitValue(nextDigits)
  activeIndex.value = Math.min(index + incoming.length, props.length - 1)
  focusCell(target, activeIndex.value)
}

function handleKeydown(index: number, event: KeyboardEvent): void {
  const current = event.currentTarget
  if (!(current instanceof HTMLInputElement)) return

  if (event.key === 'Backspace') {
    event.preventDefault()
    const nextDigits = cells.value.slice()
    if (nextDigits[index]) {
      nextDigits[index] = ''
      emitValue(nextDigits)
      activeIndex.value = Math.max(index - 1, 0)
      focusCell(current, activeIndex.value)
      return
    }
    if (index > 0) {
      nextDigits[index - 1] = ''
      emitValue(nextDigits)
      activeIndex.value = index - 1
      focusCell(current, activeIndex.value)
    }
    return
  }

  if (event.key === 'ArrowLeft' && index > 0) {
    event.preventDefault()
    activeIndex.value = index - 1
    focusCell(current, activeIndex.value)
    return
  }

  if (event.key === 'ArrowRight' && index < props.length - 1) {
    event.preventDefault()
    activeIndex.value = index + 1
    focusCell(current, activeIndex.value)
  }
}

function handlePaste(index: number, event: ClipboardEvent): void {
  const current = event.currentTarget
  if (!(current instanceof HTMLInputElement)) return

  const text = event.clipboardData?.getData('text') ?? ''
  if (!text) return
  event.preventDefault()

  const incoming = normalizeDigits(text, props.length - index)
  if (incoming.length === 0) return

  const nextDigits = cells.value.slice()
  for (let offset = 0; offset < incoming.length; offset += 1) {
    nextDigits[index + offset] = incoming[offset] ?? ''
  }

  emitValue(nextDigits)
  activeIndex.value = Math.min(index + incoming.length, props.length - 1)
  focusCell(current, activeIndex.value)
}
</script>
