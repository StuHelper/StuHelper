import type { ApiClient } from './client';
import type { operations } from '../types/api.gen';

export type CourseListParams = operations['getCourses']['parameters']['query'];
export type CourseSearchParams = operations['searchCourses']['parameters']['query'];

export const createCourseApi = (client: ApiClient) => ({
    getDepartments: (params?: { category?: string }) =>
        client.GET('/api/v1/course/departments', {
            params: { query: params },
        }),

    getTerms: () => client.GET('/api/v1/course/terms'),

    getCategories: () => client.GET('/api/v1/course/categories'),

    getCourses: (params?: CourseListParams) =>
        client.GET('/api/v1/course/courses', {
            params: {
                query: params,
            },
        }),

    getCoursesGrouped: () =>
        client.GET('/api/v1/course/courses/grouped', {}),

    searchCourses: (query: string, params?: number | Omit<CourseSearchParams, 'q'>, options?: { signal?: AbortSignal }) => {
        const normalized =
            typeof params === 'number'
                ? { pageSize: params }
                : params;

        return client.GET('/api/v1/course/courses/search', {
            params: {
                query: {
                    q: query,
                    ...normalized,
                },
            },
            signal: options?.signal,
        });
    },

    getCourse: (id: number) =>
        client.GET('/api/v1/course/courses/{courseID}', { params: { path: { courseID: id } } }),

    getStats: () => client.GET('/api/v1/course/stats'),
});
