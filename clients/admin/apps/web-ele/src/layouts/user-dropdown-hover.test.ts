import { mount } from '@vue/test-utils';
import { defineComponent, ref } from 'vue';

import { useHoverToggle } from '@vben/hooks';

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

describe('user dropdown hover controller', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not let a disabled hover leave timer close a click-opened menu', async () => {
    const wrapper = mount(
      defineComponent({
        setup() {
          const trigger = ref<HTMLElement | null>(null);
          const [open, controller] = useHoverToggle(trigger, 500);

          open.value = true;
          controller.disable();

          return { open, trigger };
        },
        template: '<button ref="trigger">Account</button>',
      }),
    );

    await vi.advanceTimersByTimeAsync(500);

    expect(wrapper.vm.open).toBe(true);
    wrapper.unmount();
  });
});
