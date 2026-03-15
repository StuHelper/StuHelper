import { apiClient } from "./client";
import {
    createAuthApi,
    createAdminApi,
    createUserAdminApi,
} from "@stuhelper/shared/api";

// Raw fetch helper for endpoints not yet in generated types
const BASE = "/api/v1";

async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
    const url = `${BASE}${path}`;
    const res = await fetch(url, {
        credentials: "include",
        headers: { "Content-Type": "application/json", ...getCSRFHeader() },
        ...options,
    });
    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        const msg = body?.error?.message || res.statusText;
        const err = new Error(msg) as Error & { status: number };
        err.status = res.status;
        throw err;
    }
    return res.json();
}

function getCSRFHeader(): Record<string, string> {
    const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
    if (match) return { "X-CSRF-Token": decodeURIComponent(match[1]) };
    return {};
}

function qs(params?: Record<string, unknown>): string {
    if (!params) return "";
    const parts: string[] = [];
    for (const [k, v] of Object.entries(params)) {
        if (v !== undefined && v !== null && v !== "")
            parts.push(`${k}=${encodeURIComponent(String(v))}`);
    }
    return parts.length ? `?${parts.join("&")}` : "";
}

export const api = {
    auth: createAuthApi(apiClient),
    admin: createAdminApi(apiClient),
    userAdmin: createUserAdminApi(apiClient),
    userSystem: {
        // Identities
        listIdentities: (params?: {
            status?: string;
            page?: number;
            pageSize?: number;
        }) =>
            fetchJSON<{ data: { list: unknown[]; total: number } }>(
                `/admin/identities${qs(params)}`,
            ),
        reviewIdentity: (
            userID: number,
            data: { approved: boolean; rejectionReason?: string },
        ) =>
            fetchJSON(`/admin/identities/${userID}`, {
                method: "PUT",
                body: JSON.stringify(data),
            }),
        // Student verifications
        listStudentVerifications: (params?: {
            status?: string;
            schoolID?: string;
            page?: number;
            pageSize?: number;
        }) =>
            fetchJSON<{ data: { list: unknown[]; total: number } }>(
                `/admin/student-verifications${qs(params)}`,
            ),
        reviewStudentVerification: (
            userID: number,
            data: { approved: boolean; rejectionReason?: string },
        ) =>
            fetchJSON(`/admin/student-verifications/${userID}`, {
                method: "PUT",
                body: JSON.stringify(data),
            }),
        // School configs
        listSchoolConfigs: () =>
            fetchJSON<{ data: unknown[] }>("/admin/school-configs"),
        updateSchoolConfig: (
            schoolID: string,
            data: {
                schoolName: string;
                verificationMethod: string;
                consentText?: string;
                enabled: boolean;
            },
        ) =>
            fetchJSON(`/admin/school-configs/${schoolID}`, {
                method: "PUT",
                body: JSON.stringify(data),
            }),
        // System configs
        listSystemConfigs: () =>
            fetchJSON<{ data: unknown[] }>("/admin/system-configs"),
        updateSystemConfig: (key: string, data: { value: string }) =>
            fetchJSON(`/admin/system-configs/${key}`, {
                method: "PUT",
                body: JSON.stringify(data),
            }),
        // RBAC
        listRoles: () =>
            fetchJSON<{ data: { roles: unknown[] } }>("/admin/roles"),
        createRole: (data: {
            name: string;
            displayName: string;
            description?: string;
        }) =>
            fetchJSON("/admin/roles", {
                method: "POST",
                body: JSON.stringify(data),
            }),
        updateRole: (
            id: number,
            data: { displayName: string; description?: string },
        ) =>
            fetchJSON(`/admin/roles/${id}`, {
                method: "PUT",
                body: JSON.stringify(data),
            }),
        deleteRole: (id: number) =>
            fetchJSON(`/admin/roles/${id}`, { method: "DELETE" }),
        getRolePermissions: (id: number) =>
            fetchJSON<{ data: { permissions: unknown[] } }>(
                `/admin/roles/${id}/permissions`,
            ),
        setRolePermissions: (id: number, permissionIDs: number[]) =>
            fetchJSON(`/admin/roles/${id}/permissions`, {
                method: "PUT",
                body: JSON.stringify({ permissionIDs }),
            }),
        listPermissions: () =>
            fetchJSON<{ data: { permissions: unknown[] } }>(
                "/admin/permissions",
            ),
        listUserRoles: (userID: number) =>
            fetchJSON<{ data: { roles: unknown[] } }>(
                `/admin/users/${userID}/roles`,
            ),
        setUserRoles: (userID: number, roleIDs: number[]) =>
            fetchJSON(`/admin/users/${userID}/roles`, {
                method: "PUT",
                body: JSON.stringify({ roleIDs }),
            }),
    },
};

export type { components } from "@stuhelper/shared";
