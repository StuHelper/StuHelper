import type { ApiCallResult, ApiEnvelope } from '@stuhelper/shared/api';

import {
  extractOptionalResultData,
  extractResultData,
  extractResultErrorCode,
  extractResultErrorMessage,
  extractResultList,
  isAuthSessionErrorCode,
  isResultFailure,
  MFA_ENROLLMENT_REQUIRED_CODE,
  readResultStatus,
  STEP_UP_REQUIRED_CODE,
  STEP_UP_REQUIRED_STATUSES,
} from '@stuhelper/shared/api';

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

  if (status === 401 || isAuthSessionErrorCode(code)) {
    return $t('admin.result.authExpired');
  }
  if (status === 403 && code === MFA_ENROLLMENT_REQUIRED_CODE) {
    return $t('admin.result.mfaEnrollmentRequired');
  }
  if (
    status !== undefined &&
    STEP_UP_REQUIRED_STATUSES.has(status) &&
    code === STEP_UP_REQUIRED_CODE
  ) {
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

  throw new Error(extractErrorMessage(result));
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

  throw new Error(extractErrorMessage(result));
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

  throw new Error(extractErrorMessage(result));
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

  throw new Error(extractErrorMessage(result));
}
