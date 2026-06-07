/**
 * 草稿状态管理
 */
import { defineStore, getActivePinia } from "pinia";
import { computed, ref } from "vue";
import type { Draft, SaveDraftParams } from "@stuhelper/shared/draft";
import { isValidRating } from "@stuhelper/shared/course";
import { normalizeReviewGrade } from "@stuhelper/shared/constants";
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

    function normalizeDraft(value: unknown): Draft {
        if (!value || typeof value !== "object") {
            throw new Error("invalid draft response");
        }

        const raw = value as Record<string, unknown>;
        if (typeof raw.id !== "string" || raw.id.trim() === "") {
            throw new Error("invalid draft response");
        }
        if (typeof raw.updatedAt !== "string" || raw.updatedAt.trim() === "") {
            throw new Error("invalid draft response");
        }
        if (
            raw.courseID !== undefined &&
            raw.courseID !== null &&
            (typeof raw.courseID !== "number" || !Number.isFinite(raw.courseID))
        ) {
            throw new Error("invalid draft response");
        }
        if (
            raw.teacherID !== undefined &&
            raw.teacherID !== null &&
            (typeof raw.teacherID !== "number" || !Number.isFinite(raw.teacherID))
        ) {
            throw new Error("invalid draft response");
        }
        for (const field of ["termID", "title", "content", "grade"] as const) {
            if (raw[field] !== undefined && typeof raw[field] !== "string") {
                throw new Error("invalid draft response");
            }
        }

        let ratings: Draft["ratings"] = undefined;
        if (raw.ratings !== undefined) {
            if (!raw.ratings || typeof raw.ratings !== "object" || Array.isArray(raw.ratings)) {
                throw new Error("invalid draft response");
            }
            const normalizedRatings: Record<string, 1 | 2 | 3 | 4 | 5> = {};
            for (const [key, rating] of Object.entries(raw.ratings)) {
                if (!isValidRating(rating)) {
                    throw new Error("invalid draft response");
                }
                normalizedRatings[key] = rating;
            }
            ratings = normalizedRatings;
        }
        const grade =
            raw.grade !== undefined ? normalizeReviewGrade(raw.grade as string) : undefined;
        if (raw.grade !== undefined && grade === undefined) {
            throw new Error("invalid draft response");
        }

        return {
            id: raw.id,
            ...(raw.courseID !== undefined && raw.courseID !== null && {
                courseID: raw.courseID as number,
            }),
            ...(raw.teacherID !== undefined && {
                teacherID: raw.teacherID as number | null,
            }),
            ...(raw.termID !== undefined && { termID: raw.termID as string }),
            ...(raw.title !== undefined && { title: raw.title as string }),
            ...(raw.content !== undefined && { content: raw.content as string }),
            ...(grade !== undefined && { grade }),
            ...(ratings !== undefined && { ratings }),
            updatedAt: raw.updatedAt,
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
            const nextDraft = normalizeDraft(res.data?.data);
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
            if (res.data?.data === null) {
                draft.value = null;
                cacheTimestamp.value = Date.now();
                return null;
            }
            const nextDraft = normalizeDraft(res.data?.data);
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
