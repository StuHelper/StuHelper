/**
 * 用户中心状态管理
 */
import { ref, type Ref } from "vue";
import { defineStore, getActivePinia } from "pinia";
import type { FavoriteCourse } from "@stuhelper/shared/course";
import { api } from "@/api";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

type PaginatedApiResponse = {
    data?: {
        data?: unknown;
    } | null;
};

function readPaginatedPayload<T>(
    payload: unknown,
    itemReader: (item: unknown) => T,
): { list: T[]; total: number } {
    if (!payload || typeof payload !== "object") {
        throw new Error("Invalid paginated response");
    }

    const { list, total } = payload as { list?: unknown; total?: unknown };
    if (
        !Array.isArray(list) ||
        typeof total !== "number" ||
        !Number.isInteger(total) ||
        total < 0
    ) {
        throw new Error("Invalid paginated response");
    }

    return { list: list.map(itemReader), total };
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function readString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string {
    const value = record[key];
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readOptionalString(
    record: Record<string, unknown>,
    key: string,
    message: string,
): string | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "string") {
        throw new Error(message);
    }
    return value;
}

function readInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number {
    const value = record[key];
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readOptionalInteger(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "number" || !Number.isInteger(value)) {
        throw new Error(message);
    }
    return value;
}

function readNumber(
    record: Record<string, unknown>,
    key: string,
    message: string,
): number {
    const value = record[key];
    if (typeof value !== "number" || !Number.isFinite(value)) {
        throw new Error(message);
    }
    return value;
}

function readOptionalBoolean(
    record: Record<string, unknown>,
    key: string,
    message: string,
): boolean | undefined {
    const value = record[key];
    if (value === undefined) {
        return undefined;
    }
    if (typeof value !== "boolean") {
        throw new Error(message);
    }
    return value;
}

function readFavoriteCourse(payload: unknown): FavoriteCourse {
    const message = "Invalid favorite course response";
    if (!isRecord(payload)) {
        throw new Error(message);
    }

    return {
        id: readInteger(payload, "id", message),
        schoolID: readOptionalInteger(payload, "schoolID", message),
        departmentID: readInteger(payload, "departmentID", message),
        departmentName: readOptionalString(payload, "departmentName", message),
        code: readOptionalString(payload, "code", message),
        name: readString(payload, "name", message),
        credits: readNumber(payload, "credits", message),
        category: readOptionalString(payload, "category", message),
        reviewCount: readInteger(payload, "reviewCount", message),
        isFavorited: readOptionalBoolean(payload, "isFavorited", message),
        favoritedAt: readString(payload, "favoritedAt", message),
    };
}

function readFavoriteStatus(payload: unknown): boolean {
    if (!payload || typeof payload !== "object") {
        throw new Error("Invalid favorite status response");
    }

    const { favorited } = payload as { favorited?: unknown };
    if (typeof favorited !== "boolean") {
        throw new Error("Invalid favorite status response");
    }

    return favorited;
}

async function fetchPaginated<T>(
    apiFn: (
        page: number,
        pageSize: number,
    ) => Promise<PaginatedApiResponse>,
    listRef: Ref<T[]>,
    totalRef: Ref<number>,
    loadingRef: Ref<boolean>,
    errorRef: Ref<string | null>,
    page: number,
    pageSize: number,
    itemReader: (item: unknown) => T,
    keySelector?: (item: T) => number | string | undefined,
) {
    loadingRef.value = true;
    errorRef.value = null;
    try {
        const res = await apiFn(page, pageSize);
        const { list: items, total } = readPaginatedPayload<T>(
            res.data?.data,
            itemReader,
        );
        if (page === 1) {
            listRef.value = items;
        } else if (!keySelector) {
            listRef.value = [...listRef.value, ...items];
        } else {
            const existingKeys = new Set(
                listRef.value
                    .map((item) => keySelector(item))
                    .filter((key): key is number | string => key !== undefined),
            );
            const newItems = items.filter((item) => {
                const key = keySelector(item);
                return key === undefined || !existingKeys.has(key);
            });
            listRef.value = [...listRef.value, ...newItems];
        }
        totalRef.value = total;
    } catch (err) {
        errorRef.value = err instanceof Error ? err.message : String(err);
        throw err;
    } finally {
        loadingRef.value = false;
    }
}

export const useUserStore = defineStore("user", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("user store requires an active Pinia instance");
    }

    const myFavorites = ref<FavoriteCourse[]>([]);
    const myFavoritesTotal = ref(0);
    const myFavoritesLoading = ref(false);
    const myFavoritesError = ref<string | null>(null);

    const favoriteStatus = ref<Partial<Record<number, boolean>>>({});

    const fetchMyFavorites = async (page = 1, pageSize = 10) => {
        await fetchPaginated(
            api.user.getMyFavorites,
            myFavorites,
            myFavoritesTotal,
            myFavoritesLoading,
            myFavoritesError,
            page,
            pageSize,
            readFavoriteCourse,
            (course) => course.id,
        );

        const updated = { ...favoriteStatus.value };
        for (const course of myFavorites.value) {
            updated[course.id] = true;
        }
        favoriteStatus.value = updated;
    };

    const toggleFavorite = async (courseID: number) => {
        const current = favoriteStatus.value[courseID] ?? false;

        favoriteStatus.value = {
            ...favoriteStatus.value,
            [courseID]: !current,
        };

        try {
            if (current) {
                await api.user.removeFavorite(courseID);
            } else {
                await api.user.addFavorite(courseID);
            }
        } catch (err) {
            favoriteStatus.value = {
                ...favoriteStatus.value,
                [courseID]: current,
            };
            throw err;
        }
    };

    const isFavorited = (courseID: number): boolean | undefined => {
        return favoriteStatus.value[courseID];
    };

    const setFavoriteStatus = (courseID: number, status: boolean) => {
        favoriteStatus.value = { ...favoriteStatus.value, [courseID]: status };
    };

    const ensureFavoriteStatus = async (courseID: number) => {
        if (favoriteStatus.value[courseID] !== undefined) return;
        try {
            const res = await api.user.getFavoriteStatus(courseID);
            const favorited = readFavoriteStatus(res.data?.data);
            favoriteStatus.value = {
                ...favoriteStatus.value,
                [courseID]: favorited,
            };
        } catch (_error) { void _error;
            // 加载失败保持未知状态
        }
    };

    const reset = () => {
        myFavorites.value = [];
        myFavoritesTotal.value = 0;
        myFavoritesLoading.value = false;
        myFavoritesError.value = null;
        favoriteStatus.value = {};
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "user",
        reset,
        pinia,
    );
    safeOnScopeDispose(unregisterSessionReset);

    return {
        myFavorites,
        myFavoritesTotal,
        myFavoritesLoading,
        myFavoritesError,
        favoriteStatus,
        fetchMyFavorites,
        toggleFavorite,
        isFavorited,
        setFavoriteStatus,
        ensureFavoriteStatus,
        reset,
    };
});
