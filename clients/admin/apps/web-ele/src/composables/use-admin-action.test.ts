import { describe, expect, it, vi } from 'vitest';

import { useAdminAction } from './use-admin-action';

const mocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  t: vi.fn((key: string) => key),
}));

vi.mock('element-plus', () => ({
  ElMessage: {
    error: mocks.error,
    success: mocks.success,
  },
}));

vi.mock('#/locales', () => ({
  $t: mocks.t,
}));

interface Deferred {
  promise: Promise<void>;
  reject: (reason: unknown) => void;
  resolve: () => void;
}

function deferred(): Deferred {
  let resolve!: () => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, reject, resolve };
}

describe('useAdminAction', () => {
  it('locks per row id while leaving other rows interactive', async () => {
    const { isActionPending, runAction } = useAdminAction();
    const gate = deferred();
    const blocked = vi.fn().mockResolvedValue(undefined);

    const firstRun = runAction(() => gate.promise, { id: 'row-a' });

    expect(isActionPending('row-a')).toBe(true);
    expect(isActionPending('row-b')).toBe(false);

    // Re-entry on the same row is rejected without invoking the action.
    const reentry = await runAction(blocked, { id: 'row-a' });
    expect(reentry).toBe(false);
    expect(blocked).not.toHaveBeenCalled();

    // A different row can act concurrently.
    const other = await runAction(blocked, { id: 'row-b' });
    expect(other).toBe(true);
    expect(blocked).toHaveBeenCalledOnce();

    gate.resolve();
    await expect(firstRun).resolves.toBe(true);
    expect(isActionPending('row-a')).toBe(false);
  });

  it('shows a single failure toast and keeps a dismissible inline error', async () => {
    const { actionError, clearActionError, runAction } = useAdminAction();

    const ok = await runAction(() =>
      Promise.reject(new Error('操作冲突，请刷新后重试')),
    );

    expect(ok).toBe(false);
    expect(actionError.value).toBe('操作冲突，请刷新后重试');
    expect(mocks.error).toHaveBeenCalledTimes(1);
    expect(mocks.error).toHaveBeenCalledWith('操作冲突，请刷新后重试');

    clearActionError();
    expect(actionError.value).toBe('');
  });

  it('falls back to the i18n message when the failure carries no message', async () => {
    const { actionError, runAction } = useAdminAction();
    const messagelessError = new Error('placeholder');
    messagelessError.message = '';

    await runAction(() => Promise.reject(messagelessError));

    expect(actionError.value).toBe('admin.result.requestFailed');
  });

  it('emits the caller-provided success toast and releases the lock', async () => {
    mocks.success.mockClear();
    const { actionPending, runAction } = useAdminAction();

    const ok = await runAction(() => Promise.resolve(), {
      id: 7,
      successMessage: 'created!',
    });

    expect(ok).toBe(true);
    expect(mocks.success).toHaveBeenCalledWith('created!');
    expect(actionPending.value).toBe(false);
  });

  it('releases the lock even when the action throws', async () => {
    const { actionPending, isActionPending, runAction } = useAdminAction();

    await runAction(() => Promise.reject(new Error('boom')), { id: 'row-a' });

    expect(isActionPending('row-a')).toBe(false);
    expect(actionPending.value).toBe(false);
  });

  it('exposes pending action kinds scoped to the locked row', async () => {
    const { pendingActionKinds, runAction } = useAdminAction();
    const rowGate = deferred();
    const pageGate = deferred();

    const rowRun = runAction(() => rowGate.promise, {
      id: 'row-a',
      kind: 'approve',
    });
    const pageRun = runAction(() => pageGate.promise, { kind: 'import' });

    // Page-level kinds never leak into the per-row map.
    expect(pendingActionKinds.value).toEqual({ 'row-a': 'approve' });

    rowGate.resolve();
    pageGate.resolve();
    await Promise.all([rowRun, pageRun]);

    expect(pendingActionKinds.value).toEqual({});
  });

  it('serializes page-level actions behind a shared lock', async () => {
    const { actionPending, runAction } = useAdminAction();
    const gate = deferred();
    const blocked = vi.fn().mockResolvedValue(undefined);

    const firstRun = runAction(() => gate.promise);
    expect(actionPending.value).toBe(true);

    const reentry = await runAction(blocked);
    expect(reentry).toBe(false);
    expect(blocked).not.toHaveBeenCalled();

    gate.resolve();
    await firstRun;
    expect(actionPending.value).toBe(false);
  });
});
