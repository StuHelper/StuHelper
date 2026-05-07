// @vitest-environment happy-dom

import type { MemberBlacklistEntry } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

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

import IndexView from './index.vue';

const sampleEntry: MemberBlacklistEntry = {
  id: 'entry-active',
  platform: 'qq',
  subjectType: 'qq_user',
  subjectID: '10001',
  scopeType: 'guild',
  guildID: 'guild-1',
  source: 'admission_failure',
  reasonCode: 'admission_failed',
  reasonText: 'too many failures',
  createdFrom: 'admin_console',
  createdByType: 'admin',
  createdByID: 'admin-1',
  createdAt: '2026-04-01T00:00:00Z',
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
} as MemberBlacklistEntry;

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
    props: [
      'loading',
      'items',
      'total',
      'canManage',
      'page',
      'pageSize',
    ],
    emits: ['release', 'pageChange'],
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

  it('forwards trimmed filter values into listMemberBlacklist params', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listMemberBlacklist.mockClear();

    const vm = wrapper.vm as unknown as {
      query: {
        page: number;
        pageSize: number;
        platform: string;
        scopeType: string;
        source: string;
        status: string;
        guildID: string;
        subjectID: string;
      };
    };
    vm.query.platform = '  qq  ';
    vm.query.scopeType = 'guild';
    vm.query.source = 'manual_admin';
    vm.query.guildID = '  g-7 ';
    vm.query.subjectID = ' 42 ';
    vm.query.status = 'active';

    const filters = wrapper.findComponent({ name: 'BlacklistFilters' });
    await filters.vm.$emit('search');
    await flushPromises();

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

  it('reset emission clears query state and refetches with status=active', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      query: { platform: string; status: string; page: number };
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
    expect(messageMocks.success).toHaveBeenCalledWith(
      '已将 20002 加入黑名单',
    );
    expect(apiMocks.listMemberBlacklist).toHaveBeenCalledTimes(1);
  });

  it('forwards release-dialog submit to releaseMemberBlacklist with id and request shape', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const releasePayload = {
      id: 'entry-active',
      request: {
        releaseReasonCode: 'manual_pardon' as const,
        releaseReason: 'verified appeal',
      },
    };
    const releaseDialog = wrapper.findComponent({
      name: 'ReleaseBlacklistDialog',
    });
    await releaseDialog.vm.$emit('submit', releasePayload);
    await flushPromises();

    expect(apiMocks.releaseMemberBlacklist).toHaveBeenCalledWith(
      'entry-active',
      releasePayload.request,
    );
    expect(messageMocks.success).toHaveBeenCalled();
  });

  it('exposes canManage based on access codes', async () => {
    accessMocks.accessCodes = ['member_blacklist:read'];
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const filters = wrapper.findComponent({ name: 'BlacklistFilters' });
    expect(filters.props('canManage')).toBe(false);
  });
});
