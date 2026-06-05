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
  error: vi.fn(),
  success: vi.fn(),
}));

vi.mock('#/api/admin', () => ({
  getSchoolConfigList: apiMocks.getSchoolConfigList,
  updateSchoolConfig: apiMocks.updateSchoolConfig,
}));

vi.mock('#/locales', () => ({
  $t: (key: string, params?: Record<string, unknown>) => {
    if (!params) return key;
    if ('enabled' in params && 'total' in params) {
      return `${key}:${params.enabled}/${params.total}`;
    }
    if ('domain' in params) {
      return `${key}:${params.domain}`;
    }
    return key;
  },
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
        '<button type="button" @click="$emit(\'click\', $event)"><slot /></button>',
    },
    ElMessage: messageMocks,
    ElSwitch: {
      name: 'ElSwitch',
      props: { modelValue: Boolean },
      emits: ['change'],
      template:
        '<input type="checkbox" :checked="modelValue" @change="$emit(\'change\', $event.target.checked)" />',
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
    schoolEmailIdentityPolicy: {
      requireStudentName: true,
      studentIDEmailDomain: 'buaa.edu.cn',
      type: 'academic_student_email',
    },
    schoolEmailOtpEnabled: true,
    schoolID: 4_111_010_006,
    schoolName: '北京航空航天大学',
    schoolSsoEnabled: true,
    verificationMethod: 'manual',
  },
  {
    approvalPolicy: 'manual',
    createdAt: '2026-06-03T00:00:00Z',
    enabled: false,
    schoolCode: '4111010001',
    schoolEmailOtpEnabled: false,
    schoolID: 4_111_010_001,
    schoolName: '北京大学',
    schoolSsoEnabled: false,
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

function requireSchool(school: SchoolConfig | undefined): SchoolConfig {
  if (!school) {
    throw new Error('Expected seeded school config');
  }
  return school;
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

describe('school config directory view', () => {
  beforeEach(() => {
    apiMocks.getSchoolConfigList.mockReset();
    apiMocks.updateSchoolConfig.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
    accessMocks.accessCodes = ['user:school:update'];
    apiMocks.getSchoolConfigList.mockResolvedValue(
      schools.map((school) => ({ ...school })),
    );
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

  it('ignores stale school list responses when a newer refresh finishes first', async () => {
    const firstRequest = deferred<SchoolConfig[]>();
    const secondRequest = deferred<SchoolConfig[]>();
    const latestSchools = [
      {
        ...requireSchool(schools.at(1)),
        enabled: true,
        schoolName: '最新学校配置',
      },
    ];

    apiMocks.getSchoolConfigList.mockReset();
    apiMocks.getSchoolConfigList
      .mockReturnValueOnce(firstRequest.promise)
      .mockReturnValueOnce(secondRequest.promise);

    const wrapper = mountPage();
    const vm = wrapper.vm as unknown as {
      fetchData: () => Promise<void>;
      schools: SchoolConfig[];
    };
    expect(apiMocks.getSchoolConfigList).toHaveBeenCalledTimes(1);

    const refresh = vm.fetchData();
    expect(apiMocks.getSchoolConfigList).toHaveBeenCalledTimes(2);

    secondRequest.resolve(latestSchools);
    await refresh;
    await flushPromises();

    expect(vm.schools).toEqual(latestSchools);
    expect(wrapper.find('[data-school-directory-summary]').text()).toBe(
      'admin.users.schoolConfig.enabledSummary:1/1',
    );

    firstRequest.resolve(schools.map((school) => ({ ...school })));
    await flushPromises();

    expect(vm.schools).toEqual(latestSchools);
    expect(wrapper.find('[data-school-directory-summary]').text()).toBe(
      'admin.users.schoolConfig.enabledSummary:1/1',
    );
  });

  it('keeps list loading failures visible and retryable', async () => {
    apiMocks.getSchoolConfigList
      .mockRejectedValueOnce(new Error('学校配置列表暂不可用'))
      .mockResolvedValueOnce(schools.map((school) => ({ ...school })));

    const wrapper = mountPage();
    await flushPromises();

    const loadError = wrapper.find('.admin-load-error');
    expect(loadError.exists()).toBe(true);
    expect(loadError.text()).toContain('学校配置列表暂不可用');
    expect(wrapper.find('[data-school-directory-summary]').text()).toBe(
      'admin.users.schoolConfig.enabledSummary:0/0',
    );

    await loadError.find('button').trigger('click');
    await flushPromises();

    expect(apiMocks.getSchoolConfigList).toHaveBeenCalledTimes(2);
    expect(wrapper.find('.admin-load-error').exists()).toBe(false);
    expect(wrapper.find('[data-school-directory-summary]').text()).toBe(
      'admin.users.schoolConfig.enabledSummary:1/2',
    );
  });

  it('updates only the enabled flag when toggling a school', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      handleToggleEnabled: (
        row: SchoolConfig,
        enabled: boolean,
      ) => Promise<void>;
      schools: SchoolConfig[];
    };
    const targetSchool = requireSchool(vm.schools.at(1));
    await vm.handleToggleEnabled(targetSchool, true);

    expect(apiMocks.updateSchoolConfig).toHaveBeenCalledWith(4_111_010_001, {
      enabled: true,
    });
    expect(targetSchool.enabled).toBe(true);
    expect(messageMocks.success).toHaveBeenCalled();
  });

  it('keeps toggle failures visible without mutating the row', async () => {
    apiMocks.updateSchoolConfig.mockRejectedValueOnce(
      new Error('学校配置启停失败'),
    );
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      handleToggleEnabled: (
        row: SchoolConfig,
        enabled: boolean,
      ) => Promise<void>;
      schools: SchoolConfig[];
    };
    const targetSchool = requireSchool(vm.schools.at(1));
    await vm.handleToggleEnabled(targetSchool, true);
    await flushPromises();

    const actionError = wrapper.find('.admin-load-error');
    expect(actionError.exists()).toBe(true);
    expect(actionError.text()).toContain('学校配置启停失败');
    expect(targetSchool.enabled).toBe(false);
    expect(messageMocks.error).toHaveBeenCalledWith('学校配置启停失败');
  });

  it('summarizes admission capabilities and email identity policy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      admissionCapabilityTags: (
        row: SchoolConfig,
      ) => Array<{ label: string; type: string }>;
      emailPolicySummary: (row: SchoolConfig) => string;
      schools: SchoolConfig[];
    };
    const buaa = requireSchool(vm.schools.at(0));
    const remoteSchool = requireSchool(vm.schools.at(1));

    expect(vm.admissionCapabilityTags(buaa).map((tag) => tag.label)).toEqual([
      'admin.users.schoolConfig.schoolSso',
      'admin.users.schoolConfig.schoolEmailOtp',
    ]);
    expect(vm.emailPolicySummary(buaa)).toBe(
      'admin.users.schoolConfig.emailPolicyDomain:@buaa.edu.cn · admin.users.schoolConfig.requiresStudentName',
    );
    expect(vm.admissionCapabilityTags(remoteSchool)).toEqual([
      {
        label: 'admin.users.schoolConfig.noAdmissionCapability',
        type: 'info',
      },
    ]);
    expect(vm.emailPolicySummary(remoteSchool)).toBe('');
  });

  it('submits the selected approval policy from the edit dialog form', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      form: {
        approvalPolicy: 'auto' | 'manual';
        schoolName: string;
      };
      handleSubmit: (form: unknown) => Promise<void>;
      openEdit: (row: SchoolConfig) => void;
      schools: SchoolConfig[];
    };

    vm.openEdit(requireSchool(vm.schools.at(0)));
    expect(vm.form.approvalPolicy).toBe('auto');

    vm.form.approvalPolicy = 'manual';
    vm.form.schoolName = '北航新配置';
    await vm.handleSubmit(vm.form);

    expect(apiMocks.updateSchoolConfig).toHaveBeenCalledWith(
      4_111_010_006,
      expect.objectContaining({
        approvalPolicy: 'manual',
        schoolName: '北航新配置',
      }),
    );
    expect(messageMocks.success).toHaveBeenCalled();
  });

  it('keeps submit failures visible and leaves the dialog state available', async () => {
    apiMocks.updateSchoolConfig.mockRejectedValueOnce(
      new Error('学校配置保存失败'),
    );
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      form: {
        approvalPolicy: 'auto' | 'manual';
        schoolName: string;
      };
      handleSubmit: (form: unknown) => Promise<void>;
      openEdit: (row: SchoolConfig) => void;
      schools: SchoolConfig[];
    };

    vm.openEdit(requireSchool(vm.schools.at(0)));
    vm.form.schoolName = '保存失败学校';
    await vm.handleSubmit(vm.form);
    await flushPromises();

    const actionError = wrapper.find('.admin-load-error');
    expect(actionError.exists()).toBe(true);
    expect(actionError.text()).toContain('学校配置保存失败');
    expect(messageMocks.error).toHaveBeenCalledWith('学校配置保存失败');
  });
});
