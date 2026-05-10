import {
    createRouter,
    createWebHistory,
    type RouteRecordRaw,
} from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { isTokenExpired } from "@/utils/auth";
import { updatePageMeta } from "@/composables/usePageMeta";
import i18n from "@/i18n";
import {
    REVIEW_CREATE,
} from "@stuhelper/shared/constants";
import {
    hasRequiredRouteCapabilityAccess,
    resolveProtectedRouteAuthFailure,
} from "@/router/auth-guard-decision";
// 静态导入，确保 chunk load 失败时仍可渲染
import ChunkErrorPage from "@/modules/errors/views/ChunkErrorPage.vue";
import NotFoundPage from "@/modules/errors/views/NotFoundPage.vue";

declare module "vue-router" {
    interface RouteMeta {
        title?: string;
        requiresAuth?: boolean;
        requiredCapabilities?: string[];
        guest?: boolean;
        titleKey?: string;
        layout?: "shell" | "none";
    }
}

function isChunkLoadError(error: unknown): boolean {
    if (!(error instanceof Error)) return false;
    const msg = error.message;
    return (
        msg.includes("Failed to fetch dynamically imported module") ||
        msg.includes("Importing a module script failed") ||
        msg.includes("Loading chunk") ||
        msg.includes("Loading CSS chunk")
    );
}

// chunk 加载失败时自动重载一次
const CHUNK_RELOAD_KEY = "stuhelper_chunk_reload_attempted";

function lazyLoad(loader: () => Promise<unknown>) {
    return () =>
        loader().catch((err) => {
            if (!isChunkLoadError(err)) throw err;

            // 首次 chunk 失败时重新抛出，让 router.onError 处理目标路由
            const hasAttempted = sessionStorage.getItem(CHUNK_RELOAD_KEY);
            if (!hasAttempted) {
                throw err;
            }

            // 已重载过仍失败时，渲染静态错误页
            sessionStorage.removeItem(CHUNK_RELOAD_KEY);
            return { default: ChunkErrorPage };
        });
}

