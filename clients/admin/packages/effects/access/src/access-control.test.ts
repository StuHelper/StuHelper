import { mount } from '@vue/test-utils';

import { describe, expect, it, vi } from 'vitest';

import AccessControl from './access-control.vue';

vi.mock('./use-access', () => ({
  useAccess: () => ({
    hasAccessByCodes: vi.fn(() => false),
    hasAccessByRoles: vi.fn(() => false),
  }),
}));

describe('AccessControl', () => {
  it('renders slot content when no codes are provided', () => {
    const wrapper = mount(AccessControl, {
      slots: {
        default: '<div data-test="content">visible</div>',
      },
    });

    expect(wrapper.find('[data-test="content"]').exists()).toBe(true);
  });

  it('renders slot content when codes is an empty array', () => {
    const wrapper = mount(AccessControl, {
      props: {
        codes: [],
      },
      slots: {
        default: '<div data-test="content">visible</div>',
      },
    });

    expect(wrapper.find('[data-test="content"]').exists()).toBe(true);
  });
});
