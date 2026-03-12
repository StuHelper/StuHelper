<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import CourseCard from "@/components/business/CourseCard.vue";
import FadeIn from "@/components/animated/FadeIn.vue";
import SearchBar from "@/components/ui/SearchBar.vue";
import Loading from "@/components/ui/Loading.vue";
import Empty from "@/components/ui/Empty.vue";
import Pagination from "@/components/ui/Pagination.vue";
import { api } from "@/api";
import type { components } from "@stuhelper/shared/types";
import { buildCourseListQuery } from "../courseListQuery";

type CourseWithExtras = components["schemas"]["Course"] & {
    averageRating?: number;
    teacherName?: string;
    hours?: number;
    isRequired?: boolean;
};

type DepartmentOption = components["schemas"]["Department"];
type CourseSortKey = "name" | "credits" | "reviewCount";

const departments = ref<DepartmentOption[]>([]);
const courses = ref<CourseWithExtras[]>([]);
const totalCount = ref(0);
const loading = ref(true);

const searchQuery = ref("");
const selectedDepartment = ref<number | "">("");
const sortBy = ref<CourseSortKey>("name");
const currentPage = ref(1);
const pageSize = ref(12);

const totalPages = computed(() =>
    totalCount.value === 0 ? 0 : Math.ceil(totalCount.value / pageSize.value),
);

const paginatedCourses = computed(() => courses.value);

async function fetchCourses() {
    loading.value = true;
    try {
        const response = await api.course.getCourses(
            buildCourseListQuery({
                page: currentPage.value,
                pageSize: pageSize.value,
                searchQuery: searchQuery.value,
                selectedDepartment: selectedDepartment.value,
                sortBy: sortBy.value,
            }),
        );
        courses.value = response.data?.data?.list || [];
        totalCount.value = response.data?.data?.total || 0;
    } catch {
        courses.value = [];
        totalCount.value = 0;
    } finally {
        loading.value = false;
    }
}

async function fetchDepartments() {
    try {
        const response = await api.course.getDepartments();
        departments.value = response.data?.data || [];
    } catch {
        const fallbackDepartments = new Map<number, DepartmentOption>();
        courses.value.forEach((course) => {
            if (!fallbackDepartments.has(course.departmentID)) {
                fallbackDepartments.set(course.departmentID, {
                    id: course.departmentID,
                    name:
                        course.departmentName || `院系 ${course.departmentID}`,
                    category: "",
                });
            }
        });
        departments.value = Array.from(fallbackDepartments.values()).sort(
            (a, b) => a.name.localeCompare(b.name, "zh-CN"),
        );
    }
}

function resetFilters() {
    searchQuery.value = "";
    selectedDepartment.value = "";
    sortBy.value = "name";
}

watch([searchQuery, selectedDepartment, sortBy, pageSize], () => {
    if (currentPage.value !== 1) {
        currentPage.value = 1;
        return;
    }
    void fetchCourses();
});

watch(currentPage, () => {
    void fetchCourses();
});

watch(totalPages, (value) => {
    if (value === 0) {
        currentPage.value = 1;
        return;
    }

    if (currentPage.value > value) {
        currentPage.value = value;
    }
});

onMounted(async () => {
    await fetchCourses();
    await fetchDepartments();
});
</script>

<template>
    <div
        class="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900"
    >
        <div
            class="relative overflow-hidden bg-gradient-to-r from-cyan-600 via-indigo-600 to-indigo-700 py-20 text-white"
        >
            <div
                class="absolute inset-0 bg-[url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PGRlZnM+PHBhdHRlcm4gaWQ9ImdyaWQiIHdpZHRoPSI2MCIgaGVpZ2h0PSI2MCIgcGF0dGVyblVuaXRzPSJ1c2VyU3BhY2VPblVzZSI+PHBhdGggZD0iTSAxMCAwIEwgMCAwIDAgMTAiIGZpbGw9Im5vbmUiIHN0cm9rZT0id2hpdGUiIHN0cm9rZS1vcGFjaXR5PSIwLjA1IiBzdHJva2Utd2lkdGg9IjEiLz48L3BhdHRlcm4+PC9kZWZzPjxyZWN0IHdpZHRoPSIxMDAlIiBoZWlnaHQ9IjEwMCUiIGZpbGw9InVybCgjZ3JpZCkiLz48L3N2Zz4=')] opacity-30"
            ></div>
            <div class="container relative z-10 mx-auto px-8">
                <h1 class="mb-4 text-6xl font-bold tracking-tight">课程列表</h1>
                <p class="text-xl text-cyan-100">
                    探索所有课程，找到更适合你的学习内容
                </p>
            </div>
        </div>

        <div class="container mx-auto px-8 py-12">
            <div
                class="mb-10 space-y-4 rounded-2xl border border-white/10 bg-white/5 p-6 shadow-2xl backdrop-blur-xl"
            >
                <SearchBar
                    v-model="searchQuery"
                    placeholder="搜索课程名称或课程代码..."
                />

                <div class="flex flex-wrap gap-4">
                    <select
                        v-model.number="selectedDepartment"
                        class="rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-white backdrop-blur-sm transition-all hover:bg-white/10 focus:border-transparent focus:ring-2 focus:ring-cyan-500"
                    >
                        <option value="" class="bg-slate-800">全部院系</option>
                        <option
                            v-for="dept in departments"
                            :key="dept.id"
                            :value="dept.id"
                            class="bg-slate-800"
                        >
                            {{ dept.name }}
                        </option>
                    </select>

                    <select
                        v-model="sortBy"
                        class="rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-white backdrop-blur-sm transition-all hover:bg-white/10 focus:border-transparent focus:ring-2 focus:ring-cyan-500"
                    >
                        <option value="name" class="bg-slate-800">
                            按名称排序
                        </option>
                        <option value="credits" class="bg-slate-800">
                            按学分排序
                        </option>
                        <option value="reviewCount" class="bg-slate-800">
                            按评价数排序
                        </option>
                    </select>
                </div>
            </div>

            <Loading v-if="loading" type="card" :count="6" />

            <Empty
                v-else-if="paginatedCourses.length === 0"
                title="暂无课程"
                description="没有找到符合条件的课程"
                action-text="清除筛选"
                @action="resetFilters"
            />

            <div
                v-else
                class="mb-12 grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-3"
            >
                <FadeIn
                    v-for="(course, index) in paginatedCourses"
                    :key="course.id"
                    :delay="index * 50"
                >
                    <RouterLink
                        :to="`/courses/${course.id}`"
                        class="block no-underline"
                    >
                        <CourseCard
                            :course="course"
                            class="transition-transform duration-300 hover:scale-105"
                        />
                    </RouterLink>
                </FadeIn>
            </div>

            <Pagination
                v-if="!loading && paginatedCourses.length > 0"
                :current-page="currentPage"
                :page-size="pageSize"
                :total-pages="totalPages"
                :total-count="totalCount"
                @update:current-page="currentPage = $event"
                @update:page-size="pageSize = $event"
            />
        </div>
    </div>
</template>
