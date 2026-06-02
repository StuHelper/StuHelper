// @vitest-environment happy-dom

import type { AdmissionSession } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import IndexView from './index.vue';

const apiMocks = vi.hoisted(() => ({
  listAdmissionSessions: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  success: vi.fn(),
}));

vi.mock('#/api/admin', () => ({
  listAdmissionSessions: apiMocks.listAdmissionSessions,
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<Record<string, unknown>>('element-plus');
  return {
    ...actual,
    ElMessage: messageMocks,
  };
});

const sampleSession: AdmissionSession = {
  id: 'session-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: '178037297',
  channelID: '178037297',
  qqID: '1390191645',
  userID: '42',
  status: 'linked',
  tokenExpiresAt: '2026-06-02T06:00:00Z',
  tokenConsumedAt: '2026-06-02T05:00:00Z',
  linkWaitDeadlineAt: '2026-06-02T06:00:00Z',
  submissionWaitDeadlineAt: '2026-06-02T07:00:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-07-02T05:00:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: 'unmute failed',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/token?qq=1390191645',
};

const childStubs = {
  AdmissionSessionFilters: {
    name: 'AdmissionSessionFilters',
    props: ['qqID', 'guildID', 'botSelfID', 'platform', 'status'],
    emits: ['search', 'reset'],
    template: '<div data-stub="filters" />',
  },
  AdmissionSessionTable: {
    name: 'AdmissionSessionTable',
    props: ['loading', 'items', 'total', 'page', 'pageSize'],
    emits: ['copyAuthURL', 'pageChange', 'pageSizeChange'],
    template: '<div data-stub="table" />',
  },
};

describe('admission sessions index view orchestration', () => {
  beforeEach(() => {
    apiMocks.listAdmissionSessions.mockReset();
    messageMocks.success.mockReset();
    apiMocks.listAdmissionSessions.mockResolvedValue({
      items: [sampleSession],
      total: 1,
    });
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
  });

  it('fetches qq admission sessions on mount with the default query shape', async () => {
    mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledTimes(1);
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      platform: 'qq',
    });
  });

  it('forwards trimmed filters into listAdmissionSessions params and resets to page 1', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const vm = wrapper.vm as unknown as {
      query: {
        botSelfID: string;
        guildID: string;
        page: number;
        pageSize: number;
        platform: string;
        qqID: string;
        status: string;
      };
    };
    vm.query.page = 4;
    vm.query.platform = ' qq ';
    vm.query.botSelfID = ' 2118785781 ';
    vm.query.guildID = ' 178037297 ';
    vm.query.qqID = ' 1390191645 ';
    vm.query.status = 'linked';

    const filters = wrapper.findComponent({ name: 'AdmissionSessionFilters' });
    await filters.vm.$emit('search');
    await flushPromises();

    expect(vm.query.page).toBe(1);
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      platform: 'qq',
      botSelfID: '2118785781',
      guildID: '178037297',
      qqID: '1390191645',
      status: 'linked',
    });
  });

  it('resets filters back to the qq production default', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const vm = wrapper.vm as unknown as {
      query: { botSelfID: string; guildID: string; platform: string; qqID: string; status: string };
    };
    vm.query.platform = 'onebot';
    vm.query.botSelfID = '514';
    vm.query.guildID = 'guild-1';
    vm.query.qqID = '10001';
    vm.query.status = 'verified';

    const filters = wrapper.findComponent({ name: 'AdmissionSessionFilters' });
    await filters.vm.$emit('reset');
    await flushPromises();

    expect(vm.query).toMatchObject({
      platform: 'qq',
      botSelfID: '',
      guildID: '',
      qqID: '',
      status: '',
    });
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      platform: 'qq',
    });
  });

  it('copies auth URLs from the table without exposing tokens in logs', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('copyAuthURL', sampleSession.authURL);

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      sampleSession.authURL,
    );
    expect(messageMocks.success).toHaveBeenCalledWith('认证链接已复制');
  });
});
