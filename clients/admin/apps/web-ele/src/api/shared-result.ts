import { ElMessage } from 'element-plus';

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

function readErrorMessage(value: unknown): null | string {
  if (typeof value === 'string') {
    return value.trim() || null;
  }

  if (!value || typeof value !== 'object') {
    return null;
  }

  const record = value as Record<string, unknown>;
  const message = record.message;
  if (typeof message === 'string' && message.trim()) {
    return message;
  }

  const error = record.error;
  if (typeof error === 'string' && error.trim()) {
    return error;
  }

  return null;
}

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  const dataError = readErrorMessage(result.data?.error);
  if (dataError) {
    return dataError;
  }

  if (result.data?.message) {
    return result.data.message;
  }

  const runtimeError = readErrorMessage(result.error);
  if (runtimeError) {
    return runtimeError;
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
