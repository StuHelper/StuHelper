import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

describe("style entrypoint", () => {
    it("loads main.css from the application entrypoint", () => {
        const mainSource = readFileSync(
            resolve(__dirname, "../../main.ts"),
            "utf-8",
        );
        expect(mainSource).toContain("./styles/main.css");
    });

    it("does not load tailwind.css directly from App.vue anymore", () => {
        const appSource = readFileSync(
            resolve(__dirname, "../../App.vue"),
            "utf-8",
        );
        expect(appSource).not.toContain("@/styles/tailwind.css");
    });

    it("hydrates sessions after mount only when a local session hint exists", () => {
        const mainSource = readFileSync(
            resolve(__dirname, "../../main.ts"),
            "utf-8",
        );

        expect(mainSource).toContain(
            "const hasSessionHint = hasStoredSessionHint()",
        );
        expect(mainSource).toMatch(
            /if \(!hasSessionHint && authStore\.isAuthenticated\) \{\s*authStore\.clearSession\(\)\s*\}/,
        );
        expect(mainSource).toMatch(
            /app\.mount\('#app'\)\s*if \(hasSessionHint\) \{\s*void authStore\.bootstrapSession\(\)\s*\}/,
        );
    });

    it("does not probe guest routes unless local auth state exists", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).toContain(
            "const hasSessionHint = hasStoredSessionHint()",
        );
        expect(routerSource).toMatch(
            /isGuestRoute:\s*Boolean\(to\.meta\.guest\),\s*requiresAuthRoute,/,
        );
        expect(routerSource).toContain(
            "isAuthenticated: authStore.isAuthenticated",
        );
    });

    it("returns identity login flows to the configured web origin", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).toContain("absoluteURLOnPreferredOrigin");
        expect(routerSource).toContain(
            "absoluteURLOnPreferredOrigin(redirect, webOrigin)",
        );
        expect(routerSource).toContain(
            "absoluteURLOnPreferredOrigin(from.fullPath, webOrigin)",
        );
    });

    it("uses the identity home route as the default login return target on the identity host", () => {
        const loginSource = readFileSync(
            resolve(__dirname, "../../modules/auth/views/LoginPage.vue"),
            "utf-8",
        );

        expect(loginSource).toContain("function defaultAuthenticatedRoute()");
        expect(loginSource).toContain("configuredIdentityOrigin");
        expect(loginSource).toContain(
            'return new URL("/identity", window.location.origin).toString()',
        );
        expect(loginSource).toContain(
            'return new URL("/", window.location.origin).toString()',
        );
        expect(loginSource).toContain("return defaultAuthenticatedRoute()");
    });

    it("brands the public login page as StuHelper ID instead of a standalone SSO site", () => {
        const loginSource = readFileSync(
            resolve(__dirname, "../../modules/auth/views/LoginPage.vue"),
            "utf-8",
        );
        const zhCommonSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/zh-CN/common.ts"),
            "utf-8",
        );
        const enCommonSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/en-US/common.ts"),
            "utf-8",
        );

        expect(loginSource).toContain("common.login.title");
        expect(loginSource).toContain("common.login.identityLogin");
        expect(loginSource).toContain("common.login.identityHint");
        expect(loginSource).not.toContain("common.login.ssoLogin");
        expect(loginSource).not.toContain("common.login.ssoHint");
        expect(zhCommonSource).toContain("title: 'StuHelper ID'");
        expect(zhCommonSource).toContain(
            "identityLogin: '使用统一身份认证登录'",
        );
        expect(zhCommonSource).not.toContain("使用 SSO 登录");
        expect(zhCommonSource).not.toContain("StuHelper SSO");
        expect(enCommonSource).toContain("title: 'StuHelper ID'");
        expect(enCommonSource).toContain(
            "identityLogin: 'Continue with StuHelper ID'",
        );
        expect(enCommonSource).not.toContain("Login with SSO");
        expect(enCommonSource).not.toContain("StuHelper SSO");
    });

    it("brands Open Platform challenge pages as StuHelper ID Connect with identity-home fallback", () => {
        const consentSource = readFileSync(
            resolve(
                __dirname,
                "../../modules/open-platform/views/ConsentPage.vue",
            ),
            "utf-8",
        );
        const completionSource = readFileSync(
            resolve(
                __dirname,
                "../../modules/open-platform/views/ProfileCompletionPage.vue",
            ),
            "utf-8",
        );
        const zhCommonSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/zh-CN/common.ts"),
            "utf-8",
        );
        const enCommonSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/en-US/common.ts"),
            "utf-8",
        );

        for (const source of [consentSource, completionSource]) {
            expect(source).toContain("connectEyebrow");
            expect(source).toContain("openIdentityHome");
            expect(source).toContain('to="/identity"');
            expect(source).not.toContain("StuHelper Identity");
        }
        expect(zhCommonSource).toContain(
            "connectEyebrow: 'StuHelper ID Connect'",
        );
        expect(zhCommonSource).toContain("openIdentityHome: '返回身份中心'");
        expect(enCommonSource).toContain(
            "connectEyebrow: 'StuHelper ID Connect'",
        );
        expect(enCommonSource).toContain(
            "openIdentityHome: 'Back to Identity Hub'",
        );
    });

    it("keeps the login page inside the shared shell without global background side effects", () => {
        const loginSource = readFileSync(
            resolve(__dirname, "../../modules/auth/views/LoginPage.vue"),
            "utf-8",
        );
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).toMatch(
            /name:\s*"login"[\s\S]*meta:\s*\{\s*titleKey:\s*"routes\.login",\s*guest:\s*true,\s*identityPortal:\s*true\s*\}/,
        );
        expect(loginSource).not.toContain("ParticleBackground");
        expect(loginSource).not.toContain(':global([data-theme="dark"])');
        expect(loginSource).not.toMatch(
            /\[data-theme="dark"\]\s*\{[\s\S]*opacity/,
        );
    });

    it("keeps anonymous favorite actions clickable without eager session bootstrap", () => {
        const favoriteSource = readFileSync(
            resolve(
                __dirname,
                "../../components/business/review/FavoriteButton.vue",
            ),
            "utf-8",
        );

        expect(favoriteSource).toContain(
            "if (loading.value || authStore.bootstrapPending) return true",
        );
        expect(favoriteSource).toContain(
            "if (authStore.isAuthenticated && !authStore.bootstrapCompleted) return true",
        );
    });

    it("keeps course and teacher lists as semantic links", () => {
        const courseListSource = readFileSync(
            resolve(__dirname, "../../modules/course/views/CourseListPage.vue"),
            "utf-8",
        );
        const teacherHubSource = readFileSync(
            resolve(__dirname, "../../modules/course/views/TeacherHubPage.vue"),
            "utf-8",
        );

        expect(courseListSource).toContain("<router-link");
        expect(courseListSource).toContain(
            ':to="`/courses/${course.id}/reviews`"',
        );
        expect(teacherHubSource).toContain("<router-link");
        expect(teacherHubSource).toContain(
            ':to="`/teachers/${teacher.teacherID}`"',
        );
    });

    it("uses plural courses routes for the course hub and course catalog", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );
        const headerSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppHeader.vue"),
            "utf-8",
        );
        const floatingNavSource = readFileSync(
            resolve(__dirname, "../../components/layout/FloatingModuleNav.vue"),
            "utf-8",
        );
        const teachingHubSource = readFileSync(
            resolve(
                __dirname,
                "../../modules/course/views/TeachingHubPage.vue",
            ),
            "utf-8",
        );

        expect(routerSource).toContain('path: "/courses"');
        expect(routerSource).toContain('name: "course-hub"');
        expect(routerSource).toContain('path: "/courses/list"');
        expect(routerSource).toContain('path: "/courses/about"');
        expect(routerSource).toContain('path: "/courses/reviews"');
        expect(routerSource).not.toContain('path: "/review"');
        expect(routerSource).not.toContain('path: "/review/');
        expect(routerSource).not.toContain('path: "/course"');
        expect(routerSource).not.toContain('path: "/course/');
        expect(headerSource).toContain("route.path === '/courses'");
        expect(headerSource).not.toContain("route.path.startsWith('/review')");
        expect(headerSource).toContain(
            "{ to: '/courses', label: t('nav.courses')",
        );
        expect(headerSource).not.toContain(
            "{ to: '/course', label: t('nav.review')",
        );
        expect(teachingHubSource).toContain('to="/courses/list"');
        expect(teachingHubSource).toContain('to="/courses/reviews"');
        expect(teachingHubSource).not.toContain('to="/review"');
        expect(floatingNavSource).toContain('{ to: "/courses/reviews"');
        expect(floatingNavSource).not.toContain('{ to: "/review"');
    });

    it("routes header write-review actions to the page form instead of the deleted modal flow", () => {
        const shellSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppShell.vue"),
            "utf-8",
        );
        const reviewPageSource = readFileSync(
            resolve(__dirname, "../../modules/review/views/ReviewPage.vue"),
            "utf-8",
        );
        const headerSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppHeader.vue"),
            "utf-8",
        );
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(shellSource).not.toContain("ReviewDialog");
        expect(shellSource).not.toContain("showPostModal");
        expect(headerSource).toContain(
            "router.push({ name: 'course-review-post' })",
        );
        expect(headerSource).not.toContain("openPostModal");
        expect(routerSource).toContain('path: "/courses/reviews/post"');
        expect(routerSource).not.toContain('path: "/courses/:id/reviews/post"');
        expect(routerSource).not.toContain(
            "rememberReviewPostCourse(courseID)",
        );
        expect(reviewPageSource).not.toContain("<ReviewDialog");
    });

    it("hides main-site floating navigation on the identity host", () => {
        const shellSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppShell.vue"),
            "utf-8",
        );

        expect(shellSource).toContain("configuredIdentityOrigin");
        expect(shellSource).toContain(
            '<FloatingModuleNav v-if="!isIdentityPortalHost" />',
        );
    });

    it("keeps the identity home route inside the identity portal", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );
        const headerSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppHeader.vue"),
            "utf-8",
        );
        const profileSource = readFileSync(
            resolve(__dirname, "../../modules/user/views/ProfileSection.vue"),
            "utf-8",
        );

        expect(routerSource).toContain('path: "/identity"');
        expect(routerSource).toContain('name: "identity-home"');
        expect(routerSource).toMatch(
            /path:\s*"\/identity"[\s\S]*meta:\s*\{[\s\S]*titleKey:\s*"routes\.identityHome"[\s\S]*requiresAuth:\s*true[\s\S]*identityPortal:\s*true[\s\S]*\}/,
        );
        expect(headerSource).toContain(
            "const logoRoute = computed(() => (isIdentityPortalHost.value ? '/identity' : '/'))",
        );
        expect(headerSource).toContain(
            "{ to: '/identity', label: t('routes.identityHome')",
        );
        expect(routerSource).toContain(
            'if (to.path === "/") {\n        return { path: "/identity", replace: true }',
        );
        expect(profileSource).toContain("accountSettingsUrl");
        expect(profileSource).toContain("user.identityHome.accountSecurity");
    });

    it("serves authorized apps as a dedicated identity portal page", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).toMatch(
            /path:\s*"\/user\/authorized-apps"[\s\S]*AuthorizedAppsPage\.vue/,
        );
        expect(routerSource).toMatch(
            /path:\s*"\/user\/authorized-apps"[\s\S]*identityPortal:\s*true/,
        );
    });

    it("keeps review drafts user-scoped and recoverable from the post page", () => {
        const draftApiSource = readFileSync(
            resolve(__dirname, "../../../../shared/src/api/draft.ts"),
            "utf-8",
        );
        const draftStoreSource = readFileSync(
            resolve(__dirname, "../../stores/draft.ts"),
            "utf-8",
        );
        const postPageSource = readFileSync(
            resolve(__dirname, "../../modules/review/views/PostReviewPage.vue"),
            "utf-8",
        );

        expect(draftApiSource).toContain(
            "client.GET('/api/v1/course/review/drafts')",
        );
        expect(draftApiSource).toContain(
            "client.DELETE('/api/v1/course/review/drafts')",
        );
        expect(draftApiSource).not.toContain("/drafts/${");
        expect(draftStoreSource).toContain(
            "const draft = ref<Draft | null>(null)",
        );
        expect(draftStoreSource).not.toContain("Record<number");
        expect(postPageSource).toContain("DraftPromptDialog");
        expect(postPageSource).toContain("await draftStore.loadDraft(true)");
        expect(postPageSource).toContain(
            "await draftStore.saveDraft(buildDraftPayload())",
        );
        expect(postPageSource).toContain("onBeforeRouteLeave");
        expect(postPageSource).toContain("await draftStore.deleteDraft()");
    });

    it("keeps clickable notification rows keyboard reachable", () => {
        const notificationItemSource = readFileSync(
            resolve(__dirname, "../../components/common/NotificationItem.vue"),
            "utf-8",
        );

        expect(notificationItemSource).toContain("<button");
        expect(notificationItemSource).toContain('type="button"');
        expect(notificationItemSource).not.toContain(
            '<div\\n    class="flex items-start gap-3 p-3 cursor-pointer',
        );
    });

    it("labels advanced search controls for direct browser interaction", () => {
        const searchSource = readFileSync(
            resolve(__dirname, "../../modules/review/views/SearchPage.vue"),
            "utf-8",
        );

        expect(searchSource).toContain('for="advanced-course-name"');
        expect(searchSource).toContain('id="advanced-course-name"');
        expect(searchSource).toContain('for="advanced-teacher-name"');
        expect(searchSource).toContain('id="advanced-teacher-name"');
    });

    it("shows explicit phone binding configuration errors", () => {
        const phoneBindingSource = readFileSync(
            resolve(__dirname, "../../modules/user/views/PhoneBindingPage.vue"),
            "utf-8",
        );
        const zhUserSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/zh-CN/user.ts"),
            "utf-8",
        );
        const enUserSource = readFileSync(
            resolve(__dirname, "../../i18n/locales/en-US/user.ts"),
            "utf-8",
        );

        expect(phoneBindingSource).toContain("status === 503");
        expect(phoneBindingSource).toContain(
            "user.verification.phone.serviceUnavailable",
        );
        expect(zhUserSource).toContain("serviceUnavailable");
        expect(enUserSource).toContain("serviceUnavailable");
    });
});
