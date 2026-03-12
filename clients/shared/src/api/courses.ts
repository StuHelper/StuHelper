import type { ApiClient } from "./client";

export interface CourseListParams {
    page?: number;
    pageSize?: number;
    limit?: number;
    q?: string;
    departmentID?: number;
    category?: string;
    sort?: 'name' | 'credits' | 'reviewCount';
}

export interface CourseSearchParams {
    page?: number;
    pageSize?: number;
    limit?: number;
}

export const createCourseApi = (client: ApiClient) => ({
    getDepartments: (params?: { category?: string }) =>
        client.GET("/api/v1/course/departments", {
            params: { query: params },
        }),

    getTerms: () => client.GET("/api/v1/course/terms"),

    getCategories: () => client.GET("/api/v1/course/categories"),

    getCourses: (params?: CourseListParams) => {
        const { limit, pageSize, ...rest } = params ?? {};
        return client.GET("/api/v1/course/courses", {
            params: {
                query: {
                    ...rest,
                    pageSize: pageSize ?? limit,
                },
            },
        });
    },

    searchCourses: (query: string, params?: number | CourseSearchParams, options?: { signal?: AbortSignal }) => {
        const normalized =
            typeof params === "number"
                ? { pageSize: params }
                : {
                      page: params?.page,
                      pageSize: params?.pageSize ?? params?.limit,
                  };

        return client.GET("/api/v1/course/courses/search", {
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
        client.GET("/api/v1/course/courses/{courseID}", { params: { path: { courseID: id } } }),

    getStats: () => client.GET("/api/v1/course/stats"),
});
