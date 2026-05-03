import { apiClient } from "./client";
import {
    createAuthApi,
    createCourseApi,
    createUserApi,
    createIdentityApi,
    createUserAdminApi,
    createNotificationApi,
    createDraftApi,
    createAdminApi,
    createRatingApi,
    createReplyApi,
    createAdmissionApi,
    NOTIFICATION_STREAM_PATH,
} from "@stuhelper/shared/api";
import { createReviewAppApi } from "./review";

// 导出统一的 API 对象
export const api = {
    auth: createAuthApi(apiClient),
    course: createCourseApi(apiClient),
    review: createReviewAppApi(apiClient),
    user: createUserApi(apiClient),
    identity: createIdentityApi(apiClient),
    userAdmin: createUserAdminApi(apiClient),
    notification: createNotificationApi(apiClient),
    draft: createDraftApi(apiClient),
    admin: createAdminApi(apiClient),
    rating: createRatingApi(apiClient),
    reply: createReplyApi(apiClient),
    admission: createAdmissionApi(apiClient),
};

// 导出类型
export type { components } from "@stuhelper/shared/types";
export { NOTIFICATION_STREAM_PATH };
