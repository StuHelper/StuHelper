<template>
    <div
        class="bg-bg-card rounded-xl shadow-card p-5 flex flex-col gap-4"
    >
        <input
            v-model="title"
            class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
            :placeholder="t('review.post.title')"
            :aria-label="t('review.post.titleLabel')"
            :maxlength="TITLE_MAX"
        />
        <span class="block text-right text-xs text-text-muted -mt-2">
            {{
                t("review.validation.charCount", {
                    current: title.length,
                    max: TITLE_MAX,
                })
            }}
        </span>

        <div>
            <label class="block text-sm text-text-secondary mb-1.5">{{ t('review.post.termLabel') }} <span class="text-danger text-xs">*</span></label>
            <select
                v-model="termID"
                class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
            >
                <option value="" disabled>{{ t('review.post.termPlaceholder') }}</option>
                <option v-for="term in termOptions" :key="term.id" :value="term.id">
                    {{ term.name }}
                </option>
            </select>
            <span v-if="submitting && !termID.trim()" class="block text-xs text-danger mt-1">{{ t('review.post.termMissing') }}</span>
        </div>

        <RatingGroup
            v-model="ratings"
            :dimensions="ratingDimensions"
            :loading="ratingDimensionsLoading"
            :load-failed="ratingDimensionsLoadFailed"
        />

        <textarea
            v-model="content"
            class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[100px] transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
            :placeholder="t('review.post.contentPlaceholder')"
            :aria-label="t('review.post.contentLabel')"
            :aria-describedby="
                contentError ? 'review-form-content-error' : undefined
            "
            :maxlength="CONTENT_MAX"
            rows="4"
        />
        <span
            class="block text-right text-xs text-text-muted -mt-2"
            :class="{
                '!text-danger':
                    content.length < CONTENT_MIN && content.length > 0,
            }"
        >
            {{
                t("review.validation.charCount", {
                    current: content.length,
                    max: CONTENT_MAX,
                })
            }}
        </span>
        <span
            v-if="contentError"
            id="review-form-content-error"
            class="block text-xs text-danger -mt-2"
            >{{ contentError }}</span
        >

        <div class="flex justify-end">
            <button
                class="py-2 px-5 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-full cursor-pointer transition-all duration-fast hover:not-disabled:opacity-90 hover:not-disabled:-translate-y-px disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!canSubmit || submitting"
                @click="handleSubmit"
            >
                {{
                    submitting
                        ? t("common.actions.loading")
                        : t("review.post.submit")
                }}
            </button>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { api } from "@/api";
import { useToast } from "@/composables/useToast";
import {
    REVIEW_TITLE_MAX_LENGTH,
    REVIEW_CONTENT_MIN_LENGTH,
    REVIEW_CONTENT_MAX_LENGTH,
} from "@/constants/review";
import RatingGroup from "./RatingGroup.vue";
import type { ReviewRatings } from "@/types/review";
import type { Term } from "@/types/course";
import { buildCreateReviewPayload } from "./reviewPayload";
import { buildTermOptions } from "@/modules/course/termOptions";
import { areRatingsComplete, useRatingDimensions } from "./composables/useRatingDimensions";

const props = defineProps<{
    courseID: number;
}>();

const emit = defineEmits<{ posted: [] }>();

const { t } = useI18n();
const toast = useToast();

const TITLE_MAX = REVIEW_TITLE_MAX_LENGTH;
const CONTENT_MIN = REVIEW_CONTENT_MIN_LENGTH;
const CONTENT_MAX = REVIEW_CONTENT_MAX_LENGTH;
const {
    dimensions: ratingDimensions,
    loading: ratingDimensionsLoading,
    loadFailed: ratingDimensionsLoadFailed,
} = useRatingDimensions();

const title = ref("");
const content = ref("");
const ratings = ref<ReviewRatings>({});
const submitting = ref(false);
const terms = ref<Term[]>([]);
const termID = ref("");
const trimmedTitleLength = computed(() => title.value.trim().length);
const termOptions = computed(() => buildTermOptions(terms.value));

const contentError = computed(() => {
    const len = content.value.trim().length;
    if (len > 0 && len < CONTENT_MIN) {
        return t("review.validation.contentTooShort", { min: CONTENT_MIN });
    }
    return "";
});

const canSubmit = computed(
    () =>
        termID.value.trim().length > 0 &&
        trimmedTitleLength.value > 0 &&
        content.value.trim().length >= CONTENT_MIN &&
        content.value.length <= CONTENT_MAX &&
        trimmedTitleLength.value <= TITLE_MAX &&
        !ratingDimensionsLoading.value &&
        !ratingDimensionsLoadFailed.value &&
        areRatingsComplete(ratings.value, ratingDimensions.value),
);

async function fetchTerms() {
    try {
        const res = await api.course.getTerms();
        terms.value = res.data?.data || [];
        if (!termID.value && terms.value.length > 0) {
            const options = buildTermOptions(terms.value);
            termID.value = options[0]?.id || "";
        }
    } catch {
        terms.value = [];
    }
}

function reset() {
    termID.value = terms.value.length > 0 ? buildTermOptions(terms.value)[0]?.id || "" : "";
    title.value = "";
    content.value = "";
    ratings.value = {};
}

onMounted(() => {
    void fetchTerms();
});

async function handleSubmit() {
    if (ratingDimensionsLoadFailed.value) {
        toast.error(t("review.post.ratingLoadFailed"));
        return;
    }
    if (!canSubmit.value) return;
    submitting.value = true;
    try {
        await api.review.createReview(
            buildCreateReviewPayload({
                courseID: props.courseID,
                termID: termID.value,
                title: title.value.trim(),
                content: content.value.trim(),
                ratings: ratings.value,
            }),
        );
        toast.success(t("review.post.success"));
        reset();
        emit("posted");
    } catch {
        toast.error(t("review.post.failed"));
    } finally {
        submitting.value = false;
    }
}
</script>
