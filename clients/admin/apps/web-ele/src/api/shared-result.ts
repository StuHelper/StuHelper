import type { ApiCallResult, ApiEnvelope } from '@stuhelper/shared/api';

import {
  extractOptionalResultData,
  extractResultData,
  extractResultErrorCode,
  extractResultErrorMessage,
  extractResultList,
  isResultFailure,
  readResultStatus,
} from '@stuhelper/shared/api';
import { ElMessage } from 'element-plus';

import { $t } from '#/locales';

export type { ApiCallResult, ApiEnvelope };

const BUSINESS_ERROR_MESSAGE_KEYS: Record<string, string> = {
  A0040011: 'admin.result.invalidAcademicTable',
  A0040012: 'admin.result.academicTableRequired',
  A0040013: 'admin.result.ldapConfigRequired',
  A0040014: 'admin.result.ldapConfigInvalid',
  A0040015: 'admin.result.systemConfigNotFound',
};

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  const code = extractResultErrorCode(result);
  const status = readResultStatus(result);

  if (status === 401 || code?.startsWith('A00101')) {
    return $t('admin.result.authExpired');
  }
  if (status === 403 && code === 'A0010204') {
    return $t('admin.result.mfaEnrollmentRequired');
  }
  if (status === 412 && code === 'A0010205') {
    return $t('admin.result.stepUpRequired');
  }
  if (code && BUSINESS_ERROR_MESSAGE_KEYS[code]) {
    return $t(BUSINESS_ERROR_MESSAGE_KEYS[code]);
  }

  return extractResultErrorMessage(result, $t('admin.result.requestFailed'));
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

export function unwrapVoid(result: ApiCallResult<unknown>): void {
  const status = readResultStatus(result);
  if (
    !isResultFailure(result) &&
    status !== undefined &&
    status >= 200 &&
    status < 300
  ) {
    return;
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
