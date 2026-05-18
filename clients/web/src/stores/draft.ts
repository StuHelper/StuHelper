/**
 * 草稿状态管理
 */
import { defineStore, getActivePinia } from "pinia";
import { computed, ref } from "vue";
import type { Draft, SaveDraftParams } from "@stuhelper/shared/draft";
import { isValidRating } from "@stuhelper/shared/course";
import { api } from "@/api";
import { safeOnScopeDispose } from "@/stores/safeScopeDispose";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

const CACHE_TTL_MS = 5 * 60 * 1000;

export const useDraftStore = defineStore("draft", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("draft store requires an active Pinia instance");
    }

    const draft = ref<Draft | null>(null);
    const cacheTimestamp = ref<number | null>(null);
    const saving = ref(false);
    const lastSavedAt = ref<Date | null>(null);
    const hasDraft = computed(() => draft.value !== null);
    const pendingSave = ref<Promise<Draft> | null>(null);

    function isCacheStale(): boolean {
        if (cacheTimestamp.value === null) return true;
        return Date.now() - cacheTimestamp.value > CACHE_TTL_MS;
    }

    function assertValidDraftParams(data: SaveDraftParams): void {
        if (data.courseID !== undefined && (!data.courseID || !Number.isFinite(data.courseID))) {
            throw new Error("invalid draft parameters");
        }
        if (data.teacherID !== undefined && (!data.teacherID || !Number.isFinite(data.teacherID))) {
            throw new Error("invalid draft parameters");
        }
        if (data.title !== undefined && typeof data.title !== "string") {
            throw new Error("invalid draft parameters");
        }
        if (data.content !== undefined && typeof data.content !== "string") {
            throw new Error("invalid draft parameters");
        }
    }

    function normalizeDraft(
        value:
            | {
                  id: string;
                  courseID?: number | null;
                  teacherID?: number | null;
                  termID?: string;
                  title?: string;
                  content?: string;
                  grade?: string;
                  ratings?: Record<string, number>;
                  updatedAt: string;
              }
            | undefined,
    ): Draft {
        if (!value) {
            throw new Error("invalid draft response");
        }

        let ratings: Draft["ratings"] = undefined;
        if (value.ratings) {
            const normalizedRatings: Record<string, 1 | 2 | 3 | 4 | 5> = {};
            for (const [key, rating] of Object.entries(value.ratings)) {
                if (!isValidRating(rating)) {
                    throw new Error("invalid draft response");
                }
                normalizedRatings[key] = rating;
            }
            ratings = normalizedRatings;
        }

        return {
            id: value.id,
            ...(value.courseID !== undefined && value.courseID !== null && {
                courseID: value.courseID,
            }),
            ...(value.teacherID !== undefined && {
                teacherID: value.teacherID,
            }),
            ...(value.termID !== undefined && { termID: value.termID }),
            ...(value.title !== undefined && { title: value.title }),
            ...(value.content !== undefined && { content: value.content }),
            ...(value.grade !== undefined && {
                grade: value.grade as Draft["grade"],
            }),
            ...(ratings !== undefined && { ratings }),
            updatedAt: value.updatedAt,
        };
    }

    function cacheDraft(value: Draft) {
        draft.value = value;
        cacheTimestamp.value = Date.now();
    }

    const saveDraft = async (data: SaveDraftParams) => {
        assertValidDraftParams(data);
        saving.value = true;
        const save = (async () => {
            const res = await api.draft.saveDraft(data);
            const nextDraft = normalizeDraft(res.data?.data ?? undefined);
            cacheDraft(nextDraft);
            lastSavedAt.value = new Date();
            return nextDraft;
        })();
        pendingSave.value = save;
        try {
            return await save;
        } finally {
            if (pendingSave.value === save) {
                pendingSave.value = null;
            }
            saving.value = false;
        }
    };

    const loadDraft = async (forceRefresh = false) => {
        if (!forceRefresh && draft.value && !isCacheStale()) {
            return draft.value;
        }

        try {
            const res = await api.draft.getDraft();
            const nextDraft = normalizeDraft(res.data?.data ?? undefined);
            cacheDraft(nextDraft);
            return nextDraft;
        } catch (err: unknown) {
            if (
                typeof err === "object" &&
                err !== null &&
                "status" in err &&
                (err as { status: number }).status === 404
            ) {
                draft.value = null;
                cacheTimestamp.value = Date.now();
                return null;
            }
            throw err;
        }
    };

    const deleteDraft = async () => {
        await pendingSave.value;
        await api.draft.deleteDraft();
        draft.value = null;
        cacheTimestamp.value = Date.now();
        lastSavedAt.value = null;
    };

    const reset = () => {
        draft.value = null;
        cacheTimestamp.value = null;
        saving.value = false;
        lastSavedAt.value = null;
        pendingSave.value = null;
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "draft",
        reset,
        pinia,
    );
    safeOnScopeDispose(unregisterSessionReset);

    return {
        draft,
        hasDraft,
        saving,
        lastSavedAt,
        saveDraft,
        loadDraft,
        deleteDraft,
        reset,
    };
});
