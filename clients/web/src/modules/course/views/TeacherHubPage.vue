<script setup lang="ts">
import { RouterLink } from "vue-router";
import { GraduationCap, Search, Sparkles } from "lucide-vue-next";

const highlights = [
    {
        icon: Search,
        title: "教师资料入口",
        description:
            "教师详情页已经接入路由，后续会补完整的教师列表、搜索与筛选入口。",
    },
    {
        icon: GraduationCap,
        title: "教学相关能力",
        description: "该模块会逐步承接教师画像、课程关联、评分维度和统计展示。",
    },
    {
        icon: Sparkles,
        title: "当前状态",
        description:
            "教师模块主页仍在实现中，因此这里先提供占位说明，而不再错误跳转到课程列表。",
    },
];
</script>

<template>
    <main
        class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(79,70,229,0.18),_transparent_34%),linear-gradient(180deg,_#f8fafc_0%,_#eef2ff_100%)] px-6 py-16"
    >
        <section
            class="mx-auto max-w-5xl rounded-[36px] border border-indigo-100 bg-white/90 p-8 shadow-[0_32px_120px_rgba(79,70,229,0.12)] backdrop-blur"
        >
            <div class="grid gap-10 lg:grid-cols-[1.2fr_0.8fr]">
                <div>
                    <p
                        class="mb-3 text-xs font-semibold uppercase tracking-[0.28em] text-indigo-600"
                    >
                        Teacher Hub
                    </p>
                    <h1
                        class="m-0 text-4xl font-black tracking-tight text-slate-900"
                    >
                        教师主页正在建设中
                    </h1>
                    <p
                        class="mt-4 max-w-2xl text-base leading-7 text-slate-600"
                    >
                        当前已经保留了教师详情页路由，但聚合入口还没有最终完成。这里先提供稳定占位页，避免用户被带到无关页面。
                    </p>

                    <div class="mt-8 flex flex-wrap gap-3">
                        <RouterLink
                            to="/courses"
                            class="rounded-full bg-indigo-600 px-5 py-2.5 text-sm font-medium text-white no-underline transition-opacity hover:opacity-90"
                        >
                            先去看课程列表
                        </RouterLink>
                        <RouterLink
                            to="/review"
                            class="rounded-full border border-indigo-200 px-5 py-2.5 text-sm font-medium text-indigo-700 no-underline transition-colors hover:border-indigo-500 hover:text-indigo-600"
                        >
                            返回评课中心
                        </RouterLink>
                    </div>
                </div>

                <div class="space-y-4">
                    <article
                        v-for="item in highlights"
                        :key="item.title"
                        class="rounded-2xl border border-slate-200 bg-slate-50/80 p-5"
                    >
                        <component
                            :is="item.icon"
                            :size="20"
                            class="text-indigo-600"
                        />
                        <h2 class="mt-4 mb-1 text-lg font-bold text-slate-900">
                            {{ item.title }}
                        </h2>
                        <p class="m-0 text-sm leading-7 text-slate-600">
                            {{ item.description }}
                        </p>
                    </article>
                </div>
            </div>
        </section>
    </main>
</template>
