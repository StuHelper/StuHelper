import type { components } from '@stuhelper/shared';

import { createAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const adminApi = createAdminApi(sharedApiClient);

export type Review = components['schemas']['Review'];
export type Report = components['schemas']['ReviewReport'];
export type Teacher = components['schemas']['AdminTeacher'];
export type SensitiveWord = components['schemas']['AdminSensitiveWord'];
export type OperationLog = components['schemas']['AdminOperationLog'];
export type AdminStats = components['schemas']['AdminStats'];

export interface ReviewListParams {
  page?: number;
  pageSize?: number;
  status?: 'all' | 'deleted' | 'hidden' | 'pending_review' | 'published';
}

export async function getReviewList(params: ReviewListParams) {
  return unwrapListData<Review>(
    await adminApi.getReviews({
      page: params.page,
      pageSize: params.pageSize,
      status: params.status,
    }),
  );
}

export async function updateReview(
  reviewId: number | string,
  data: { action: 'delete' | 'hide' | 'restore'; reason?: string },
) {
  return unwrapData(await adminApi.updateReview(String(reviewId), data));
}

export async function batchUpdateReviews(data: {
  action: 'delete' | 'hide' | 'restore';
  ids: Array<number | string>;
  reason?: string;
}) {
  return unwrapData(
    await adminApi.batchUpdateReviews({
      action: data.action,
      ids: data.ids.map(String),
    }),
  );
}

export interface ReportListParams {
  page?: number;
  pageSize?: number;
  status?: 'all' | 'pending' | 'rejected' | 'resolved';
}

export async function getReportList(params: ReportListParams) {
  return unwrapListData<Report>(
    await adminApi.getReports({
      page: params.page,
      pageSize: params.pageSize,
      status: params.status,
    }),
  );
}

export async function processReport(
  reportId: number | string,
  data: { action: 'delete' | 'hide' | 'reject'; note?: string },
) {
  return unwrapData(await adminApi.processReport(String(reportId), data));
}

export interface TeacherListParams {
  departmentID?: number;
  keyword?: string;
  page?: number;
  pageSize?: number;
}

export async function getTeacherList(params: TeacherListParams) {
  return unwrapListData<Teacher>(
    await adminApi.getTeachers({
      departmentID: params.departmentID,
      page: params.page,
      pageSize: params.pageSize,
      search: params.keyword,
    }),
  );
}

export async function createTeacher(data: {
  departmentID?: number;
  name: string;
}) {
  return unwrapData(await adminApi.createTeacher(data));
}

export async function updateTeacher(
  teacherId: number,
  data: { departmentID?: null | number; name: string },
) {
  return unwrapData(await adminApi.updateTeacher(teacherId, data));
}

export async function deleteTeacher(teacherId: number) {
  return unwrapData(await adminApi.deleteTeacher(teacherId));
}

export async function getSensitiveWordList(params: {
  category?: string;
  level?: 'block' | 'review' | 'warn';
  page?: number;
  pageSize?: number;
}) {
  return unwrapListData<SensitiveWord>(
    await adminApi.getSensitiveWords({
      category: params.category,
      level: params.level,
      page: params.page,
      pageSize: params.pageSize,
    }),
  );
}

export async function createSensitiveWord(data: {
  category: string;
  level: 'block' | 'review' | 'warn';
  word: string;
}) {
  return unwrapData(await adminApi.createSensitiveWord(data));
}

export async function updateSensitiveWord(
  id: string,
  data: {
    category?: string;
    isActive?: boolean;
    level?: 'block' | 'review' | 'warn';
    word?: string;
  },
) {
  return unwrapData(await adminApi.updateSensitiveWord(id, data));
}

export async function deleteSensitiveWord(id: string) {
  return unwrapData(await adminApi.deleteSensitiveWord(id));
}

export async function getOperationLogs(params: {
  page?: number;
  pageSize?: number;
}) {
  return unwrapListData<OperationLog>(await adminApi.getLogs(params));
}

export async function getAdminStats() {
  return unwrapData<AdminStats>(await adminApi.getStats());
}
