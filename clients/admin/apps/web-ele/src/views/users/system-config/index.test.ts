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
  error: vi.fn(),
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
    ElAlert: {
      name: 'ElAlert',
      props: { title: String },
      template: '<section><strong>{{ title }}</strong><slot /></section>',
    },
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

function requireSystemConfig(config: SystemConfig | undefined): SystemConfig {
  if (!config) {
    throw new Error('Expected seeded system config');
  }
  return config;
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

function deferred<T>() {
  let reject!: (reason?: unknown) => void;
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((innerResolve, innerReject) => {
    resolve = innerResolve;
    reject = innerReject;
  });
  return { promise, reject, resolve };
}

describe('system config index view', () => {
  beforeEach(() => {
    apiMocks.getSystemConfigList.mockReset();
    apiMocks.updateSystemConfig.mockReset();
    messageMocks.error.mockReset();
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

    expect(readOnly.find('[data-email-policy-edit-button]').exists()).toBe(
      false,
    );
  });

  it('ignores stale system config responses when a newer refresh finishes first', async () => {
    const firstRequest = deferred<SystemConfig[]>();
    const secondRequest = deferred<SystemConfig[]>();
    const latestConfigs = [
      {
        ...requireSystemConfig(systemConfigs().at(0)),
        value: JSON.stringify({
          mode: 'weighted',
          maxAttempts: 1,
          providers: [
            { name: 'resend', enabled: true, priority: 10, weight: 50 },
          ],
        }),
      },
    ];

    apiMocks.getSystemConfigList.mockReset();
    apiMocks.getSystemConfigList
      .mockReturnValueOnce(firstRequest.promise)
      .mockReturnValueOnce(secondRequest.promise);

    const wrapper = mountPage();
    const vm = wrapper.vm as unknown as {
      configs: SystemConfig[];
      fetchData: () => Promise<void>;
    };
    expect(apiMocks.getSystemConfigList).toHaveBeenCalledTimes(1);

    const refresh = vm.fetchData();
    expect(apiMocks.getSystemConfigList).toHaveBeenCalledTimes(2);

    secondRequest.resolve(latestConfigs);
    await refresh;
    await flushPromises();

    expect(vm.configs).toEqual(latestConfigs);
    expect(wrapper.find('[data-email-policy-edit-button]').exists()).toBe(true);

    firstRequest.resolve(systemConfigs());
    await flushPromises();

    expect(vm.configs).toEqual(latestConfigs);
  });

  it('keeps list loading failures visible and retryable', async () => {
    apiMocks.getSystemConfigList
      .mockRejectedValueOnce(new Error('系统配置列表暂不可用'))
      .mockResolvedValueOnce(systemConfigs());

    const wrapper = mountPage();
    await flushPromises();

    const loadError = wrapper.find('.admin-load-error');
    expect(loadError.exists()).toBe(true);
    expect(loadError.text()).toContain('系统配置列表暂不可用');
    expect(wrapper.find('[data-email-policy-edit-button]').exists()).toBe(
      false,
    );

    await loadError.find('button').trigger('click');
    await flushPromises();

    expect(apiMocks.getSystemConfigList).toHaveBeenCalledTimes(2);
    expect(wrapper.find('.admin-load-error').exists()).toBe(false);
    expect(wrapper.find('[data-email-policy-edit-button]').exists()).toBe(true);
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
    if (!save) {
      throw new Error('Expected email policy save button');
    }
    await save.trigger('click');
    await flushPromises();

    expect(apiMocks.updateSystemConfig).toHaveBeenCalledTimes(1);
    const firstUpdateCall = apiMocks.updateSystemConfig.mock.calls.at(0);
    if (!firstUpdateCall) {
      throw new Error('Expected email policy update call');
    }
    const [key, payload] = firstUpdateCall;
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

  it('keeps update failures visible without closing the editor', async () => {
    apiMocks.updateSystemConfig.mockRejectedValueOnce(
      new Error('系统配置保存失败'),
    );
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-email-policy-edit-button]').trigger('click');
    await flushPromises();

    const save = wrapper
      .findAll('button')
      .find((button) => button.text() === 'admin.common.save');
    if (!save) {
      throw new Error('Expected email policy save button');
    }
    await save.trigger('click');
    await flushPromises();

    const actionError = wrapper.find('.admin-load-error');
    expect(actionError.exists()).toBe(true);
    expect(actionError.text()).toContain('系统配置保存失败');
    expect(wrapper.find('[data-dialog]').exists()).toBe(true);
    expect(messageMocks.error).toHaveBeenCalledWith('系统配置保存失败');
  });
});
