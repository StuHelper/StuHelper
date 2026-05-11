<template>
    <div v-if="!selectedCourse">
        <input
            :value="query"
            autocomplete="off"
            data-1p-ignore
            data-lpignore="true"
            data-form-type="other"
            role="combobox"
            :aria-expanded="results.length > 0"
            aria-haspopup="listbox"
            aria-autocomplete="list"
            :aria-activedescendant="
                highlightedIndex >= 0
                    ? `course-option-${results[highlightedIndex]?.id}`
                    : undefined
            "
            aria-controls="course-search-listbox"
            class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans focus-ring-field"
            :placeholder="t('review.post.searchCourse')"
            :aria-label="t('review.post.searchCourseLabel')"
            @input="onInput"
            @keydown="handleSearchKeydown"
        />
        <div
            v-if="results.length > 0"
            id="course-search-listbox"
            role="listbox"
            class="border-0 rounded-lg max-h-[200px] overflow-y-auto mt-2"
        >
            <button
                v-for="(course, idx) in results"
                :id="`course-option-${course.id}`"
                :key="course.id"
                role="option"
                :aria-selected="idx === highlightedIndex"
                class="flex items-center gap-2 w-full p-3 text-left text-sm text-text-primary cursor-pointer transition-[background] duration-fast hover:bg-bg-hover"
                :class="{ 'bg-bg-hover': idx === highlightedIndex }"
                @click="emit('select', course)"
            >
                <span class="font-medium truncate">{{ course.name }}</span>
                <span class="shrink-0 text-xs text-text-muted"
                    ><template v-if="course.credits"
                        >{{
                            t("review.course.creditsBadge", {
                                n: course.credits,
                            })
                        }}
                        · </template
                    >{{ course.departmentName }}</span
                >
                <span
                    class="shrink-0 text-xs tabular-nums text-text-muted ml-auto"
                    >{{
                        t("review.course.reviewCountBadge", {
                            count: course.reviewCount ?? 0,
                        })
                    }}</span
                >
            </button>
        </div>
    </div>

    <div
        v-else
        class="flex items-center justify-between p-3 bg-primary/[0.06] rounded-lg border border-primary/15"
    >
        <span class="font-semibold text-sm">{{ selectedCourse.name }}</span>
        <button
            class="text-xs text-primary cursor-pointer"
            :aria-label="t('common.actions.edit')"
            @click="emit('clearSelection')"
        >
            {{ t("common.actions.edit") }}
        </button>
    </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { Course } from "@stuhelper/shared/course";

const props = defineProps<{
    query: string;
    results: Course[];
    selectedCourse: Course | null;
}>();

const emit = defineEmits<{
    "update:query": [value: string];
    select: [course: Course];
    clearSelection: [];
}>();

const { t } = useI18n();
const highlightedIndex = ref(-1);

watch(
    () => props.results,
    () => {
        highlightedIndex.value = -1;
    },
);

function onInput(event: Event) {
    emit("update:query", (event.target as HTMLInputElement).value);
}

function handleSearchKeydown(event: KeyboardEvent) {
    const len = props.results.length;
    if (len === 0) return;

    switch (event.key) {
        case "ArrowDown":
            event.preventDefault();
            highlightedIndex.value = (highlightedIndex.value + 1) % len;
            break;
        case "ArrowUp":
            event.preventDefault();
            highlightedIndex.value = (highlightedIndex.value - 1 + len) % len;
            break;
        case "Enter":
            event.preventDefault();
            if (highlightedIndex.value >= 0 && highlightedIndex.value < len) {
                emit("select", props.results[highlightedIndex.value]);
            }
            break;
    }
}
</script>
