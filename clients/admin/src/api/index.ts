import { apiClient } from "./client";
import {
    createAuthApi,
    createAdminApi,
    createUserAdminApi,
    createRbacApi,
} from "@stuhelper/shared/api";

export const api = {
    auth: createAuthApi(apiClient),
    admin: createAdminApi(apiClient),
    userAdmin: createUserAdminApi(apiClient),
    rbac: createRbacApi(apiClient),
};

export type { components } from "@stuhelper/shared";
