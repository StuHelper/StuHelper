import {
    createRouter,
    createWebHistory,
    type RouteLocationNormalized,
    type RouteLocationRaw,
    type RouteRecordRaw,
} from "vue-router";
import { useAuthStore } from "@/stores/auth";
import { isTokenExpired } from "@/utils/auth";
import { hasStoredSessionHint } from "@/utils/sessionHint";
import { updatePageMeta } from "@/composables/usePageMeta";
import i18n from "@/i18n";
import { REVIEW_CREATE } from "@stuhelper/shared/constants";
import {
    hasRequiredRouteCapabilityAccess,
    resolveProtectedRouteAuthFailure,
    shouldResolveRouteSession,
} from "@/router/auth-guard-decision";
import {
    currentHostname,
    shouldBlockAdmissionPathOutsideJoinHost,
    shouldBlockJoinHostRoute,
} from "@/router/join-domain";
import {
    safeGetSessionStorageItem,
    safeRemoveSessionStorageItem,
    safeSetSessionStorageItem,
} from "@/utils/browserStorage";
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
let chunkReloadAttemptedInMemory = false;

function hasChunkReloadAttempted(): boolean {
    return chunkReloadAttemptedInMemory ||
        safeGetSessionStorageItem(CHUNK_RELOAD_KEY) === "1";
}

function markChunkReloadAttempted(): boolean {
    chunkReloadAttemptedInMemory = true;
    return safeSetSessionStorageItem(CHUNK_RELOAD_KEY, "1");
}

function clearChunkReloadAttempted(): void {
    chunkReloadAttemptedInMemory = false;
    safeRemoveSessionStorageItem(CHUNK_RELOAD_KEY);
}

function lazyLoad(loader: () => Promise<unknown>) {
    return () =>
        loader().catch((err) => {
            if (!isChunkLoadError(err)) throw err;

            // 首次 chunk 失败时重新抛出，让 router.onError 处理目标路由
            if (!hasChunkReloadAttempted()) {
                throw err;
            }

            // 已重载过仍失败时，渲染静态错误页
            clearChunkReloadAttempted();
            return { default: ChunkErrorPage };
        });
}

