import {
    createRouter,
    createWebHistory,
    type RouteRecordRaw,
} from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { useReviewPost } from "@/composables/useReviewPost";
import { isTokenExpired } from "@/utils/auth";
import { updatePageMeta } from "@/composables/usePageMeta";
import i18n from "@/i18n";
// H3: 静态导入，确保 chunk load 失败时仍可渲染
import ChunkErrorPage from "@/modules/errors/views/ChunkErrorPage.vue";
import NotFoundPage from "@/modules/errors/views/NotFoundPage.vue";

declare module "vue-router" {
    interface RouteMeta {
        title?: string;
        requiresAuth?: boolean;
        requiresAdmin?: boolean;
        guest?: boolean;
        titleKey?: string;
        layout?: "shell" | "none" | "admin";
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

// H3: chunk load 失败时自动重载一次，防止部署更新导致白屏
const CHUNK_RELOAD_KEY = "chunk_reload_attempted";

function lazyLoad(loader: () => Promise<unknown>) {
    return () =>
        loader().catch((err) => {
            if (!isChunkLoadError(err)) throw err;

            // 首次 chunk 失败 → 重新抛出，让 router.onError 处理（它有目标路由信息）
            const hasAttempted = sessionStorage.getItem(CHUNK_RELOAD_KEY);
            if (!hasAttempted) {
                throw err;
            }

            // 已重载过仍失败 → 渲染静态错误页作为最终兜底
            sessionStorage.removeItem(CHUNK_RELOAD_KEY);
            return { default: ChunkErrorPage };
        });
}

const routes: RouteRecordRaw[] = [
    // Auth routes
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

    // Homepage (new fancy design)
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

    // Course hub (old homepage)
    {
        path: "/course",
        name: "course-hub",
        component: lazyLoad(
            () => import("@/modules/course/views/TeachingHubPage.vue"),
        ),
        meta: { titleKey: "routes.teachingHub" },
    },

    // Review module
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
        meta: { titleKey: "routes.postReview", requiresAuth: true },
    },

    // Teacher profile
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

    // User center
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

    // Verification pages
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

    // Notifications
    {
        path: "/notifications",
        name: "notifications",
        component: lazyLoad(
            () => import("@/modules/user/views/NotificationsPage.vue"),
        ),
        meta: { titleKey: "routes.notifications", requiresAuth: true },
    },

    // Admin panel
    {
        path: "/admin",
        component: lazyLoad(
            () => import("@/modules/admin/views/AdminLayout.vue"),
        ),
        meta: {
            titleKey: "routes.admin",
            requiresAuth: true,
            requiresAdmin: true,
            layout: "admin",
        },
        children: [
            {
                path: "",
                name: "admin-dashboard",
                component: lazyLoad(
                    () => import("@/modules/admin/views/DashboardPage.vue"),
                ),
            },
            {
                path: "reports",
                name: "admin-reports",
                component: lazyLoad(
                    () => import("@/modules/admin/views/ReportsPage.vue"),
                ),
            },
            {
                path: "reviews",
                name: "admin-reviews",
                component: lazyLoad(
                    () => import("@/modules/admin/views/ReviewsManagePage.vue"),
                ),
            },
            {
                path: "teachers",
                name: "admin-teachers",
                component: lazyLoad(
                    () =>
                        import("@/modules/admin/views/TeachersManagePage.vue"),
                ),
            },
            {
                path: "sensitive-words",
                name: "admin-sensitive-words",
                component: lazyLoad(
                    () =>
                        import("@/modules/admin/views/SensitiveWordsPage.vue"),
                ),
            },
            {
                path: "logs",
                name: "admin-logs",
                component: lazyLoad(
                    () => import("@/modules/admin/views/LogsPage.vue"),
                ),
            },
        ],
    },

    // Redirects
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

    // Update page meta — prefer titleKey (i18n) over hardcoded title
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

    if (authStore.isAuthenticated && isTokenExpired()) {
        try {
            await authStore.refreshSession();
        } catch {
            return undefined;
        }
    }

    const isAuthenticated = authStore.isAuthenticated;

    // Auth guards
    if (to.meta.requiresAuth && !isAuthenticated) {
        return { name: "login", query: { redirect: to.fullPath } };
    }

    if (
        to.matched.some((r) => r.meta.requiresAdmin) &&
        authStore.user?.isAdmin !== true
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
    const { showPostModal, closePostModal } = useReviewPost();
    if (showPostModal.value) closePostModal();
    // H3: 导航成功说明 chunk 加载正常，清除重试标记
    sessionStorage.removeItem(CHUNK_RELOAD_KEY);
});

// H3: chunk load 首次失败时，用目标路由 fullPath 重载（而非 reload 当前页）
router.onError((err, to) => {
    if (!isChunkLoadError(err)) return;
    const hasAttempted = sessionStorage.getItem(CHUNK_RELOAD_KEY);
    if (!hasAttempted) {
        sessionStorage.setItem(CHUNK_RELOAD_KEY, "1");
        window.location.assign(to.fullPath);
    }
});

export default router;
