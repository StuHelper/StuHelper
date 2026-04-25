<template>
    <Teleport to="body">
        <Transition name="overlay">
            <div
                v-if="visible"
                class="fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] flex items-center justify-center p-4"
                @click.self="handleCancel"
            >
                <div
                    ref="modalRef"
                    class="relative w-full max-w-[660px] max-h-[85vh] bg-bg-card rounded-xl shadow-xl flex flex-col overflow-hidden animate-modal-in"
                    role="dialog"
                    aria-modal="true"
                    aria-labelledby="review-dialog-title"
                    tabindex="-1"
                    @keydown.esc="handleCancel"
                    @keydown="trapFocus"
                >
                    <div
                        class="flex items-center justify-between p-5 border-b border-border-light"
                    >
                        <h2
                            id="review-dialog-title"
                            class="text-lg font-bold tracking-tight"
                        >
                            {{ t("review.hub.postReview") }}
                        </h2>
                        <button
                            class="text-2xl text-text-muted leading-none cursor-pointer w-8 h-8 flex items-center justify-center rounded-full transition-all duration-fast hover:text-text-primary hover:bg-bg-secondary"
                            :aria-label="t('common.actions.close')"
                            @click="$emit('close')"
                        >
                            &times;
                        </button>
                    </div>

                    <div class="flex-1 overflow-y-auto p-5 flex flex-col gap-4">
                        <ReviewCourseSelector
                            :query="courseQuery"
                            :results="courseResults"
                            :selected-course="selectedCourse"
                            @update:query="courseQuery = $event"
                            @select="selectCourse"
                            @clear-selection="selectedCourse = null"
                        />

                        <!-- 表单 -->
                        <template v-if="selectedCourse">
                            <ReviewTeacherSelect
                                v-model="teacherQuery"
                                v-model:selected-teacher-id="selectedTeacherID"
                                :teachers="teacherList"
                                :loading="loadingTeachers"
                            />

                            <div class="relative">
                                <label
                                    for="review-term-select"
                                    class="font-medium text-text-primary text-sm mb-1.5 block"
                                    >{{ t("review.post.termLabel") }}
                                    <span class="text-danger text-xs"
                                        >*</span
                                    ></label
                                >
                                <select
                                    id="review-term-select"
                                    v-model="selectedTermID"
                                    class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans focus-ring-field"
                                    :class="
                                        attempted && !selectedTermID
                                            ? 'border-danger'
                                            : 'border-border'
                                    "
                                >
                                    <option value="" disabled>
                                        {{ t("review.post.termPlaceholder") }}
                                    </option>
                                    <option
                                        v-for="term in termOptions"
                                        :key="term.id"
                                        :value="term.id"
                                    >
                                        {{ term.name }}
                                    </option>
                                </select>
                                <span
                                    v-if="attempted && !selectedTermID"
                                    class="block text-xs text-danger mt-1"
                                    >{{ t("review.post.termMissing") }}</span
                                >
                            </div>

                            <div class="relative">
                                <label
                                    for="review-title-input"
                                    class="font-medium text-text-primary text-sm mb-1.5 block"
                                    >{{ t("review.post.titleRequired") }}
                                    <span class="text-danger text-xs"
                                        >*</span
                                    ></label
                                >
                                <input
                                    id="review-title-input"
                                    v-model="title"
                                    autocomplete="off"
                                    class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans focus-ring-field"
                                    :class="
                                        attempted && titleInvalid
                                            ? 'border-danger'
                                            : 'border-border'
                                    "
                                    :placeholder="
                                        t('review.post.titlePlaceholder')
                                    "
                                    :maxlength="TITLE_MAX"
                                />
                                <span
                                    v-if="attempted && titleInvalid"
                                    class="block text-xs text-danger mt-1"
                                    >{{ t("review.post.titleMissing") }}</span
                                >
                                <span
                                    v-else
                                    class="block text-right text-xs text-text-muted mt-1"
                                >
                                    {{
                                        t("review.validation.charCount", {
                                            current: title.length,
                                            max: TITLE_MAX,
                                        })
                                    }}
                                </span>
                            </div>

                            <div
                                :class="{
                                    'ring-1 ring-danger rounded-lg':
                                        attempted && ratingsInvalid,
                                }"
                            >
                                <RatingGroup
                                    v-model="ratings"
                                    :dimensions="ratingDimensions"
                                    :loading="ratingDimensionsLoading"
                                    :load-failed="ratingDimensionsLoadFailed"
                                />
                            </div>
                            <span
                                v-if="attempted && ratingsInvalid"
                                class="block text-xs text-danger -mt-2"
                            >
                                {{
                                    ratingDimensionsLoadFailed
                                        ? t("review.post.ratingLoadFailed")
                                        : t("review.post.ratingMissing")
                                }}
                            </span>

                            <div class="relative">
                                <label
                                    for="review-content-input"
                                    class="font-medium text-text-primary text-sm mb-1.5 block"
                                    >{{ t("review.post.detailedReview") }}
                                    <span class="text-danger text-xs"
                                        >*</span
                                    ></label
                                >
                                <textarea
                                    id="review-content-input"
                                    v-model="content"
                                    class="w-full p-3 bg-bg-secondary border rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[120px] focus-ring-field"
                                    :class="
                                        attempted && contentInvalid
                                            ? 'border-danger'
                                            : 'border-border'
                                    "
                                    :placeholder="
                                        t('review.post.contentPlaceholder')
                                    "
                                    :aria-describedby="
                                        contentError
                                            ? 'review-dialog-content-error'
                                            : undefined
                                    "
                                    :maxlength="CONTENT_MAX"
                                    rows="5"
                                />
                                <span
                                    v-if="
                                        contentError ||
                                        (attempted && contentInvalid)
                                    "
                                    id="review-dialog-content-error"
                                    class="block text-xs text-danger mt-1"
                                >
                                    {{
                                        contentError ||
                                        t("review.post.contentMinError", {
                                            min: CONTENT_MIN,
                                        })
                                    }}
                                </span>
                            </div>

                            <div class="relative">
                                <label
                                    for="review-grade-input"
                                    class="font-medium text-text-primary text-sm mb-1.5 block"
                                    >{{ t("review.post.gradeLabel") }}
                                    <span
                                        class="text-text-muted font-normal text-xs"
                                        >（{{
                                            t("review.post.gradeOptional")
                                        }}）</span
                                    ></label
                                >
                                <input
                                    id="review-grade-input"
                                    v-model="grade"
                                    autocomplete="off"
                                    class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans focus-ring-field"
                                    :placeholder="
                                        t('review.post.gradePlaceholder')
                                    "
                                    maxlength="20"
                                />
                            </div>
                        </template>
                    </div>

                    <div
                        v-if="selectedCourse"
                        class="flex justify-end gap-3 px-5 py-4 border-t border-border-light"
                    >
                        <button
                            class="py-2 px-4 text-sm text-text-secondary rounded-full cursor-pointer"
                            @click="handleCancel"
                        >
                            {{ t("common.actions.cancel") }}
                        </button>
                        <button
                            class="py-2 px-5 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-full cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                            :disabled="submitting"
                            @click="handleSubmit"
                        >
                            {{
                                submitting
                                    ? t("common.actions.loading")
                                    : t("review.post.submit")
                            }}
                        </button>
                    </div>

                    <ReviewRedirectOverlay :countdown="redirectCountdown" />
                </div>
            </div>
        </Transition>

        <ReviewExitConfirmDialog
            :visible="showCancelConfirm"
            :saving="savingDraft"
            @close="showCancelConfirm = false"
            @discard="confirmDiscard"
            @save="confirmSaveDraft"
        />
    </Teleport>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import RatingGroup from "./RatingGroup.vue";
import ReviewCourseSelector from "./ReviewCourseSelector.vue";
import ReviewTeacherSelect from "./ReviewTeacherSelect.vue";
import ReviewRedirectOverlay from "./ReviewRedirectOverlay.vue";
import ReviewExitConfirmDialog from "./ReviewExitConfirmDialog.vue";
import { useReviewDialogController } from "./useReviewDialogController";

const props = defineProps<{ visible: boolean }>();
const emit = defineEmits<{ close: []; posted: [] }>();
const modalRef = ref<HTMLElement | null>(null);
const { t } = useI18n();

const {
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
} = useReviewDialogController(
    props,
    {
        close: () => emit("close"),
        posted: () => emit("posted"),
    },
    modalRef,
);
</script>

<style scoped>
/* Vue Transition hooks — 无法用 utility 表达 */
.overlay-enter-active {
    animation: overlayIn var(--duration-base) var(--ease-out);
}
.overlay-leave-active {
    animation: overlayIn var(--duration-fast) var(--ease-out) reverse;
}
</style>