const routes: RouteRecordRaw[] = [
    // 认证相关路由
    {
        path: "/login",
        name: "login",
        component: lazyLoad(() => import("@/modules/auth/views/LoginPage.vue")),
        meta: { titleKey: "routes.login", guest: true },
    },
    {
        path: "/auth/callback",
        name: "auth-callback",
        component: lazyLoad(
            () => import("@/modules/auth/views/AuthCallbackPage.vue"),
        ),
        meta: {
            titleKey: "routes.authCallback",
            guest: true,
            layout: "none",
        },
    },
    {
        path: "/consent",
        name: "open-platform-consent",
        component: lazyLoad(
            () => import("@/modules/open-platform/views/ConsentPage.vue"),
        ),
        meta: {
            titleKey: "routes.openPlatformConsent",
            requiresAuth: true,
            layout: "none",
        },
    },
    {
        path: "/complete-profile",
        name: "open-platform-profile-completion",
        component: lazyLoad(
            () =>
                import("@/modules/open-platform/views/ProfileCompletionPage.vue"),
        ),
        meta: {
            titleKey: "routes.openPlatformProfileCompletion",
            requiresAuth: true,
            layout: "none",
        },
    },
    {
        path: "/identity",
        name: "identity-home",
        component: lazyLoad(
            () => import("@/modules/user/views/IdentityHomePage.vue"),
        ),
        meta: {
            titleKey: "routes.identityHome",
            requiresAuth: true,
        },
    },
    {
        path: "/account/profile",
        name: "account-profile",
        component: lazyLoad(
            () => import("@/modules/user/views/AccountProfilePage.vue"),
        ),
        meta: {
            titleKey: "routes.accountProfile",
            requiresAuth: true,
        },
    },
    {
        path: "/account/security",
        name: "account-security",
        component: lazyLoad(
            () => import("@/modules/user/views/AccountSecurityPage.vue"),
        ),
        meta: {
            titleKey: "routes.accountSecurity",
            requiresAuth: true,
        },
    },
    {
        path: "/connect",
        name: "identity-connect",
        component: lazyLoad(
            () => import("@/modules/open-platform/views/ConnectPage.vue"),
        ),
        meta: {
            titleKey: "routes.identityConnect",
        },
    },
    {
        path: "/developers/apps",
        name: "open-platform-developer-apps",
        component: lazyLoad(
            () => import("@/modules/open-platform/views/DeveloperAppsPage.vue"),
        ),
        meta: {
            titleKey: "routes.openPlatformDeveloperApps",
            requiresAuth: true,
        },
    },
    {
        path: "/start",
        name: "join-start",
        component: lazyLoad(
            () => import("@/modules/admission/views/JoinStartPage.vue"),
        ),
        meta: { title: "学生认证与 QQ 绑定", layout: "none" },
    },
    {
        path: "/verify/:code",
        name: "admission-token",
        component: lazyLoad(
            () => import("@/modules/admission/views/AdmissionPage.vue"),
        ),
        meta: { title: "入群身份认证", layout: "none" },
    },
    {
        path: "/admission/freshman/camera/:token",
        name: "admission-freshman-mobile-camera",
        component: lazyLoad(
            () =>
                import(
                    "@/modules/admission/views/FreshmanMobileCameraPage.vue"
                ),
        ),
        meta: { title: "新生材料拍照", layout: "none" },
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
        path: "/courses",
        name: "course-hub",
        component: lazyLoad(
            () => import("@/modules/course/views/TeachingHubPage.vue"),
        ),
        meta: { titleKey: "routes.teachingHub" },
    },
    {
        path: "/courses/list",
        name: "course-list",
        component: lazyLoad(
            () => import("@/modules/course/views/CourseListPage.vue"),
        ),
        meta: { titleKey: "routes.courseList" },
    },
    {
        path: "/courses/about",
        name: "course-about",
        component: lazyLoad(
            () => import("@/modules/review/views/AboutPage.vue"),
        ),
        meta: { titleKey: "routes.courseAbout" },
    },

    // 资料共享
    {
        path: "/resources",
        name: "resource-list",
        component: lazyLoad(
            () => import("@/modules/resource/views/ResourceListPage.vue"),
        ),
        meta: { titleKey: "routes.resource" },
    },
    {
        path: "/resources/new",
        name: "resource-new",
        component: lazyLoad(
            () => import("@/modules/resource/views/ResourceEditPage.vue"),
        ),
        meta: { titleKey: "routes.resourceCreate", requiresAuth: true },
    },
    {
        path: "/resources/mine",
        name: "resource-mine",
        component: lazyLoad(
            () => import("@/modules/resource/views/ResourceMinePage.vue"),
        ),
        meta: { titleKey: "routes.resourceMine", requiresAuth: true },
    },
    {
        path: "/resources/:id/edit",
        name: "resource-edit",
        component: lazyLoad(
            () => import("@/modules/resource/views/ResourceEditPage.vue"),
        ),
        meta: { titleKey: "routes.resourceEdit", requiresAuth: true },
    },
    {
        path: "/resources/:id",
        name: "resource-detail",
        component: lazyLoad(
            () => import("@/modules/resource/views/ResourceDetailPage.vue"),
        ),
        meta: { titleKey: "routes.resourceDetail" },
    },

    // 评测模块
    {
        path: "/courses/reviews",
        name: "review",
        component: lazyLoad(
            () => import("@/modules/review/views/ReviewPage.vue"),
        ),
        meta: { titleKey: "routes.review" },
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
        path: "/courses/reviews/post",
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
        path: "/user/authorized-apps",
        name: "user-authorized-apps",
        component: lazyLoad(
            () => import("@/modules/user/views/AuthorizedAppsPage.vue"),
        ),
        meta: {
            titleKey: "routes.userAuthorizedApps",
            requiresAuth: true,
        },
    },

    // 认证与资料页面
    {
        path: "/user/identity-verification",
        name: "identity-verification",
        component: lazyLoad(
            () => import("@/modules/user/views/IdentityVerificationPage.vue"),
        ),
        meta: {
            titleKey: "routes.identityVerification",
            requiresAuth: true,
        },
    },
    {
        path: "/user/student-verification",
        name: "student-verification",
        component: lazyLoad(
            () => import("@/modules/user/views/StudentVerificationPage.vue"),
        ),
        meta: {
            titleKey: "routes.studentVerification",
            requiresAuth: true,
        },
    },
    {
        path: "/user/phone-binding",
        name: "phone-binding",
        component: lazyLoad(
            () => import("@/modules/user/views/PhoneBindingPage.vue"),
        ),
        meta: {
            titleKey: "routes.phoneBinding",
            requiresAuth: true,
        },
    },
    {
        path: "/user/qq-binding",
        name: "qq-binding",
        component: lazyLoad(
            () => import("@/modules/user/views/QQBindingPage.vue"),
        ),
        meta: {
            titleKey: "routes.qqBinding",
            requiresAuth: true,
        },
    },
    {
        path: "/user/academic-info",
        name: "academic-info",
        component: lazyLoad(
            () => import("@/modules/user/views/AcademicInfoPage.vue"),
        ),
        meta: {
            titleKey: "routes.academicInfo",
            requiresAuth: true,
        },
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

function buildNotFoundRoute(to: RouteLocationNormalized): RouteLocationRaw {
    const pathMatch = to.path.replace(/^\/+/, "").split("/").filter(Boolean);
    return {
        name: "not-found",
        params: pathMatch.length > 0 ? { pathMatch } : {},
        query: to.query,
        hash: to.hash,
        replace: true,
    };
}

router.beforeEach(async (to) => {
    const hostname = currentHostname();
    if (
        to.name !== "not-found" &&
        (shouldBlockJoinHostRoute(hostname, to.path) ||
            shouldBlockAdmissionPathOutsideJoinHost(hostname, to.path))
    ) {
        return buildNotFoundRoute(to);
    }

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

    const routeHasRequiredCapabilities = to.matched.some(
        (route) =>
            Array.isArray(route.meta.requiredCapabilities) &&
            route.meta.requiredCapabilities.length > 0,
    );
    const requiresAuthRoute =
        Boolean(to.meta.requiresAuth) || routeHasRequiredCapabilities;
    const hasSessionHint = hasStoredSessionHint();
    if (!hasSessionHint && authStore.isAuthenticated) {
        authStore.clearSession();
    }
    const needsResolvedSession = shouldResolveRouteSession({
        hasSessionHint,
        isAuthenticated: authStore.isAuthenticated,
        isGuestRoute: Boolean(to.meta.guest),
        requiresAuthRoute,
    });

    if (needsResolvedSession && !authStore.bootstrapCompleted) {
        await authStore.bootstrapSession();
    }

    let refreshFailed = false;
    if (needsResolvedSession && authStore.isAuthenticated && isTokenExpired()) {
        try {
            await authStore.refreshSession();
        } catch (err) {
            refreshFailed = true;
            if (import.meta.env.DEV) {
                console.warn(
                    "[Router] session refresh failed during navigation:",
                    err,
                );
            }
        }
    }

    const isAuthenticated = authStore.isAuthenticated;
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
            return false;
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

    const isReauthLogin =
        to.name === "login" &&
        typeof to.query.reauth === "string" &&
        to.query.reauth === "1";
    if (to.meta.guest && isAuthenticated && !isReauthLogin) {
        const redirect =
            typeof to.query.redirect === "string"
                ? to.query.redirect
                : undefined;
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
    clearChunkReloadAttempted();
});

// chunk 首次加载失败时，用目标路由 fullPath 重载，而不是直接刷新当前页
router.onError((err, to) => {
    if (!isChunkLoadError(err)) return;
    if (!hasChunkReloadAttempted()) {
        const persisted = markChunkReloadAttempted();
        if (!persisted) {
            void router.replace(to.fullPath);
            return;
        }
        window.location.assign(to.fullPath);
    }
});

export default router;
