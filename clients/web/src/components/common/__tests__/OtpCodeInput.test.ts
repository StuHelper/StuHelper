// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'

import OtpCodeInput from '../OtpCodeInput.vue'

describe('OtpCodeInput', () => {
  it('keeps only digits, splits them into boxes, and emits complete at six digits', async () => {
    const onComplete = vi.fn()
    const Wrapper = defineComponent({
      components: { OtpCodeInput },
      setup() {
        const code = ref('')
        return { code, onComplete }
      },
      template: `
        <OtpCodeInput
          v-model="code"
          aria-label="verification code"
          @complete="onComplete"
        />
      `,
    })

    const wrapper = mount(Wrapper)
    const inputs = wrapper.findAll('input')

    await inputs[0].setValue('1')
    await inputs[1].setValue('2')
    await inputs[2].setValue('3')
    await inputs[3].setValue('4')
    await inputs[4].setValue('5')
    await inputs[5].setValue('6')
    await flushPromises()

    expect(wrapper.vm.code).toBe('123456')
    expect(onComplete).toHaveBeenCalledWith('123456')
    expect(wrapper.findAll('input')).toHaveLength(6)
    expect(inputs.map((input) => (input.element as HTMLInputElement).value)).toEqual([
      '1',
      '2',
      '3',
      '4',
      '5',
      '6',
    ])
  })

  it('pads nothing and only emits complete when six digits are present', async () => {
    const onComplete = vi.fn()
    const wrapper = mount(OtpCodeInput, {
      props: {
        modelValue: '12345',
        'onUpdate:modelValue': () => {},
        onComplete,
      },
    })

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('12345')
    await flushPromises()

    expect(onComplete).not.toHaveBeenCalled()
  })
})
