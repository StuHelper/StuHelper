// @vitest-environment happy-dom

import type { SchoolConfig } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import IndexView from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getSchoolConfigList: vi.fn(),
  updateSchoolConfig: vi.fn(),
}));

const accessMocks = vi.hoisted(() => ({
  accessCodes: ['user:school:update'] as string[],
}));

const messageMocks = vi.hoisted(() => ({
  success: vi.fn(),
}));

vi.mock('#/api/admin', () => ({
  getSchoolConfigList: apiMocks.getSchoolConfigList,
  updateSchoolConfig: apiMocks.updateSchoolConfig,
}));

vi.mock('#/locales', () => ({
  $t: (key: string, params?: Record<string, unknown>) =>
    params ? `${key}:${params.enabled}/${params.total}` : key,
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
      template: '<button type="button" @click="$emit(\'click\', $event)"><slot /></button>',
    },
    ElMessage: messageMocks,
    ElSwitch: {
      name: 'ElSwitch',
      props: { modelValue: Boolean },
      emits: ['change'],
      template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'change\', $event.target.checked)" />',
    },
    ElTag: {
      name: 'ElTag',
      template: '<span><slot /></span>',
    },
  };
});

const schools: SchoolConfig[] = [
  {
    approvalPolicy: 'auto',
    createdAt: '2026-06-03T00:00:00Z',
    enabled: true,
    schoolCode: '4111010006',
    schoolID: 4111010006,
    schoolName: '北京航空航天大学',
    verificationMethod: 'manual',
  },
  {
    approvalPolicy: 'manual',
    createdAt: '2026-06-03T00:00:00Z',
    enabled: false,
    schoolCode: '4111010001',
    schoolID: 4111010001,
    schoolName: '北京大学',
    verificationMethod: 'manual',
  },
];

function mountPage() {
  return mount(IndexView, {
    global: {
      stubs: {
        AdminContentLayout: {
          name: 'AdminContentLayout',
          template:
            '<main><div data-toolbar><slot name="toolbar" /></div><slot /></main>',
        },
        PersistentAdminTable: {
          name: 'PersistentAdminTable',
          template: '<div data-stub="school-directory-table"><slot /></div>',
        },
        PersistentAdminTableColumn: {
          name: 'PersistentAdminTableColumn',
          template: '<div data-stub="school-directory-column" />',
        },
        SchoolConfigDialog: {
          name: 'SchoolConfigDialog',
          template: '<div data-stub="school-config-dialog" />',
        },
      },
    },
  });
}

describe('school config directory view', () => {
  beforeEach(() => {
    apiMocks.getSchoolConfigList.mockReset();
    apiMocks.updateSchoolConfig.mockReset();
    messageMocks.success.mockReset();
    accessMocks.accessCodes = ['user:school:update'];
    apiMocks.getSchoolConfigList.mockResolvedValue(schools.map((school) => ({ ...school })));
    apiMocks.updateSchoolConfig.mockResolvedValue(undefined);
  });

  it('summarizes enabled schools as the active school whitelist', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getSchoolConfigList).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[data-school-directory-summary]').text()).toBe(
      'admin.users.schoolConfig.enabledSummary:1/2',
    );
  });

  it('updates only the enabled flag when toggling a school', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      handleToggleEnabled: (row: SchoolConfig, enabled: boolean) => Promise<void>;
      schools: SchoolConfig[];
    };
    await vm.handleToggleEnabled(vm.schools[1]!, true);

    expect(apiMocks.updateSchoolConfig).toHaveBeenCalledWith(4111010001, {
      enabled: true,
    });
    expect(vm.schools[1]!.enabled).toBe(true);
    expect(messageMocks.success).toHaveBeenCalled();
  });
});