const routes: RouteRecordRaw[] = [
    // 认证相关路由
    {
        path: "/login",
        name: "login",
        component: lazyLoad(() => import("@/modules/auth/views/LoginPage.vue")),
        meta: { titleKey: "routes.login", guest: true, layout: "none" },
    },
    {
        path: "/auth/callback",
        name: "auth-callback",
        component: lazyLoad(
            () => import("@/modules/auth/views/AuthCallbackPage.vue"),
        ),
        meta: { titleKey: "routes.authCallback", guest: true, layout: "none" },
    },
    {
        path: "/admission/a/:code",
        name: "admission-token",
        component: lazyLoad(
            () => import("@/modules/admission/views/AdmissionPage.vue"),
        ),
        meta: { title: "入群身份认证", layout: "none" },
    },

    // 首页
    {
        path: "/",
        name: "home",
        component: lazyLoad(() => import("@/modules/home/views/HomePage.vue")),
        meta: { titleKey: "routes.home" },
    },

    {
        path: "/about",
        name: "about",
        component: lazyLoad(
            () => import("@/modules/common/views/InfoPage.vue"),
        ),
        props: { pageKey: "about" },
        meta: { titleKey: "routes.about" },
    },
    {
        path: "/privacy",
        name: "privacy",
        component: lazyLoad(
            () => import("@/modules/common/views/InfoPage.vue"),
        ),
        props: { pageKey: "privacy" },
        meta: { titleKey: "routes.privacy" },
    },
    {
        path: "/terms",
        name: "terms",
        component: lazyLoad(
            () => import("@/modules/common/views/InfoPage.vue"),
        ),
        props: { pageKey: "terms" },
        meta: { titleKey: "routes.terms" },
    },

    // 课程入口
    {
        path: "/course",
        name: "course-hub",
        component: lazyLoad(
            () => import("@/modules/course/views/TeachingHubPage.vue"),
        ),
        meta: { titleKey: "routes.teachingHub" },
    },

    // 评测模块
    {
        path: "/review",
        name: "review",
        component: lazyLoad(
            () => import("@/modules/review/views/ReviewPage.vue"),
        ),
        meta: { titleKey: "routes.review" },
    },
    {
        path: "/courses",
        name: "course-list",
        component: lazyLoad(
            () => import("@/modules/course/views/CourseListPage.vue"),
        ),
        meta: { titleKey: "routes.courseList" },
    },
    {
        path: "/courses/:id",
        name: "course-detail",
        component: lazyLoad(
            () => import("@/modules/review/views/CourseDetailPage.vue"),
        ),
        meta: { titleKey: "routes.courseDetail" },
    },
    {
        path: "/courses/:id/reviews",
        name: "course-reviews",
        component: lazyLoad(
            () => import("@/modules/review/views/CourseDetailPage.vue"),
        ),
        meta: { titleKey: "routes.courseReviews" },
    },
    {
        path: "/courses/:id/reviews/post",
        name: "course-review-post",
        component: lazyLoad(
            () => import("@/modules/review/views/PostReviewPage.vue"),
        ),
        meta: {
            titleKey: "routes.postReview",
            requiresAuth: true,
            requiredCapabilities: [REVIEW_CREATE],
        },
    },

    // 搜索页
    {
        path: "/search",
        name: "search",
        component: lazyLoad(
            () => import("@/modules/review/views/SearchPage.vue"),
        ),
        meta: { titleKey: "routes.search" },
    },

    // 评测说明页
    {
        path: "/course/about",
        name: "course-about",
        component: lazyLoad(
            () => import("@/modules/review/views/AboutPage.vue"),
        ),
        meta: { titleKey: "routes.courseAbout" },
    },

    // 教师相关页面
    {
        path: "/teachers",
        name: "teacher-hub",
        component: lazyLoad(
            () => import("@/modules/course/views/TeacherHubPage.vue"),
        ),
        meta: { titleKey: "routes.teacherHub" },
    },
    {
        path: "/teachers/:id",
        name: "teacher-profile",
        component: lazyLoad(
            () => import("@/modules/review/views/TeacherProfilePage.vue"),
        ),
        meta: { titleKey: "routes.teacherProfile" },
    },

    // 用户中心
    {
        path: "/user",
        redirect: { name: "user-reviews" },
    },
    {
        path: "/user/reviews",
        name: "user-reviews",
        component: lazyLoad(
            () => import("@/modules/user/views/UserCenterPage.vue"),
        ),
        meta: { titleKey: "routes.userReviews", requiresAuth: true },
    },
    {
        path: "/user/votes",
        name: "user-votes",
        component: lazyLoad(
            () => import("@/modules/user/views/UserCenterPage.vue"),
        ),
        meta: { titleKey: "routes.userVotes", requiresAuth: true },
    },
    {
        path: "/user/favorites",
        name: "user-favorites",
        component: lazyLoad(
            () => import("@/modules/user/views/UserCenterPage.vue"),
        ),
        meta: { titleKey: "routes.userFavorites", requiresAuth: true },
    },
    {
        path: "/user-center",
        name: "user-center",
        redirect: { name: "user-reviews" },
    },

    // 认证与资料页面
    {
        path: "/user/identity-verification",
        name: "identity-verification",
        component: lazyLoad(
            () => import("@/modules/user/views/IdentityVerificationPage.vue"),
        ),
        meta: { titleKey: "routes.identityVerification", requiresAuth: true },
    },
    {
        path: "/user/student-verification",
        name: "student-verification",
        component: lazyLoad(
            () => import("@/modules/user/views/StudentVerificationPage.vue"),
        ),
        meta: { titleKey: "routes.studentVerification", requiresAuth: true },
    },
    {
        path: "/user/phone-binding",
        name: "phone-binding",
        component: lazyLoad(
            () => import("@/modules/user/views/PhoneBindingPage.vue"),
        ),
        meta: { titleKey: "routes.phoneBinding", requiresAuth: true },
    },
    {
        path: "/user/qq-binding",
        name: "qq-binding",
        component: lazyLoad(
            () => import("@/modules/user/views/QQBindingPage.vue"),
        ),
        meta: { titleKey: "routes.qqBinding", requiresAuth: true },
    },
    {
        path: "/user/academic-info",
        name: "academic-info",
        component: lazyLoad(
            () => import("@/modules/user/views/AcademicInfoPage.vue"),
        ),
        meta: { titleKey: "routes.academicInfo", requiresAuth: true },
    },

    // 通知中心
    {
        path: "/notifications",
        name: "notifications",
        component: lazyLoad(
            () => import("@/modules/user/views/NotificationsPage.vue"),
        ),
        meta: { titleKey: "routes.notifications", requiresAuth: true },
    },

    // 兼容旧链接的重定向
    {
        path: "/review/courses/:id",
        redirect: (to) => `/courses/${to.params.id}/reviews`,
    },
    { path: "/review/courses", redirect: { name: "course-list" } },
    {
        path: "/review/teachers/:id",
        redirect: (to) => `/teachers/${to.params.id}`,
    },
    {
        path: "/courses/:id/review",
        redirect: (to) => `/courses/${to.params.id}/reviews`,
    },
    { path: "/teacher", redirect: "/teachers" },
    {
        path: "/:pathMatch(.*)*",
        name: "not-found",
        component: NotFoundPage,
        meta: { titleKey: "routes.notFound", layout: "none" },
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
    scrollBehavior(_to, _from, savedPosition) {
        return savedPosition || { top: 0 };
    },
});

