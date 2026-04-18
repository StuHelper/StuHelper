/**
 * 草稿状态管理
 */
import { defineStore, getActivePinia } from "pinia";
import { onScopeDispose, ref } from "vue";
import type { Draft, SaveDraftParams } from "@stuhelper/shared/draft";
import { isValidRating } from "@stuhelper/shared/course";
import { api } from "@/api";
import { registerSessionResetHandler } from "@/stores/sessionOrchestrator";

// 草稿缓存最大条目数
const MAX_DRAFT_ENTRIES = 100;

export const useDraftStore = defineStore("draft", () => {
    const pinia = getActivePinia();
    if (!pinia) {
        throw new Error("draft store requires an active Pinia instance");
    }

    // 使用 Record 替代 Map，确保完整的 Pinia 响应式支持
    // reactive Record 的属性变更能正确触发响应式更新
    const drafts = ref<Record<number, Draft>>({});
    // 缓存时间戳（毫秒）
    const cacheTimestamps = new Map<number, number>();
    // 缓存 TTL：5 分钟
    const CACHE_TTL_MS = 5 * 60 * 1000;

    // 保存状态
    const saving = ref(false);
    const lastSavedAt = ref<Date | null>(null);

    function isCacheStale(courseID: number): boolean {
        const ts = cacheTimestamps.get(courseID);
        if (ts == null) return true;
        return Date.now() - ts > CACHE_TTL_MS;
    }

    // 超过上限时清除最旧条目
    function evictIfNeeded() {
        const keys = Object.keys(drafts.value).map(Number);
        if (keys.length <= MAX_DRAFT_ENTRIES) return;
        // 按缓存时间戳排序，清除最旧的
        keys.sort(
            (a, b) =>
                (cacheTimestamps.get(a) ?? 0) - (cacheTimestamps.get(b) ?? 0),
        );
        const toRemove = keys.slice(0, keys.length - MAX_DRAFT_ENTRIES);
        const next = { ...drafts.value };
        for (const key of toRemove) {
            delete next[key];
            cacheTimestamps.delete(key);
        }
        drafts.value = next;
    }

    // 校验草稿字段
    function validateDraftParams(data: SaveDraftParams): boolean {
        if (!data.courseID || !Number.isFinite(data.courseID)) return false;
        if (data.title !== undefined && typeof data.title !== "string")
            return false;
        if (data.content !== undefined && typeof data.content !== "string")
            return false;
        return true;
    }

    function normalizeDraft(
        draft:
            | {
                  id: string;
                  courseID: number;
                  teacherID?: number | null;
                  termID?: string;
                  title?: string;
                  content?: string;
                  grade?: string;
                  ratings?: Record<string, number>;
                  updatedAt: string;
              }
            | undefined,
    ): Draft | undefined {
        if (!draft) return undefined;

        let ratings: Draft["ratings"] = undefined;
        if (draft.ratings) {
            const normalizedRatings: Record<string, 1 | 2 | 3 | 4 | 5> = {};
            for (const [key, value] of Object.entries(draft.ratings)) {
                if (!isValidRating(value)) return undefined;
                normalizedRatings[key] = value;
            }
            ratings = normalizedRatings;
        }

        return {
            id: draft.id,
            courseID: draft.courseID,
            ...(draft.teacherID !== undefined && {
                teacherID: draft.teacherID,
            }),
            ...(draft.termID !== undefined && { termID: draft.termID }),
            ...(draft.title !== undefined && { title: draft.title }),
            ...(draft.content !== undefined && { content: draft.content }),
            ...(draft.grade !== undefined && {
                grade: draft.grade as Draft["grade"],
            }),
            ...(ratings !== undefined && { ratings }),
            updatedAt: draft.updatedAt,
        };
    }

    // 保存草稿
    const saveDraft = async (data: SaveDraftParams) => {
        if (!validateDraftParams(data)) return undefined;
        saving.value = true;
        try {
            const res = await api.draft.saveDraft(data);
            const draft = normalizeDraft(res.data?.data ?? undefined);
            if (draft) {
                drafts.value = { ...drafts.value, [data.courseID]: draft };
                cacheTimestamps.set(data.courseID, Date.now());
                evictIfNeeded();
                lastSavedAt.value = new Date();
            }
            return draft;
        } finally {
            saving.value = false;
        }
    };

    // 获取草稿
    const loadDraft = async (courseID: number, forceRefresh = false) => {
        if (!courseID || !Number.isFinite(courseID)) return null;
        // 先检查缓存（未过期且非强制刷新）
        if (
            !forceRefresh &&
            courseID in drafts.value &&
            !isCacheStale(courseID)
        ) {
            return drafts.value[courseID];
        }

        try {
            const res = await api.draft.getDraft(courseID);
            const draft = normalizeDraft(res.data?.data ?? undefined);
            if (draft) {
                drafts.value = { ...drafts.value, [courseID]: draft };
                cacheTimestamps.set(courseID, Date.now());
                evictIfNeeded();
            }
            return draft;
        } catch (err: unknown) {
            // 404 = no draft exists, return null silently
            if (
                typeof err === "object" &&
                err !== null &&
                "status" in err &&
                (err as { status: number }).status === 404
            ) {
                return null;
            }
            throw err;
        }
    };

    // 删除草稿
    const deleteDraft = async (courseID: number) => {
        await api.draft.deleteDraft(courseID);
        const next = { ...drafts.value };
        delete next[courseID];
        drafts.value = next;
        cacheTimestamps.delete(courseID);
    };

    // 检查是否有草稿
    const hasDraft = (courseID: number) => {
        return courseID in drafts.value;
    };

    // 获取缓存的草稿
    const getCachedDraft = (courseID: number) => {
        return drafts.value[courseID] ?? undefined;
    };

    // 重置状态（登出时调用，防止用户数据泄漏）
    const reset = () => {
        drafts.value = {};
        cacheTimestamps.clear();
        saving.value = false;
        lastSavedAt.value = null;
    };

    const unregisterSessionReset = registerSessionResetHandler(
        "draft",
        reset,
        pinia,
    );
    onScopeDispose(unregisterSessionReset);

    return {
        drafts,
        saving,
        lastSavedAt,
        saveDraft,
        loadDraft,
        deleteDraft,
        hasDraft,
        getCachedDraft,
        reset,
    };
});
