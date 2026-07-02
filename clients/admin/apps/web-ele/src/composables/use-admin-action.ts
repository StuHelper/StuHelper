import { computed, ref } from 'vue';

import { ElMessage } from 'element-plus';

import { adminErrorMessage } from './use-admin-list';

const PAGE_LOCK_KEY = '__page__';
const ID_LOCK_PREFIX = 'id:';

function lockKey(id?: number | string): string {
  return id === undefined ? PAGE_LOCK_KEY : `${ID_LOCK_PREFIX}${id}`;
}

export interface RunAdminActionOptions {
  /**
   * Lock key for row-level pending state. Actions with the same id are
   * serialized while unrelated rows stay interactive. Omit for page-level
   * actions (e.g. create dialogs), which share a single page lock.
   */
  id?: number | string;
  /**
   * Optional action label (e.g. 'approve' / 'reject') exposed through
   * `pendingActionKinds` so views can scope loading spinners to the exact
   * button that triggered the mutation.
   */
  kind?: string;
  /** Translated success toast text supplied by the caller. */
  successMessage?: string;
}

/**
 * Mutation runner shared by admin views.
 *
 * Owns the single presentation layer for action failures: one toast plus a
 * dismissible inline `actionError`. The API layer only throws typed errors.
 */
export function useAdminAction() {
  const actionError = ref('');
  const pendingByKey = ref<ReadonlyMap<string, string>>(new Map());

  const actionPending = computed(() => pendingByKey.value.size > 0);

  /** Row id → pending action kind, for per-action loading indicators. */
  const pendingActionKinds = computed<Readonly<Record<string, string>>>(() => {
    const kinds: Record<string, string> = {};
    for (const [key, kind] of pendingByKey.value) {
      if (key.startsWith(ID_LOCK_PREFIX) && kind !== '') {
        kinds[key.slice(ID_LOCK_PREFIX.length)] = kind;
      }
    }
    return kinds;
  });

  function isActionPending(id?: number | string): boolean {
    return pendingByKey.value.has(lockKey(id));
  }

  function clearActionError() {
    actionError.value = '';
  }

  async function runAction(
    action: () => Promise<unknown>,
    options: RunAdminActionOptions = {},
  ): Promise<boolean> {
    const key = lockKey(options.id);
    if (pendingByKey.value.has(key)) {
      return false;
    }

    pendingByKey.value = new Map([
      ...pendingByKey.value,
      [key, options.kind ?? ''],
    ]);
    actionError.value = '';
    try {
      await action();
      if (options.successMessage) {
        ElMessage.success(options.successMessage);
      }
      return true;
    } catch (error) {
      actionError.value = adminErrorMessage(error);
      ElMessage.error(actionError.value);
      return false;
    } finally {
      const remaining = new Map(pendingByKey.value);
      remaining.delete(key);
      pendingByKey.value = remaining;
    }
  }

  return {
    actionError,
    actionPending,
    clearActionError,
    isActionPending,
    pendingActionKinds,
    runAction,
  };
}