router.beforeEach(async (to) => {
    if (to.path === "/auth/callback") {
        const code = typeof to.query.code === "string" ? to.query.code : "";
        const state = typeof to.query.state === "string" ? to.query.state : "";
        if (code.length > 4096 || state.length > 4096) {
            return { name: "login", query: { error: "invalid_callback" } };
        }
    }

    const authStore = useAuthStore();

    // 优先使用 titleKey 更新页面标题，其次退回静态 title
    const matchedTitleRoute = [...to.matched]
        .reverse()
        .find(
            (route) =>
                typeof route.meta.titleKey === "string" ||
                typeof route.meta.title === "string",
        );
    const resolvedTitle = matchedTitleRoute?.meta.titleKey
        ? i18n.global.t(matchedTitleRoute.meta.titleKey)
        : matchedTitleRoute?.meta.title;
    updatePageMeta({ title: resolvedTitle });

    const needsResolvedSession =
        Boolean(to.meta.requiresAuth) ||
        Boolean(to.meta.guest) ||
        to.matched.some((route) =>
            Array.isArray(route.meta.requiredCapabilities) &&
            route.meta.requiredCapabilities.length > 0,
        );

    if (needsResolvedSession && !authStore.bootstrapCompleted) {
        await authStore.bootstrapSession();
    }

    let refreshFailed = false;
    if (authStore.isAuthenticated && isTokenExpired()) {
        try {
            await authStore.refreshSession();
        } catch (err) {
            refreshFailed = true;
            if (import.meta.env.DEV) {
                console.warn("[Router] session refresh failed during navigation:", err);
            }
        }
    }

    const isAuthenticated = authStore.isAuthenticated;
    const requiresAuthRoute =
        to.meta.requiresAuth ||
        to.matched.some((route) => {
            const requiredCapabilities = Array.isArray(route.meta.requiredCapabilities)
                ? route.meta.requiredCapabilities
                : [];
            return requiredCapabilities.length > 0;
        });
    const authFailureDecision = resolveProtectedRouteAuthFailure({
        redirect: to.fullPath,
        refreshFailed,
        requiresAuthRoute,
        sessionBootstrapFailed:
            needsResolvedSession &&
            !authStore.bootstrapCompleted &&
            !authStore.isAuthenticated,
        stillAuthenticated: authStore.isAuthenticated,
    });
    if (authFailureDecision !== null) {
        if (authFailureDecision === false) {
            return true;
        }
        return authFailureDecision;
    }

    // 登录守卫
    if (to.meta.requiresAuth && !isAuthenticated) {
        return { name: "login", query: { redirect: to.fullPath } };
    }

    const requiredCapabilities = to.matched.flatMap((route) =>
        Array.isArray(route.meta.requiredCapabilities)
            ? route.meta.requiredCapabilities
            : [],
    );
    if (
        requiredCapabilities.length > 0 &&
        !hasRequiredRouteCapabilityAccess(
            authStore.user?.capabilities ?? [],
            requiredCapabilities,
        )
    ) {
        return { name: "home" };
    }

    if (to.meta.guest && isAuthenticated) {
        const redirect = typeof to.query.redirect === "string" ? to.query.redirect : undefined;
        if (
            redirect &&
            redirect.startsWith("/") &&
            !redirect.startsWith("//")
        ) {
            return redirect;
        }
        return { name: "home" };
    }

    return true;
});

router.afterEach(() => {
    // 导航成功说明 chunk 加载正常，清除重试标记
    sessionStorage.removeItem(CHUNK_RELOAD_KEY);
});

// chunk 首次加载失败时，用目标路由 fullPath 重载，而不是直接刷新当前页
router.onError((err, to) => {
    if (!isChunkLoadError(err)) return;
    const hasAttempted = sessionStorage.getItem(CHUNK_RELOAD_KEY);
    if (!hasAttempted) {
        sessionStorage.setItem(CHUNK_RELOAD_KEY, "1");
        window.location.assign(to.fullPath);
    }
});

export default router;
