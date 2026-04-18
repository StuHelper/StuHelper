<template>
    <div class="relative">
        <label
            for="review-teacher-input"
            class="font-medium text-text-primary text-sm mb-1.5 block"
            >{{ t("review.post.teacherLabel") }}
            <span class="text-text-muted font-normal text-xs"
                >（{{ t("review.post.teacherOptional") }}）</span
            ></label
        >
        <div v-if="loading" class="text-xs text-text-muted py-2">
            {{ t("review.post.teacherLoading") }}
        </div>
        <div v-else class="relative">
            <input
                id="review-teacher-input"
                :value="modelValue"
                autocomplete="off"
                class="w-full p-3 bg-bg-secondary rounded-lg text-sm text-text-primary font-sans focus-ring-field"
                :placeholder="t('review.post.teacherPlaceholder')"
                @focus="dropdownOpen = true"
                @input="handleInput"
            />
            <button
                v-if="modelValue"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
                :aria-label="t('common.actions.clear')"
                @click="clearTeacher"
            >
                &times;
            </button>
            <div
                v-if="dropdownOpen && filteredTeachers.length > 0"
                class="absolute left-0 right-0 mt-1 rounded-lg bg-bg-card shadow-md max-h-[160px] overflow-y-auto z-10"
            >
                <button
                    v-for="teacher in filteredTeachers"
                    :key="teacher.teacherID"
                    class="flex items-center justify-between w-full p-2.5 text-left text-sm text-text-primary hover:bg-bg-hover transition-colors duration-fast"
                    @mousedown.prevent="selectTeacher(teacher)"
                >
                    <span>{{ teacher.teacherName }}</span>
                    <span class="text-xs text-text-muted">{{
                        teacher.departmentName
                    }}</span>
                </button>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { TeacherStats } from "@stuhelper/shared/course";

const props = defineProps<{
    modelValue: string;
    selectedTeacherId: number | null;
    teachers: TeacherStats[];
    loading: boolean;
}>();

const emit = defineEmits<{
    "update:modelValue": [value: string];
    "update:selectedTeacherId": [value: number | null];
}>();

const { t } = useI18n();
const dropdownOpen = ref(false);

const filteredTeachers = computed(() => {
    const q = props.modelValue.trim().toLowerCase();
    if (!q) return props.teachers;
    return props.teachers.filter((teacher) =>
        teacher.teacherName.toLowerCase().includes(q),
    );
});

function handleInput(event: Event) {
    emit("update:modelValue", (event.target as HTMLInputElement).value);
    emit("update:selectedTeacherId", null);
}

function selectTeacher(teacher: TeacherStats) {
    emit("update:selectedTeacherId", teacher.teacherID);
    emit("update:modelValue", teacher.teacherName);
    dropdownOpen.value = false;
}

function clearTeacher() {
    emit("update:selectedTeacherId", null);
    emit("update:modelValue", "");
}
</script>
