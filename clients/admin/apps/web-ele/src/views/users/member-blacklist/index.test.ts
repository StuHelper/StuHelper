// @vitest-environment happy-dom

import type { MemberBlacklistEntry } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import IndexView from './index.vue';

const apiMocks = vi.hoisted(() => ({
  listMemberBlacklist: vi.fn(),
  createMemberBlacklist: vi.fn(),
  releaseMemberBlacklist: vi.fn(),
}));

const accessMocks = vi.hoisted(() => ({
  accessCodes: ['member_blacklist:manage'] as string[],
}));

const messageMocks = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
}));

vi.mock('#/api/admin', () => ({
  listMemberBlacklist: apiMocks.listMemberBlacklist,
  createMemberBlacklist: apiMocks.createMemberBlacklist,
  releaseMemberBlacklist: apiMocks.releaseMemberBlacklist,
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
    ElMessage: messageMocks,
  };
});

const sampleEntry: MemberBlacklistEntry = {
  id: 'entry-active',
  platform: 'qq',
  subjectType: 'qq_user',
  subjectID: '10001',
  scopeType: 'guild',
  guildID: 'guild-1',
  source: 'admission_failure',
  reasonCode: 'admission_timeout_limit',
  reasonText: 'too many failures',
  createdFrom: 'admin_console',
  createdByType: 'admin_user',
  createdByID: 'admin-1',
  createdAt: '2026-04-01T00:00:00Z',
  updatedAt: '2026-04-01T00:00:00Z',
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
};

const childStubs = {
  BlacklistFilters: {
    name: 'BlacklistFilters',
    props: [
      'platform',
      'scopeType',
      'source',
      'status',
      'guildID',
      'subjectID',
      'canManage',
    ],
    emits: ['search', 'reset', 'openCreate'],
    template: '<div data-stub="filters" />',
  },
  BlacklistTable: {
    name: 'BlacklistTable',
    props: ['loading', 'items', 'total', 'canManage', 'page', 'pageSize'],
    emits: ['release', 'pageChange', 'pageSizeChange'],
    template: '<div data-stub="table" />',
  },
  CreateBlacklistDialog: {
    name: 'CreateBlacklistDialog',
    props: ['visible', 'submitting'],
    emits: ['submit', 'update:visible'],
    template: '<div data-stub="create-dialog" />',
    methods: {
      reset() {
        // matches defineExpose contract on the real component
      },
    },
  },
  ReleaseBlacklistDialog: {
    name: 'ReleaseBlacklistDialog',
    props: ['visible', 'submitting', 'target'],
    emits: ['submit', 'update:visible'],
    template: '<div data-stub="release-dialog" />',
  },
};

