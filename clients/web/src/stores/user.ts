/**
 * 用户中心状态管理
 */
import { ref, type Ref } from "vue";
import { defineStore, getActivePinia } from "pinia";
import type { FavoriteCourse } from "@stuhelper/shared/course";
import { api } from "@/api";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

async function fetchPaginated<T>(
    apiFn: (
        page: number,
        pageSize: number,
    ) => Promise<{ data?: { data?: { list: T[]; total: number } } }>,
    listRef: Ref<T[]>,
    totalRef: Ref<number>,
    loadingRef: Ref<boolean>,
    errorRef: Ref<string | null>,
    page: number,
    pageSize: number,
    keySelector?: (item: T) => number | string | undefined,
) {
    loadingRef.value = true;
    errorRef.value = null;
    try {
        const res = await apiFn(page, pageSize);
        const items = res.data?.data?.list || [];
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
        totalRef.value = res.data?.data?.total || 0;
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
            const favorited = res.data?.data?.favorited ?? false;
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
