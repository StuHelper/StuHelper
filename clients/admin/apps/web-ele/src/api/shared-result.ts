import type { ApiCallResult, ApiEnvelope } from '@stuhelper/shared/api';

import {
  extractOptionalResultData,
  extractResultData,
  extractResultErrorCode,
  extractResultList,
  readResultStatus,
} from '@stuhelper/shared/api';
import { ElMessage } from 'element-plus';

import { $t } from '#/locales';

export type { ApiCallResult, ApiEnvelope };

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  const code = extractResultErrorCode(result);
  const status = readResultStatus(result);

  if (status === 401 || code?.startsWith('A00101')) {
    return $t('admin.result.authExpired');
  }

  return $t('admin.result.requestFailed');
}

export function unwrapData<T>(result: ApiCallResult<T>): T {
  const payload = extractResultData(result);
  if (payload !== undefined) {
    return payload;
  }

  const message = extractErrorMessage(result);
  ElMessage.error(message);
  throw new Error(message);
}

export function unwrapOptionalData<T>(result: ApiCallResult<T>): null | T {
  const payload = extractOptionalResultData(result);
  if (payload !== undefined) {
    return payload;
  }

  const status = readResultStatus(result);
  if (status === 401 || status === 404) {
    return null;
  }

  const message = extractErrorMessage(result);
  ElMessage.error(message);
  throw new Error(message);
}

export function unwrapListData<T>(
  result: ApiCallResult<{ items?: T[]; list?: T[]; total?: number }>,
): {
  items: T[];
  total: number;
} {
  const payload = extractResultList(result);
  if (payload) {
    return { items: payload.list, total: payload.total };
  }

  const message = extractErrorMessage(result);
  ElMessage.error(message);
  throw new Error(message);
}