describe('member-blacklist index view orchestration', () => {
  beforeEach(() => {
    apiMocks.listMemberBlacklist.mockReset();
    apiMocks.createMemberBlacklist.mockReset();
    apiMocks.releaseMemberBlacklist.mockReset();
    messageMocks.success.mockReset();
    messageMocks.error.mockReset();
    accessMocks.accessCodes = ['member_blacklist:manage'];

    apiMocks.listMemberBlacklist.mockResolvedValue({
      items: [sampleEntry],
      total: 1,
    });
    apiMocks.createMemberBlacklist.mockResolvedValue(sampleEntry);
    apiMocks.releaseMemberBlacklist.mockResolvedValue(sampleEntry);
  });

  it('fetches the active list on mount with the default query shape', async () => {
    mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'active',
    });
  });

  it('forwards trimmed filter values into listMemberBlacklist params and resets to page 1', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const vm = wrapper.vm as unknown as {
      query: {
        guildID: string;
        page: number;
        pageSize: number;
        platform: string;
        scopeType: string;
        source: string;
        status: string;
        subjectID: string;
      };
    };
    // operator is on a deep page when refining the filter — search must
    // jump back to page 1 so the new filter set is visible.
    vm.query.page = 5;
    vm.query.platform = '  qq  ';
    vm.query.scopeType = 'guild';
    vm.query.source = 'manual_admin';
    vm.query.guildID = '  g-7 ';
    vm.query.subjectID = ' 42 ';
    vm.query.status = 'active';

    const filters = wrapper.findComponent({ name: 'BlacklistFilters' });
    await filters.vm.$emit('search');
    await flushPromises();

    expect(vm.query.page).toBe(1);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'active',
      platform: 'qq',
      scopeType: 'guild',
      source: 'manual_admin',
      guildID: 'g-7',
      subjectID: '42',
    });
  });

  it('pageSizeChange resets to page 1 before refetching', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const vm = wrapper.vm as unknown as {
      query: { page: number; pageSize: number };
    };
    vm.query.page = 4;
    vm.query.pageSize = 50;

    const table = wrapper.findComponent({ name: 'BlacklistTable' });
    await table.vm.$emit('pageSizeChange');
    await flushPromises();

    expect(vm.query.page).toBe(1);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledWith({
      page: 1,
      pageSize: 50,
      status: 'active',
    });
  });

  it('pageChange refetches without resetting the current page', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const vm = wrapper.vm as unknown as {
      query: { page: number; pageSize: number };
    };
    vm.query.page = 3;

    const table = wrapper.findComponent({ name: 'BlacklistTable' });
    await table.vm.$emit('pageChange');
    await flushPromises();

    expect(vm.query.page).toBe(3);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledWith({
      page: 3,
      pageSize: 20,
      status: 'active',
    });
  });

  it('reset emission clears query state and refetches with status=active', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      query: { page: number; platform: string; status: string };
    };
    vm.query.platform = 'qq';
    vm.query.status = 'released';
    vm.query.page = 5;
    apiMocks.listMemberBlacklist.mockClear();

    const filters = wrapper.findComponent({ name: 'BlacklistFilters' });
    await filters.vm.$emit('reset');
    await flushPromises();

    expect(vm.query.platform).toBe('');
    expect(vm.query.status).toBe('active');
    expect(vm.query.page).toBe(1);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'active',
    });
  });

  it('forwards a create-dialog submit payload verbatim to createMemberBlacklist and refetches', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const payload = {
      platform: 'qq',
      subjectType: 'qq_user',
      subjectID: '20002',
      scopeType: 'guild' as const,
      guildID: 'guild-42',
      source: 'manual_admin' as const,
      reasonCode: 'manual_blacklist',
      reasonText: 'spamming',
      expiresAt: undefined,
      metadata: { operatorInput: '20002' },
    };
    const dialog = wrapper.findComponent({ name: 'CreateBlacklistDialog' });
    await dialog.vm.$emit('submit', payload);
    await flushPromises();

    expect(apiMocks.createMemberBlacklist).toHaveBeenCalledWith(payload);
    expect(messageMocks.success).toHaveBeenCalledWith('已将 20002 加入黑名单');
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
  });

  it('ignores duplicate create submissions while the first request is pending', async () => {
    let resolveCreate!: () => void;
    apiMocks.createMemberBlacklist.mockReturnValueOnce(
      new Promise<MemberBlacklistEntry>((resolve) => {
        resolveCreate = () => resolve(sampleEntry);
      }),
    );
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const payload = {
      platform: 'qq',
      subjectType: 'qq_user',
      subjectID: '20002',
      scopeType: 'guild' as const,
      guildID: 'guild-42',
      source: 'manual_admin' as const,
      reasonCode: 'manual_blacklist',
      reasonText: 'spamming',
      expiresAt: undefined,
      metadata: { operatorInput: '20002' },
    };
    const dialog = wrapper.findComponent({ name: 'CreateBlacklistDialog' });
    await dialog.vm.$emit('submit', payload);
    await flushPromises();

    expect(dialog.props('submitting')).toBe(true);

    await dialog.vm.$emit('submit', payload);
    await flushPromises();

    expect(apiMocks.createMemberBlacklist).toHaveBeenCalledTimes(1);

    resolveCreate();
    await flushPromises();

    expect(dialog.props('submitting')).toBe(false);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
  });

  it('forwards release-dialog submit to releaseMemberBlacklist with id and request shape', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const releasePayload = {
      id: 'entry-active',
      request: {
        releaseReasonCode: 'manual_pardon' as const,
        releaseReason: 'verified appeal',
      },
    };
    const table = wrapper.findComponent({ name: 'BlacklistTable' });
    await table.vm.$emit('release', sampleEntry);

    const releaseDialog = wrapper.findComponent({
      name: 'ReleaseBlacklistDialog',
    });
    expect(releaseDialog.props('target')).toEqual(sampleEntry);

    await releaseDialog.vm.$emit('submit', releasePayload);
    await flushPromises();

    expect(apiMocks.releaseMemberBlacklist).toHaveBeenCalledWith(
      'entry-active',
      releasePayload.request,
    );
    expect(messageMocks.success).toHaveBeenCalledWith('已解除黑名单：10001');
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
  });

  it('ignores stale and duplicate release submissions', async () => {
    let resolveRelease!: () => void;
    apiMocks.releaseMemberBlacklist.mockReturnValueOnce(
      new Promise<MemberBlacklistEntry>((resolve) => {
        resolveRelease = () => resolve(sampleEntry);
      }),
    );
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const table = wrapper.findComponent({ name: 'BlacklistTable' });
    await table.vm.$emit('release', sampleEntry);

    const releaseDialog = wrapper.findComponent({
      name: 'ReleaseBlacklistDialog',
    });

    await releaseDialog.vm.$emit('submit', {
      id: 'stale-entry',
      request: { releaseReasonCode: 'manual_pardon' },
    });
    await flushPromises();

    expect(apiMocks.releaseMemberBlacklist).not.toHaveBeenCalled();

    await releaseDialog.vm.$emit('submit', {
      id: sampleEntry.id,
      request: { releaseReasonCode: 'manual_pardon' },
    });
    await flushPromises();

    expect(releaseDialog.props('submitting')).toBe(true);

    await releaseDialog.vm.$emit('submit', {
      id: sampleEntry.id,
      request: { releaseReasonCode: 'manual_pardon' },
    });
    await flushPromises();

    expect(apiMocks.releaseMemberBlacklist).toHaveBeenCalledTimes(1);

    resolveRelease();
    await flushPromises();

    expect(releaseDialog.props('submitting')).toBe(false);
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
  });

  it('exposes canManage based on access codes', async () => {
    accessMocks.accessCodes = ['member_blacklist:read'];
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const filters = wrapper.findComponent({ name: 'BlacklistFilters' });
    expect(filters.props('canManage')).toBe(false);
  });
});
