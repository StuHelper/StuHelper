// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { defineComponent, h, nextTick } from 'vue';

import { describe, expect, it, vi } from 'vitest';

import {
  adminErrorMessage,
  useAdminList,
  useAdminLoad,
} from './use-admin-list';

const mocks = vi.hoisted(() => ({
  replace: vi.fn(() => Promise.resolve()),
  route: {
    path: '/content/reviews',
    query: {} as Record<string, string>,
  },
  t: vi.fn((key: string) => key),
  warn: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ replace: mocks.replace }),
}));

vi.mock('#/locales', () => ({
  $t: mocks.t,
}));

vi.mock('#/utils/admin-logger', () => ({
  adminLogger: { warn: mocks.warn },
}));

interface Deferred<T> {
  promise: Promise<T>;
  reject: (reason: unknown) => void;
  resolve: (value: T) => void;
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, reject, resolve };
}

function flushPromises() {
  return new Promise<void>((resolve) => setTimeout(resolve, 0));
}

function mountList<T>(setup: () => T): { result: T; unmount: () => void } {
  let result!: T;
  const wrapper = mount(
    defineComponent({
      setup() {
        result = setup();
        return () => h('div');
      },
    }),
  );
  return { result, unmount: () => wrapper.unmount() };
}

describe('useAdminList', () => {
  it('fetches on mount and exposes items and total', async () => {
    mocks.route.query = {};
    const fetcher = vi.fn().mockResolvedValue({
      items: [{ id: 'r1' }],
      total: 1,
    });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20, status: 'all' },
      }),
    );

    await flushPromises();

    expect(fetcher).toHaveBeenCalledOnce();
    expect(result.items.value).toEqual([{ id: 'r1' }]);
    expect(result.total.value).toBe(1);
    expect(result.loading.value).toBe(false);
    expect(result.loadError.value).toBe('');
  });

  it('hydrates filters and pagination from the route query', async () => {
    mocks.route.query = { page: '3', pageSize: '50', status: 'hidden' };
    const fetcher = vi.fn().mockResolvedValue({ items: [], total: 0 });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20, status: 'all' },
      }),
    );

    await flushPromises();

    expect(result.query.page).toBe(3);
    expect(result.query.pageSize).toBe(50);
    expect(result.query.status).toBe('hidden');
  });

  it('ignores malformed route query values', async () => {
    mocks.route.query = { page: 'NaN-page', pageSize: '-5' };
    const fetcher = vi.fn().mockResolvedValue({ items: [], total: 0 });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20 },
      }),
    );

    await flushPromises();

    expect(result.query.page).toBe(1);
    expect(result.query.pageSize).toBe(20);
  });

  it('mirrors non-default filters into the URL via router.replace', async () => {
    mocks.route.query = {};
    mocks.replace.mockClear();
    const fetcher = vi.fn().mockResolvedValue({ items: [], total: 0 });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20, status: 'all' },
      }),
    );

    await flushPromises();
    // Defaults only: nothing to mirror.
    expect(mocks.replace).not.toHaveBeenCalled();

    result.query.status = 'hidden';
    result.query.page = 2;
    await result.fetchData();

    expect(mocks.replace).toHaveBeenCalledWith({
      query: { page: '2', status: 'hidden' },
    });
  });

  it('discards stale responses so the latest request wins', async () => {
    mocks.route.query = {};
    const first = deferred<{ items: Array<{ id: string }>; total: number }>();
    const second = deferred<{ items: Array<{ id: string }>; total: number }>();
    const fetcher = vi
      .fn()
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20 },
      }),
    );

    await nextTick();
    void result.fetchData();

    second.resolve({ items: [{ id: 'fresh' }], total: 1 });
    await flushPromises();
    first.resolve({ items: [{ id: 'stale' }], total: 99 });
    await flushPromises();

    expect(result.items.value).toEqual([{ id: 'fresh' }]);
    expect(result.total.value).toBe(1);
    expect(result.loading.value).toBe(false);
  });

  it('records load failures without resetting previously loaded rows', async () => {
    mocks.route.query = {};
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce({ items: [{ id: 'r1' }], total: 1 })
      .mockRejectedValueOnce(new Error('网络异常'));

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20 },
      }),
    );

    await flushPromises();
    await result.fetchData();

    expect(result.loadError.value).toBe('网络异常');
    expect(result.items.value).toEqual([{ id: 'r1' }]);
    expect(result.loading.value).toBe(false);
  });

  it('skips the mount fetch when fetchOnMount is false', async () => {
    mocks.route.query = {};
    const fetcher = vi.fn().mockResolvedValue({ items: [], total: 0 });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        fetchOnMount: false,
        initialQuery: { page: 1, pageSize: 20 },
      }),
    );

    await flushPromises();
    expect(fetcher).not.toHaveBeenCalled();

    await result.fetchData();
    expect(fetcher).toHaveBeenCalledOnce();
  });

  it('resets to the first page before applying filters', async () => {
    mocks.route.query = {};
    const fetcher = vi.fn().mockResolvedValue({ items: [], total: 0 });

    const { result } = mountList(() =>
      useAdminList({
        fetcher,
        initialQuery: { page: 1, pageSize: 20 },
      }),
    );

    await flushPromises();
    result.query.page = 4;
    result.resetPageAndFetch();
    await flushPromises();

    expect(result.query.page).toBe(1);
    const lastCall = fetcher.mock.calls.at(-1)?.[0];
    expect(lastCall.page).toBe(1);
  });
});

describe('useAdminLoad', () => {
  it('loads on mount and surfaces failures through loadError', async () => {
    const loader = vi
      .fn()
      .mockResolvedValueOnce({ totalReviews: 7 })
      .mockRejectedValueOnce(new Error('统计服务不可用'));

    const { result } = mountList(() => useAdminLoad(loader));

    await flushPromises();
    expect(result.data.value).toEqual({ totalReviews: 7 });
    expect(result.loading.value).toBe(false);

    await result.load();
    expect(result.loadError.value).toBe('统计服务不可用');
    expect(result.data.value).toEqual({ totalReviews: 7 });
  });
});

describe('adminErrorMessage', () => {
  it('prefers typed error messages and falls back to the i18n copy', () => {
    const messagelessError = new Error('placeholder');
    messagelessError.message = '';

    expect(adminErrorMessage(new Error('请求超时'))).toBe('请求超时');
    expect(adminErrorMessage(messagelessError)).toBe(
      'admin.result.requestFailed',
    );
    expect(adminErrorMessage('not-an-error')).toBe(
      'admin.result.requestFailed',
    );
  });
});
