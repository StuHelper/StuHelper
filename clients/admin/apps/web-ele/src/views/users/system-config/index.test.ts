// @vitest-environment happy-dom

import type { SystemConfig } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import IndexView from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getSystemConfigList: vi.fn(),
  updateSystemConfig: vi.fn(),
}));

const accessMocks = vi.hoisted(() => ({
  accessCodes: ['user:system:update'] as string[],
}));

const messageMocks = vi.hoisted(() => ({
  success: vi.fn(),
}));

vi.mock('#/api/admin', () => ({
  getSystemConfigList: apiMocks.getSystemConfigList,
  updateSystemConfig: apiMocks.updateSystemConfig,
}));

vi.mock('#/locales', () => ({
  $t: (key: string) => key,
}));

vi.mock('@vben/stores', () => ({
  useAccessStore: () => ({
    get accessCodes() {
      return accessMocks.accessCodes;
    },
  }),
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<Record<string, unknown>>('element-plus');
  return {
    ...actual,
    ElButton: {
      name: 'ElButton',
      emits: ['click'],
      template:
        '<button type="button" :disabled="$attrs.disabled" @click="$emit(\'click\', $event)"><slot /></button>',
    },
    ElDialog: {
      name: 'ElDialog',
      props: { modelValue: Boolean },
      template:
        '<section v-if="modelValue" data-dialog><slot /><footer><slot name="footer" /></footer></section>',
    },
    ElForm: { name: 'ElForm', template: '<form><slot /></form>' },
    ElFormItem: { name: 'ElFormItem', template: '<label><slot /></label>' },
    ElInput: {
      name: 'ElInput',
      props: { modelValue: String },
      emits: ['update:modelValue'],
      template:
        '<textarea v-if="$attrs.type === \'textarea\'" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" /><input v-else :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    },
    ElInputNumber: {
      name: 'ElInputNumber',
      props: { modelValue: Number },
      emits: ['update:modelValue'],
      template:
        '<input type="number" :value="modelValue" @input="$emit(\'update:modelValue\', Number($event.target.value))" />',
    },
    ElMessage: messageMocks,
    ElOption: {
      name: 'ElOption',
      props: { label: String, value: String },
      template: '<option :value="value">{{ label }}</option>',
    },
    ElSelect: {
      name: 'ElSelect',
      props: { modelValue: String },
      emits: ['update:modelValue'],
      template:
        '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><slot /></select>',
    },
    ElSwitch: {
      name: 'ElSwitch',
      props: { modelValue: Boolean },
      emits: ['update:modelValue'],
      template:
        '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />',
    },
  };
});

const emailPolicyValue = JSON.stringify({
  mode: 'priority',
  maxAttempts: 2,
  providers: [
    { name: 'tencent_ses', enabled: true, priority: 10, weight: 100 },
    { name: 'resend', enabled: true, priority: 20, weight: 100 },
  ],
});

function systemConfigs(): SystemConfig[] {
  return [
    {
      key: 'email.delivery_policy',
      value: emailPolicyValue,
      description: '邮件发送提供商策略',
      updatedAt: '2026-06-02T06:00:00Z',
    },
  ];
}

function mountPage() {
  return mount(IndexView, {
    global: {
      mocks: {
        $t: (key: string) => key,
      },
      stubs: {
        AdminContentLayout: {
          name: 'AdminContentLayout',
          template:
            '<main><div data-toolbar><slot name="toolbar" /></div><slot /></main>',
        },
        PersistentAdminTable: {
          name: 'PersistentAdminTable',
          template: '<div data-stub="system-config-table"><slot /></div>',
        },
        PersistentAdminTableColumn: {
          name: 'PersistentAdminTableColumn',
          template: '<div data-stub="system-config-column" />',
        },
      },
    },
  });
}

describe('system config index view', () => {
  beforeEach(() => {
    apiMocks.getSystemConfigList.mockReset();
    apiMocks.updateSystemConfig.mockReset();
    messageMocks.success.mockReset();
    accessMocks.accessCodes = ['user:system:update'];
    apiMocks.getSystemConfigList.mockResolvedValue(systemConfigs());
    apiMocks.updateSystemConfig.mockResolvedValue(undefined);
  });

  it('shows the structured email policy editor only to system config editors', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getSystemConfigList).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[data-email-policy-edit-button]').exists()).toBe(true);

    accessMocks.accessCodes = ['user:system:read'];
    const readOnly = mountPage();
    await flushPromises();

    expect(readOnly.find('[data-email-policy-edit-button]').exists()).toBe(false);
  });

  it('serializes provider priority, weight and enabled state back to email.delivery_policy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-email-policy-edit-button]').trigger('click');
    await flushPromises();

    await wrapper.find('[data-email-policy-mode-select]').setValue('weighted');
    await wrapper.find('[data-email-policy-max-attempts]').setValue('1');
    await wrapper
      .find('[data-email-policy-provider-enabled="tencent_ses"]')
      .setValue(false);
    await wrapper
      .find('[data-email-policy-provider-priority="resend"]')
      .setValue('10');
    await wrapper
      .find('[data-email-policy-provider-weight="tencent_ses"]')
      .setValue('50');

    const save = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.common.save');
    expect(save).toBeDefined();
    await save!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateSystemConfig).toHaveBeenCalledTimes(1);
    const [key, payload] = apiMocks.updateSystemConfig.mock.calls[0]!;
    expect(key).toBe('email.delivery_policy');
    const parsed = JSON.parse(payload.value);
    expect(parsed).toEqual({
      mode: 'weighted',
      maxAttempts: 1,
      providers: [
        { name: 'tencent_ses', enabled: false, priority: 10, weight: 50 },
        { name: 'resend', enabled: true, priority: 10, weight: 100 },
      ],
    });
    expect(messageMocks.success).toHaveBeenCalledWith(
      'admin.users.systemConfig.updated',
    );
  });
});
