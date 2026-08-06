// @vitest-environment happy-dom

import type { AdmissionSession } from '#/api/admin';

import { flushPromises, mount } from '@vue/test-utils';

import { beforeEach, describe, expect, it, vi } from 'vitest';

import IndexView from './index.vue';

const apiMocks = vi.hoisted(() => ({
  cancelAdminAdmissionSession: vi.fn(),
  listAdmissionSessions: vi.fn(),
  regenerateAdminAdmissionSession: vi.fn(),
  resendAdminAdmissionSession: vi.fn(),
}));

const accessMocks = vi.hoisted(() => ({
  accessCodes: ['admission:session:manage'] as string[],
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

function mockExecCommand(result: boolean, copiedText: string[] = []) {
  const execCommand = vi.fn((command: string) => {
    if (command === 'copy') {
      copiedText.push(document.body.querySelector('textarea')?.value ?? '');
    }
    return result;
  });
  Object.defineProperty(document, 'execCommand', {
    configurable: true,
    value: execCommand,
  });
  return execCommand;
}

vi.mock('#/api/admin', () => ({
  cancelAdminAdmissionSession: apiMocks.cancelAdminAdmissionSession,
  listAdmissionSessions: apiMocks.listAdmissionSessions,
  regenerateAdminAdmissionSession: apiMocks.regenerateAdminAdmissionSession,
  resendAdminAdmissionSession: apiMocks.resendAdminAdmissionSession,
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

const sampleSession: AdmissionSession = {
  id: 'session-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: '178037297',
  channelID: '178037297',
  qqID: '1390191645',
  userID: '42',
  status: 'awaiting_requirements',
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
  authURL: 'https://join.stuhelper.com/verify/token',
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
    props: [
      'actionLoadingById',
      'canManage',
      'loading',
      'items',
      'total',
      'page',
      'pageSize',
    ],
    emits: [
      'copyAuthURL',
      'copyReissueCommand',
      'pageChange',
      'pageSizeChange',
      'requestCancel',
      'requestRegenerate',
      'requestResend',
    ],
    template: '<div data-stub="table" />',
  },
};

describe('admission sessions index view orchestration', () => {
  beforeEach(() => {
    apiMocks.cancelAdminAdmissionSession.mockReset();
    apiMocks.listAdmissionSessions.mockReset();
    apiMocks.regenerateAdminAdmissionSession.mockReset();
    apiMocks.resendAdminAdmissionSession.mockReset();
    messageMocks.error.mockReset();
    messageMocks.success.mockReset();
    accessMocks.accessCodes = ['admission:session:manage'];
    apiMocks.listAdmissionSessions.mockResolvedValue({
      items: [sampleSession],
      total: 1,
    });
    apiMocks.cancelAdminAdmissionSession.mockResolvedValue(sampleSession);
    apiMocks.regenerateAdminAdmissionSession.mockResolvedValue({
      authURL: sampleSession.authURL,
      session: sampleSession,
      token: 'redacted-test-token',
    });
    apiMocks.resendAdminAdmissionSession.mockResolvedValue(sampleSession);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
    mockExecCommand(true);
  });

  it('fetches qq admission sessions on mount with the default query shape', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledTimes(1);
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      platform: 'qq',
    });
    expect(
      wrapper
        .findComponent({ name: 'AdmissionSessionTable' })
        .props('canManage'),
    ).toBe(true);
  });

  it('passes read-only capability state to the table', async () => {
    accessMocks.accessCodes = ['admission:session:read'];

    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    expect(
      wrapper
        .findComponent({ name: 'AdmissionSessionTable' })
        .props('canManage'),
    ).toBe(false);
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
    vm.query.status = 'awaiting_requirements';

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
      status: 'awaiting_requirements',
    });
  });

  it('resets filters back to the qq production default', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const vm = wrapper.vm as unknown as {
      query: {
        botSelfID: string;
        guildID: string;
        platform: string;
        qqID: string;
        status: string;
      };
    };
    vm.query.platform = 'onebot';
    vm.query.botSelfID = '514';
    vm.query.guildID = 'guild-1';
    vm.query.qqID = '10001';
    vm.query.status = 'admitted';

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

  it('falls back to execCommand when auth URL clipboard writes fail', async () => {
    const copiedText: string[] = [];
    const execCommand = mockExecCommand(true, copiedText);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn().mockRejectedValue(new Error('permission denied')),
      },
    });
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('copyAuthURL', sampleSession.authURL);
    await flushPromises();

    expect(execCommand).toHaveBeenCalledWith('copy');
    expect(copiedText).toContain(sampleSession.authURL);
    expect(document.querySelector('textarea')).toBeNull();
    expect(messageMocks.success).toHaveBeenCalledWith('认证链接已复制');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('copies Koishi reissue commands for group operators', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('copyReissueCommand', '重新生成认证链接 1390191645');

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      '重新生成认证链接 1390191645',
    );
    expect(messageMocks.success).toHaveBeenCalledWith('重生命令已复制');
  });

  it('falls back to execCommand when copying reissue commands without clipboard API', async () => {
    const copiedText: string[] = [];
    const execCommand = mockExecCommand(true, copiedText);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    });
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('copyReissueCommand', '重新生成认证链接 1390191645');
    await flushPromises();

    expect(execCommand).toHaveBeenCalledWith('copy');
    expect(copiedText).toContain('重新生成认证链接 1390191645');
    expect(document.querySelector('textarea')).toBeNull();
    expect(messageMocks.success).toHaveBeenCalledWith('重生命令已复制');
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('reports clipboard failures when modern and fallback copy both fail', async () => {
    mockExecCommand(false);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: {
        writeText: vi.fn().mockRejectedValue(new Error('permission denied')),
      },
    });
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('copyReissueCommand', '重新生成认证链接 1390191645');
    await flushPromises();

    expect(messageMocks.error).toHaveBeenCalledWith('复制失败，请手动复制');
    expect(messageMocks.success).not.toHaveBeenCalledWith('重生命令已复制');
    expect(document.querySelector('textarea')).toBeNull();
  });

  it('runs admin session actions and refreshes the table', async () => {
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('requestResend', sampleSession.id);
    await flushPromises();
    await table.vm.$emit('requestRegenerate', sampleSession.id);
    await flushPromises();
    await table.vm.$emit('requestCancel', sampleSession.id);
    await flushPromises();

    expect(apiMocks.resendAdminAdmissionSession).toHaveBeenCalledWith(
      sampleSession.id,
    );
    expect(apiMocks.regenerateAdminAdmissionSession).toHaveBeenCalledWith(
      sampleSession.id,
    );
    expect(apiMocks.cancelAdminAdmissionSession).toHaveBeenCalledWith(
      sampleSession.id,
    );
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledTimes(3);
    expect(messageMocks.success).toHaveBeenCalledWith('已加入机器人重发队列');
    expect(messageMocks.success).toHaveBeenCalledWith(
      '已重新生成并加入机器人提醒队列',
    );
    expect(messageMocks.success).toHaveBeenCalledWith('认证会话已取消');
  });

  it('keeps session action loading scoped to the current session id', async () => {
    let resolveResend!: () => void;
    apiMocks.resendAdminAdmissionSession.mockReturnValueOnce(
      new Promise<AdmissionSession>((resolve) => {
        resolveResend = () => resolve(sampleSession);
      }),
    );
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('requestResend', sampleSession.id);
    await flushPromises();

    expect(apiMocks.resendAdminAdmissionSession).toHaveBeenCalledWith(
      sampleSession.id,
    );
    expect(table.props('loading')).toBe(false);
    expect(table.props('actionLoadingById')).toMatchObject({
      [sampleSession.id]: 'resend',
    });

    await table.vm.$emit('requestCancel', sampleSession.id);
    await flushPromises();

    expect(apiMocks.cancelAdminAdmissionSession).not.toHaveBeenCalled();

    resolveResend();
    await flushPromises();

    expect(
      (table.props('actionLoadingById') as Record<string, string | undefined>)[
        sampleSession.id
      ],
    ).toBeUndefined();
    expect(apiMocks.listAdmissionSessions).toHaveBeenCalledTimes(1);
  });

  it('keeps admin session action failures visible and does not refresh', async () => {
    apiMocks.resendAdminAdmissionSession.mockRejectedValueOnce(
      new Error('机器人队列写入失败'),
    );
    const wrapper = mount(IndexView, { global: { stubs: childStubs } });
    await flushPromises();
    apiMocks.listAdmissionSessions.mockClear();

    const table = wrapper.findComponent({ name: 'AdmissionSessionTable' });
    await table.vm.$emit('requestResend', sampleSession.id);
    await flushPromises();

    const actionError = wrapper.find('.admin-load-error');
    expect(actionError.exists()).toBe(true);
    expect(actionError.text()).toContain('机器人队列写入失败');
    expect(messageMocks.error).toHaveBeenCalledWith('机器人队列写入失败');
    expect(apiMocks.listAdmissionSessions).not.toHaveBeenCalled();
  });
});
