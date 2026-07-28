// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'

import BaseDialog from '../Dialog.vue'
import DialogContent from '../DialogContent.vue'
import DialogDescription from '../DialogDescription.vue'
import DialogTitle from '../DialogTitle.vue'

const mountedWrappers: Array<{ unmount(): void }> = []

afterEach(() => {
  while (mountedWrappers.length > 0) {
    mountedWrappers.pop()?.unmount()
  }
  document.body.innerHTML = ''
})

function mountDialog() {
  const Host = defineComponent({
    components: {
      BaseDialog,
      DialogContent,
      DialogDescription,
      DialogTitle,
    },
    setup() {
      const open = ref(false)
      return { open }
    },
    template: `
      <button id="dialog-trigger" type="button" @click="open = true">打开</button>
      <BaseDialog :open="open" @update:open="open = $event">
        <DialogContent>
          <DialogTitle>确认操作</DialogTitle>
          <DialogDescription>请检查后继续。</DialogDescription>
          <button id="dialog-action" type="button">继续</button>
        </DialogContent>
      </BaseDialog>
    `,
  })

  const wrapper = mount(Host, { attachTo: document.body })
  mountedWrappers.push(wrapper)
  return wrapper
}

describe('Dialog accessibility contract', () => {
  it('connects its accessible name and description and moves focus inside', async () => {
    const wrapper = mountDialog()
    await wrapper.get('#dialog-trigger').trigger('click')
    await nextTick()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.getAttribute('aria-labelledby')).toBeTruthy()
    expect(dialog?.getAttribute('aria-describedby')).toBeTruthy()
    expect(document.getElementById(dialog?.getAttribute('aria-labelledby') ?? '')?.textContent)
      .toBe('确认操作')
    expect(document.getElementById(dialog?.getAttribute('aria-describedby') ?? '')?.textContent)
      .toBe('请检查后继续。')
    expect(document.activeElement).toBe(dialog)
  })

  it('traps Tab, closes with Escape, and restores focus to its trigger', async () => {
    const wrapper = mountDialog()
    const trigger = wrapper.get<HTMLButtonElement>('#dialog-trigger')
    trigger.element.focus()
    await trigger.trigger('click')
    await nextTick()

    const dialog = document.querySelector<HTMLElement>('[role="dialog"]')
    const closeButton = dialog?.querySelectorAll<HTMLButtonElement>('button').item(1)
    const firstButton = dialog?.querySelector<HTMLButtonElement>('#dialog-action')
    expect(dialog).not.toBeNull()
    expect(closeButton).not.toBeNull()
    expect(firstButton).not.toBeNull()

    dialog?.focus()
    dialog?.dispatchEvent(new KeyboardEvent('keydown', {
      bubbles: true,
      key: 'Tab',
      shiftKey: true,
    }))
    expect(document.activeElement).toBe(closeButton)

    closeButton?.focus()
    closeButton?.dispatchEvent(new KeyboardEvent('keydown', {
      bubbles: true,
      key: 'Tab',
    }))
    expect(document.activeElement).toBe(firstButton)

    firstButton?.dispatchEvent(new KeyboardEvent('keydown', {
      bubbles: true,
      key: 'Escape',
    }))
    await nextTick()
    await nextTick()

    expect(document.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger.element)
  })
})
