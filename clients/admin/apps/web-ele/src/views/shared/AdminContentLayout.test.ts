// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import AdminContentLayout from './AdminContentLayout.vue';

describe('AdminContentLayout', () => {
  it('renders the optional page description under the title', () => {
    const wrapper = mount(AdminContentLayout, {
      props: {
        description: 'Explain the page scope.',
        title: 'Admission policies',
        total: 3,
      },
    });

    expect(wrapper.get('h1').text()).toBe('Admission policies');
    expect(wrapper.get('.admin-content-page__description').text()).toBe(
      'Explain the page scope.',
    );
    expect(wrapper.get('.admin-content-page__total').text()).toBe('3');
  });

  it('does not render an empty description paragraph', () => {
    const wrapper = mount(AdminContentLayout, {
      props: { title: 'Reviews' },
    });

    expect(wrapper.find('.admin-content-page__description').exists()).toBe(
      false,
    );
  });
});
