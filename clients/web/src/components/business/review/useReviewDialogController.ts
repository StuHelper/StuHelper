import { computed, nextTick, onUnmounted, ref, watch, type Ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { api } from "@/api";
import { useToast } from "@/composables/useToast";
import { useReviewPost } from "@/composables/useReviewPost";
import { useAuthStore } from "@/stores/auth";
import {
    REVIEW_CONTENT_MAX_LENGTH,
    REVIEW_CONTENT_MIN_LENGTH,
    REVIEW_TITLE_MAX_LENGTH,
} from "@stuhelper/shared/constants";
import type { Course, TeacherStats, Term } from "@stuhelper/shared/course";
import type { ReviewRatings } from "@stuhelper/shared/review";
import { buildTermOptions } from "@/modules/course/termOptions";
import { buildCreateReviewPayload } from "./reviewPayload";
import {
    areRatingsComplete,
    filterRatingsByDimensions,
    useRatingDimensions,
} from "./composables/useRatingDimensions";
import {
    clearLocalReviewDraft,
    createDraftCourse,
    createLocalReviewDraft,
    getLocalReviewDraftClearedAt,
    loadLocalReviewDraft,
    sanitizeDraftRatings,
    saveLocalReviewDraft,
    type ReviewDialogServerDraft,
} from "./reviewDialogDraftState";

export function useReviewDialogController(
    props: { visible: boolean },
    callbacks: { close: () => void; posted: () => void },
    modalRef: Ref<HTMLElement | null>,
) {
    const { t } = useI18n();
    const toast = useToast();
    const router = useRouter();
    const authStore = useAuthStore();
    const { preselectedCourse } = useReviewPost();

    const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH;
    const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH;
    const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH;
    const AUTO_SAVE_DEBOUNCE_MS = 300;
    const GRADE_PATTERN = /^[A-Za-z0-9+\-./\s]*$/;
    const {
        dimensions: ratingDimensions,
        ratingKeySet,
        loading: ratingDimensionsLoading,
        loadFailed: ratingDimensionsLoadFailed,
    } = useRatingDimensions();

    const templateLabels = computed(() => [
        t("review.post.templateListening"),
        t("review.post.templateWorkload"),
        t("review.post.templateExam"),
    ]);
    const contentTemplate = computed(() => templateLabels.value.join("\n"));

    let userHasEditedContent = false;

    watch(contentTemplate, (newTpl, oldTpl) => {
        if (
            props.visible &&
            !userHasEditedContent &&
            content.value === oldTpl
        ) {
            content.value = newTpl;
        }
    });

    const courseQuery = ref("");
    const courseResults = ref<Course[]>([]);
    const selectedCourse = ref<Course | null>(null);
    const title = ref("");
    const content = ref("");
    const grade = ref("");
    const ratings = ref<ReviewRatings>({});
    const submitting = ref(false);
    const attempted = ref(false);
    const redirectCountdown = ref(0);
    const showCancelConfirm = ref(false);
    const savingDraft = ref(false);
    const teacherList = ref<TeacherStats[]>([]);
    const selectedTeacherID = ref<number | null>(null);
    const teacherQuery = ref("");
    const loadingTeachers = ref(false);
    const terms = ref<Term[]>([]);
    const selectedTermID = ref("");
    const termOptions = computed(() => buildTermOptions(terms.value));

    let searchTimer: ReturnType<typeof setTimeout> | null = null;
    let courseSearchController: AbortController | undefined;
    let countdownTimer: ReturnType<typeof setInterval> | null = null;
    let autoSaveTimer: ReturnType<typeof setTimeout> | null = null;
    let restoreVersion = 0;
    let unloadController: AbortController | null = null;

    watch(ratingKeySet, (keys) => {
        if (keys.size === 0) return;
        ratings.value = filterRatingsByDimensions(ratings.value, keys);
    });

    function saveLocalDraft() {
        if (!selectedCourse.value) return;
        saveLocalReviewDraft(
            createLocalReviewDraft({
                course: selectedCourse.value,
                teacherID: selectedTeacherID.value,
                teacherName: teacherQuery.value.trim() || undefined,
                title: title.value,
                content: content.value,
                grade: grade.value,
                termID: selectedTermID.value,
                ratings: ratings.value,
            }),
        );
    }

    function getUserContentLength(raw: string): number {
        let text = raw;
        for (const label of templateLabels.value) {
            text = text.split(label).join("");
        }
        return text.trim().length;
    }

    function hasFormContent(): boolean {
        return (
            title.value.trim().length > 0 ||
            getUserContentLength(content.value) > 0 ||
            grade.value.trim().length > 0 ||
            Object.keys(ratings.value).length > 0
        );
    }

    async function saveDraftAuto() {
        if (!selectedCourse.value) return;
        if (authStore.isAuthenticated) {
            try {
                await api.draft.saveDraft({
                    courseID: selectedCourse.value.id,
                    teacherID: selectedTeacherID.value ?? undefined,
                    termID: selectedTermID.value || undefined,
                    title: title.value.trim() || undefined,
                    content: content.value.trim() || undefined,
                    grade: grade.value?.trim() || undefined,
                    ratings:
                        Object.keys(ratings.value).length > 0
                            ? ratings.value
                            : undefined,
                });
            } catch (_error) { void _error;
                saveLocalDraft();
            }
        } else {
            saveLocalDraft();
        }
    }

    function trapFocus(event: KeyboardEvent) {
        if (event.key !== "Tab" || !modalRef.value) return;
        const focusable = modalRef.value.querySelectorAll<HTMLElement>(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
        );
        if (focusable.length === 0) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (event.shiftKey) {
            if (document.activeElement === first) {
                event.preventDefault();
                last.focus();
            }
        } else if (document.activeElement === last) {
            event.preventDefault();
            first.focus();
        }
    }

    watch(courseQuery, (val) => {
        if (searchTimer) clearTimeout(searchTimer);
        const q = val.trim();
        if (!q) {
            if (courseSearchController) {
                courseSearchController.abort();
                courseSearchController = undefined;
            }
            courseResults.value = [];
            return;
        }

        searchTimer = setTimeout(async () => {
            if (courseSearchController) courseSearchController.abort();
            const controller = new AbortController();
            courseSearchController = controller;

            try {
                const res = await api.course.searchCourses(q, 8, {
                    signal: controller.signal,
                });
                if (controller.signal.aborted) return;
                const list = res.data?.data?.list || [];
                const seen = new Set<number>();
                courseResults.value = list.filter((course: Course) => {
                    if (seen.has(course.id)) return false;
                    seen.add(course.id);
                    return true;
                });
            } catch (err) {
                if (controller.signal.aborted) return;
                if (import.meta.env.DEV) {
                    console.warn("[ReviewDialog] Course search failed:", err);
                }
                courseResults.value = [];
            }
        }, 300);
    });

    function onBeforeUnload(event: BeforeUnloadEvent) {
        if (hasFormContent()) {
            event.preventDefault();
        }
    }

    function clearCountdownTimer() {
        if (countdownTimer) {
            clearInterval(countdownTimer);
            countdownTimer = null;
        }
        redirectCountdown.value = 0;
    }

    function startRedirectCountdown() {
        clearCountdownTimer();
        redirectCountdown.value = 3;
        countdownTimer = setInterval(() => {
            redirectCountdown.value--;
            if (redirectCountdown.value <= 0) {
                clearCountdownTimer();
                callbacks.close();
                authStore.login();
            }
        }, 1000);
    }

    watch(
        [title, content, grade, ratings],
        () => {
            userHasEditedContent = true;
            if (!selectedCourse.value) return;
            if (autoSaveTimer) clearTimeout(autoSaveTimer);
            autoSaveTimer = setTimeout(() => {
                saveLocalDraft();
            }, AUTO_SAVE_DEBOUNCE_MS);
        },
        { deep: true },
    );

    const gradeInvalid = computed(() => {
        const normalizedGrade = grade.value.trim();
        return (
            normalizedGrade.length > 0 && !GRADE_PATTERN.test(normalizedGrade)
        );
    });

    async function loadTeachers(courseID: number) {
        loadingTeachers.value = true;
        teacherList.value = [];
        selectedTeacherID.value = null;
        teacherQuery.value = "";
        try {
            const res = await api.rating.getCourseTeachers(courseID);
            teacherList.value = res.data?.data ?? [];
        } catch (_error) { void _error;
            // noop
        } finally {
            loadingTeachers.value = false;
        }
    }

    async function loadTerms() {
        try {
            const res = await api.course.getTerms();
            terms.value = res.data?.data ?? [];
            if (!selectedTermID.value && terms.value.length > 0) {
                selectedTermID.value =
                    buildTermOptions(terms.value)[0]?.id || "";
            }
        } catch (_error) { void _error;
            terms.value = [];
            selectedTermID.value = "";
        }
    }

    function applyTeacherDraftSelection(
        teacherID?: number | null,
        teacherName?: string,
    ) {
        if (teacherID === undefined || teacherID === null) {
            selectedTeacherID.value = null;
            teacherQuery.value = teacherName ?? "";
            return;
        }

        selectedTeacherID.value = teacherID;
        const matched = teacherList.value.find(
            (teacher) => teacher.teacherID === teacherID,
        );
        teacherQuery.value = matched?.teacherName ?? teacherName ?? "";
    }

    async function tryRestoreDraft(expectedVersion: number) {
        const local = loadLocalReviewDraft(
            ratingKeySet.value.size > 0 ? ratingKeySet.value : undefined,
            selectedCourse.value?.id,
        );
        const clearedAt = getLocalReviewDraftClearedAt(
            local?.courseID ?? selectedCourse.value?.id,
        );
        let serverDraft: ReviewDialogServerDraft | null = null;

        if (authStore.isAuthenticated && (selectedCourse.value || local)) {
            const courseID = selectedCourse.value?.id ?? local?.courseID;
            if (courseID) {
                try {
                    const res = await api.draft.getDraft(courseID);
                    serverDraft = res.data?.data ?? null;
                } catch (_error) { void _error;
                    // noop
                }
            }
        }

        if (restoreVersion !== expectedVersion) return;

        const localTime = local?.updatedAt ?? 0;
        const serverTime = serverDraft
            ? new Date(serverDraft.updatedAt).getTime()
            : 0;
        const latest = Math.max(localTime, serverTime, clearedAt);
        if (latest === 0 || latest === clearedAt) return;

        if (latest === localTime && local) {
            if (
                !selectedCourse.value ||
                selectedCourse.value.id !== local.courseID
            ) {
                selectedCourse.value = createDraftCourse(local);
                await Promise.all([loadTeachers(local.courseID), loadTerms()]);
            }
            if (local.title) title.value = local.title;
            if (local.content) content.value = local.content;
            if (local.grade) grade.value = local.grade;
            selectedTermID.value = local.termID;
            if (local.ratings && Object.keys(local.ratings).length > 0) {
                ratings.value = local.ratings;
            }
            applyTeacherDraftSelection(
                local.teacherID ?? null,
                local.teacherName,
            );
        } else if (serverDraft) {
            if (serverDraft.title) title.value = serverDraft.title;
            if (serverDraft.content) content.value = serverDraft.content;
            if (serverDraft.grade) grade.value = serverDraft.grade;
            if (serverDraft.termID) selectedTermID.value = serverDraft.termID;
            applyTeacherDraftSelection(serverDraft.teacherID ?? null);
            const sanitizedRatings = sanitizeDraftRatings(
                serverDraft.ratings,
                ratingKeySet.value.size > 0 ? ratingKeySet.value : undefined,
            );
            if (sanitizedRatings && Object.keys(sanitizedRatings).length > 0) {
                ratings.value = sanitizedRatings;
            }
        }

        toast.info(t("review.draft.restored"));
    }

    const titleInvalid = computed(() => title.value.trim().length === 0);
    const ratingsInvalid = computed(() => {
        if (ratingDimensionsLoading.value || ratingDimensionsLoadFailed.value) {
            return true;
        }
        return !areRatingsComplete(ratings.value, ratingDimensions.value);
    });
    const contentInvalid = computed(
        () => getUserContentLength(content.value) < CONTENT_MIN,
    );
    const contentError = computed(() => {
        const userLen = getUserContentLength(content.value);
        if (userLen > 0 && userLen < CONTENT_MIN) {
            return t("review.post.contentMinErrorNoTemplate", {
                min: CONTENT_MIN,
            });
        }
        return "";
    });
    const canSubmit = computed(() => {
        return (
            selectedCourse.value &&
            !titleInvalid.value &&
            !contentInvalid.value &&
            !gradeInvalid.value &&
            !ratingDimensionsLoading.value &&
            !ratingDimensionsLoadFailed.value &&
            content.value.length <= CONTENT_MAX &&
            title.value.length <= TITLE_MAX &&
            !ratingsInvalid.value
        );
    });

    watch(
        () => props.visible,
        async (val) => {
            if (val) {
                document.body.style.overflow = "hidden";
                unloadController?.abort();
                unloadController = new AbortController();
                window.addEventListener("beforeunload", onBeforeUnload, {
                    signal: unloadController.signal,
                });
                courseQuery.value = "";
                courseResults.value = [];
                selectedCourse.value = preselectedCourse.value ?? null;
                title.value = "";
                content.value = contentTemplate.value;
                grade.value = "";
                ratings.value = {};
                selectedTermID.value = "";
                selectedTeacherID.value = null;
                teacherQuery.value = "";
                submitting.value = false;
                attempted.value = false;
                showCancelConfirm.value = false;
                userHasEditedContent = false;
                if (selectedCourse.value) {
                    await Promise.all([
                        loadTeachers(selectedCourse.value.id),
                        loadTerms(),
                    ]);
                } else {
                    teacherList.value = [];
                    terms.value = [];
                }
                const currentVersion = ++restoreVersion;
                await nextTick();
                await tryRestoreDraft(currentVersion);
                nextTick(() => {
                    const firstInput =
                        modalRef.value?.querySelector<HTMLElement>(
                            "input, textarea",
                        );
                    if (firstInput) firstInput.focus();
                    else modalRef.value?.focus();
                });
            } else {
                document.body.style.overflow = "";
                if (searchTimer) clearTimeout(searchTimer);
                if (autoSaveTimer) clearTimeout(autoSaveTimer);
                clearCountdownTimer();
                unloadController?.abort();
                unloadController = null;
            }
        },
    );

    onUnmounted(() => {
        document.body.style.overflow = "";
        if (searchTimer) clearTimeout(searchTimer);
        if (courseSearchController) {
            courseSearchController.abort();
            courseSearchController = undefined;
        }
        if (autoSaveTimer) clearTimeout(autoSaveTimer);
        clearCountdownTimer();
        unloadController?.abort();
        unloadController = null;
    });

    function selectCourse(course: Course) {
        selectedCourse.value = course;
        courseQuery.value = "";
        courseResults.value = [];
        void loadTeachers(course.id);
        void loadTerms();
    }

    async function handleSubmit() {
        if (submitting.value) return;
        attempted.value = true;
        if (ratingDimensionsLoadFailed.value) {
            toast.error(t("review.post.ratingLoadFailed"));
            return;
        }
        if (!canSubmit.value || !selectedCourse.value) return;

        if (!authStore.isAuthenticated) {
            saveLocalDraft();
            sessionStorage.setItem(
                "draft_redirect",
                router.currentRoute.value.fullPath,
            );
            sessionStorage.setItem("draft_pending", "1");
            startRedirectCountdown();
            return;
        }

        submitting.value = true;
        try {
            const checkRes = await api.review.checkContent({
                content: content.value.trim(),
            });
            const checkResult = checkRes.data?.data;
            if (checkResult && !checkResult.isValid) {
                if (checkResult.level === "block") {
                    toast.error(t("review.post.contentBlocked"));
                    return;
                }
                if (checkResult.level === "warn") {
                    toast.warning(t("review.post.contentWarning"));
                }
            }

            await api.review.createReview(
                buildCreateReviewPayload({
                    courseID: selectedCourse.value.id,
                    teacherID: selectedTeacherID.value ?? undefined,
                    termID: selectedTermID.value,
                    title: title.value.trim(),
                    content: content.value.trim(),
                    grade: grade.value?.trim() || undefined,
                    ratings: ratings.value,
                }),
            );
            clearLocalReviewDraft(selectedCourse.value.id);
            try {
                await api.draft.deleteDraft(selectedCourse.value.id);
            } catch (_error) { void _error;
                // noop
            }
            toast.success(t("review.post.success"));
            callbacks.posted();
            callbacks.close();
        } catch (_error) { void _error;
            toast.error(t("review.post.failed"));
        } finally {
            submitting.value = false;
        }
    }

    function handleCancel() {
        if (redirectCountdown.value > 0) return;
        if (selectedCourse.value && hasFormContent()) {
            showCancelConfirm.value = true;
            return;
        }
        callbacks.close();
    }

    async function confirmSaveDraft() {
        savingDraft.value = true;
        try {
            await saveDraftAuto();
            toast.success(t("review.draft.saved"));
            showCancelConfirm.value = false;
            callbacks.close();
        } catch (_error) { void _error;
            toast.error(t("review.draft.saveFailed"));
            showCancelConfirm.value = false;
        } finally {
            savingDraft.value = false;
        }
    }

    function confirmDiscard() {
        if (selectedCourse.value) {
            clearLocalReviewDraft(selectedCourse.value.id);
        } else {
            clearLocalReviewDraft();
        }
        if (authStore.isAuthenticated && selectedCourse.value) {
            void api.draft.deleteDraft(selectedCourse.value.id).catch(() => {});
        }
        showCancelConfirm.value = false;
        callbacks.close();
    }

    return {
        TITLE_MAX,
        CONTENT_MIN,
        CONTENT_MAX,
        ratingDimensions,
        ratingDimensionsLoading,
        ratingDimensionsLoadFailed,
        courseQuery,
        courseResults,
        selectedCourse,
        title,
        content,
        grade,
        ratings,
        submitting,
        attempted,
        redirectCountdown,
        showCancelConfirm,
        savingDraft,
        teacherList,
        selectedTeacherID,
        teacherQuery,
        loadingTeachers,
        selectedTermID,
        termOptions,
        titleInvalid,
        ratingsInvalid,
        contentInvalid,
        contentError,
        trapFocus,
        selectCourse,
        handleSubmit,
        handleCancel,
        confirmSaveDraft,
        confirmDiscard,
    };
}
