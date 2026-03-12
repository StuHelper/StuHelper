<script setup lang="ts">
import { computed } from "vue";
import { RouterLink } from "vue-router";

type PageKey = "about" | "privacy" | "terms";

const props = defineProps<{
    pageKey: PageKey;
}>();

const pageContent = computed(() => {
    switch (props.pageKey) {
        case "privacy":
            return {
                eyebrow: "Privacy",
                title: "隐私政策",
                intro: "StuHelper 仅收集维持服务运行所必需的数据，并尽量避免直接处理可识别个人身份的信息。",
                sections: [
                    {
                        heading: "我们收集什么",
                        body: "登录状态、必要的访问日志、课程互动产生的数据，以及用于安全校验的 Cookie。",
                    },
                    {
                        heading: "我们如何使用",
                        body: "用于身份验证、风险控制、内容审核、功能统计，以及改进课程评价体验。",
                    },
                    {
                        heading: "你的控制权",
                        body: "你可以随时退出登录、删除或修改自己发布的内容；如需进一步的数据协助，请联系维护者。",
                    },
                ],
            };
        case "terms":
            return {
                eyebrow: "Terms",
                title: "服务条款",
                intro: "使用 StuHelper 即表示你同意遵守校园社区的基本规则，不发布违法、骚扰、虚假或恶意内容。",
                sections: [
                    {
                        heading: "内容责任",
                        body: "你需要对自己发布的测评、回复和举报信息负责，平台有权对违规内容进行隐藏、删除或限制。",
                    },
                    {
                        heading: "账户与访问",
                        body: "登录能力依赖统一身份认证系统，异常访问、批量抓取或绕过安全机制的行为会被限制。",
                    },
                    {
                        heading: "服务变更",
                        body: "平台会持续迭代功能与规则，重要调整会通过站内公告或文档说明同步。",
                    },
                ],
            };
        case "about":
        default:
            return {
                eyebrow: "About",
                title: "关于 StuHelper",
                intro: "StuHelper 是面向校园学习场景的课程评价与信息协作平台，目标是让选课、了解教师和获取学习信息更直接。",
                sections: [
                    {
                        heading: "当前能力",
                        body: "现阶段已上线课程评价、教师详情、通知中心与基础审核后台，其他模块会按优先级逐步开放。",
                    },
                    {
                        heading: "内容原则",
                        body: "平台鼓励基于真实体验的理性表达，强调可验证、可追溯和对他人有帮助的信息。",
                    },
                    {
                        heading: "开发方向",
                        body: "项目采用 OpenAPI 3 Spec-First、多端共享 API 客户端和统一 SSO 登录，后续会继续完善教学相关模块。",
                    },
                ],
            };
    }
});
</script>

<template>
    <main
        class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(14,165,233,0.14),_transparent_38%),linear-gradient(180deg,_#f8fafc_0%,_#ffffff_100%)] px-6 py-16"
    >
        <section
            class="mx-auto max-w-4xl overflow-hidden rounded-[32px] border border-slate-200/80 bg-white/90 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur"
        >
            <div class="border-b border-slate-200/80 px-8 py-10">
                <p
                    class="mb-3 text-xs font-semibold uppercase tracking-[0.28em] text-cyan-600"
                >
                    {{ pageContent.eyebrow }}
                </p>
                <h1
                    class="m-0 text-4xl font-black tracking-tight text-slate-900"
                >
                    {{ pageContent.title }}
                </h1>
                <p class="mt-4 max-w-3xl text-base leading-7 text-slate-600">
                    {{ pageContent.intro }}
                </p>
            </div>

            <div class="space-y-8 px-8 py-10">
                <article
                    v-for="section in pageContent.sections"
                    :key="section.heading"
                    class="rounded-2xl border border-slate-200 bg-slate-50/70 p-6"
                >
                    <h2 class="m-0 text-xl font-bold text-slate-900">
                        {{ section.heading }}
                    </h2>
                    <p class="mt-3 mb-0 text-sm leading-7 text-slate-600">
                        {{ section.body }}
                    </p>
                </article>

                <div class="flex flex-wrap gap-3">
                    <RouterLink
                        to="/"
                        class="rounded-full bg-slate-900 px-5 py-2.5 text-sm font-medium text-white no-underline transition-opacity hover:opacity-90"
                    >
                        返回首页
                    </RouterLink>
                    <RouterLink
                        to="/course"
                        class="rounded-full border border-slate-300 px-5 py-2.5 text-sm font-medium text-slate-700 no-underline transition-colors hover:border-cyan-500 hover:text-cyan-600"
                    >
                        进入学习中心
                    </RouterLink>
                </div>
            </div>
        </section>
    </main>
</template>
