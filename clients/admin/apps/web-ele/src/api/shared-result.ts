import { ElMessage } from 'element-plus';

import { extractApiErrorMessage } from '@stuhelper/shared/api';
import { $t } from '#/locales';

export interface ApiEnvelope<T> {
  success?: boolean;
  data?: T;
  error?: unknown;
  message?: string;
  code?: string;
}

export interface ApiCallResult<T = unknown> {
  data?: ApiEnvelope<T>;
  error?: unknown;
  response?: {
    status?: number;
  };
}

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  // 优先从 envelope 提取（使用 shared 统一解析）
  if (result.data) {
    const msg = extractApiErrorMessage(result.data, '');
    if (msg) return msg;
  }

  // runtime error
  if (result.error) {
    const msg = extractApiErrorMessage(result.error, '');
    if (msg) return msg;
  }

  if (result.response?.status === 401) {
    return $t('admin.result.authExpired');
  }

  return $t('admin.result.requestFailed');
}

export function unwrapData<T>(result: ApiCallResult<T>): T {
  if (result.data && 'data' in result.data && result.data.data !== undefined) {
    return result.data.data as T;
  }

  const message = extractErrorMessage(result);
  ElMessage.error(message);
  throw new Error(message);
}

export function unwrapOptionalData<T>(result: ApiCallResult<T>): null | T {
  if (result.data && 'data' in result.data) {
    return (result.data.data ?? null) as null | T;
  }

  if (result.response?.status === 401 || result.response?.status === 404) {
    return null;
  }

  const message = extractErrorMessage(result);
  ElMessage.error(message);
  throw new Error(message);
}

export function unwrapListData<T>(
  result: ApiCallResult<{ list?: T[]; total?: number }>,
): { items: T[]; total: number } {
  const payload = unwrapData(result);

  return {
    items: Array.isArray(payload.list) ? payload.list : [],
    total: typeof payload.total === 'number' ? payload.total : 0,
  };
}
