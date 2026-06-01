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

    it("keeps protected login redirects on the current web/business origin", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).not.toContain("absoluteURLOnPreferredOrigin");
        expect(routerSource).not.toContain("configuredIdentityOrigin");
        expect(routerSource).toContain(
            'return { name: "login", query: { redirect: to.fullPath } }',
        );
    });

    it("sends review-triggered verification actions directly to the account center", () => {
        const reviewPostSource = readFileSync(
            resolve(__dirname, "../../composables/useReviewPost.ts"),
            "utf-8",
        );
        const courseDetailSource = readFileSync(
            resolve(__dirname, "../../modules/review/views/CourseDetailPage.vue"),
            "utf-8",
        );
        const reviewCardSource = readFileSync(
            resolve(__dirname, "../../components/business/review/ReviewCard.vue"),
            "utf-8",
        );

        for (const source of [reviewPostSource, courseDetailSource, reviewCardSource]) {
            expect(source).toContain("identityPortalURL");
            expect(source).toContain("/user/student-verification");
            expect(source).not.toContain("name: 'student-verification'");
        }
        expect(reviewPostSource).toContain("/user/identity-verification");
        expect(reviewPostSource).not.toContain("name: 'identity-verification'");
    });

    it("does not keep the legacy cross-origin identity redirect bootstrap exception", () => {
        const mainSource = readFileSync(
            resolve(__dirname, "../../main.ts"),
            "utf-8",
        );
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );

        expect(routerSource).not.toContain("pendingExternalLocationRedirect");
        expect(routerSource).not.toContain("hasPendingExternalLocationRedirect");
        expect(routerSource).not.toContain("replaceWithExternalLocation");
        expect(mainSource).not.toContain("isExpectedExternalRedirectAbort");
        expect(mainSource).not.toContain("hasPendingExternalLocationRedirect");
        expect(mainSource).not.toContain(
            "router startup interrupted by external redirect",
        );
    });

    it("uses the main site home route as the default login return target", () => {
        const loginSource = readFileSync(
            resolve(__dirname, "../../modules/auth/views/LoginPage.vue"),
            "utf-8",
        );

        expect(loginSource).toContain("function defaultAuthenticatedRoute()");
        expect(loginSource).not.toContain("configuredIdentityOrigin");
        expect(loginSource).not.toContain('return new URL("/identity"');
        expect(loginSource).toContain(
            'return new URL("/", window.location.origin).toString()',
        );
        expect(loginSource).toContain("return defaultAuthenticatedRoute()");
    });

    it("keeps account security as the identity-side hub for verification and bindings", () => {
        const accountSecuritySource = readFileSync(
            resolve(
                __dirname,
                "../../modules/user/views/AccountSecurityPage.vue",
            ),
            "utf-8",
        );
        const smokeSource = readFileSync(
            resolve(
                __dirname,
                "../../../../../infra/ops/prod-parity-browser-smoke.mjs",
            ),
            "utf-8",
        );

        for (const path of [
            "/user/phone-binding",
            "/user/authorized-apps",
            "/user/identity-verification",
            "/user/student-verification",
            "/user/qq-binding",
            "/user/academic-info",
        ]) {
            expect(accountSecuritySource).toContain(`to="${path}"`);
        }
        for (const label of [
            "绑定手机",
            "授权应用",
            "实名认证",
            "学生认证",
            "绑定 QQ",
            "学业信息",
        ]) {
            expect(smokeSource).toContain(label);
        }
    });

    it("keeps verification and binding page back actions inside the identity portal", () => {
        const identityPageFiles = [
            "../../modules/user/views/IdentityVerificationPage.vue",
            "../../modules/user/views/StudentVerificationPage.vue",
            "../../modules/user/views/PhoneBindingPage.vue",
            "../../modules/user/views/QQBindingPage.vue",
            "../../modules/user/views/AcademicInfoPage.vue",
        ];

        for (const file of identityPageFiles) {
            const source = readFileSync(resolve(__dirname, file), "utf-8");
            expect(source, file).toContain("function goBack");
            expect(source, file).toContain("router.push");
            expect(source, file).toContain("/identity");
            expect(source, file).not.toContain("router.back");
            expect(source, file).not.toContain("window.history");
        }
    });

    it("exposes Casdoor SSO as the public Connect issuer and keeps business data behind Open API", () => {
        const issuerFiles = [
            "../../i18n/locales/zh-CN/developer.ts",
            "../../i18n/locales/en-US/developer.ts",
            "../../modules/open-platform/connectEndpoints.ts",
        ];
        const checkedFiles = [
            ...issuerFiles,
            "../../modules/open-platform/views/ConnectPage.vue",
            "../../modules/open-platform/components/ConnectEndpointsPanel.vue",
            "../../modules/open-platform/views/DeveloperAppsPage.vue",
        ];

        for (const file of issuerFiles) {
            const source = readFileSync(resolve(__dirname, file), "utf-8");
            expect(source, file).toContain("sso.stuhelper.com");
            expect(source, file).not.toContain("id.stuhelper.com");
        }
        for (const file of checkedFiles) {
            const source = readFileSync(resolve(__dirname, file), "utf-8");
            expect(source, file).not.toContain("id.stuhelper.com");
            expect(source, file).not.toContain("StuHelper SSO");
        }
    });

    it("brands the public login page as StuHelper unified sign-in instead of a standalone SSO site", () => {
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
        expect(zhCommonSource).toContain("title: 'StuHelper 统一登录'");
        expect(zhCommonSource).toContain(
            "identityLogin: '使用统一身份认证登录'",
        );
        expect(zhCommonSource).not.toContain("使用 SSO 登录");
        expect(zhCommonSource).not.toContain("StuHelper SSO");
        expect(enCommonSource).toContain("title: 'StuHelper Sign-in'");
        expect(enCommonSource).toContain(
            "identityLogin: 'Continue with unified sign-in'",
        );
        expect(enCommonSource).not.toContain("Login with SSO");
        expect(enCommonSource).not.toContain("StuHelper SSO");
    });

    it("brands Open Platform challenge pages as StuHelper Connect with account-center fallback", () => {
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
            "connectEyebrow: 'StuHelper Connect'",
        );
        expect(zhCommonSource).toContain("openIdentityHome: '返回账号中心'");
        expect(enCommonSource).toContain(
            "connectEyebrow: 'StuHelper Connect'",
        );
        expect(enCommonSource).toContain(
            "openIdentityHome: 'Back to Account Center'",
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
            /name:\s*"login"[\s\S]*meta:\s*\{\s*titleKey:\s*"routes\.login",\s*guest:\s*true\s*\}/,
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

    it("keeps main-site floating navigation available because account pages live on the main site", () => {
        const shellSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppShell.vue"),
            "utf-8",
        );

        expect(shellSource).toContain("configuredIdentityOrigin");
        expect(shellSource).toContain(
            '<FloatingModuleNav v-if="!isIdentityPortalHost" />',
        );
    });

    it("keeps account-center routes on the main web app", () => {
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
        expect(routerSource).toContain('path: "/account/profile"');
        expect(routerSource).toContain('name: "account-profile"');
        expect(routerSource).toContain('path: "/account/security"');
        expect(routerSource).toContain('name: "account-security"');
        expect(routerSource).toContain('path: "/connect"');
        expect(routerSource).toContain('name: "identity-connect"');
        expect(routerSource).not.toContain("identityPortal");
        expect(headerSource).toContain("const logoRoute = computed");
        expect(headerSource).toContain(
            "{ to: '/identity', label: t('routes.identityHome')",
        );
        expect(headerSource).toContain(
            "{ to: '/account/profile', label: t('routes.accountProfile')",
        );
        expect(headerSource).toContain(
            "{ to: '/connect', label: t('routes.identityConnect')",
        );
        expect(routerSource).not.toContain('return { path: "/identity", replace: true }');
        expect(profileSource).toContain('to="/account/profile"');
        expect(profileSource).toContain("user.identityHome.accountProfile.title");
    });

    it("serves authorized apps as a dedicated account-center page", () => {
        const routerSource = readFileSync(
            resolve(__dirname, "../../router/index.ts"),
            "utf-8",
        );
        const userCenterSource = readFileSync(
            resolve(__dirname, "../../modules/user/views/UserCenterPage.vue"),
            "utf-8",
        );

        expect(routerSource).toMatch(
            /path:\s*"\/user\/authorized-apps"[\s\S]*AuthorizedAppsPage\.vue/,
        );
        expect(routerSource).not.toContain("identityPortal");
        expect(userCenterSource).not.toContain("AuthorizedAppsTab");
        expect(userCenterSource).not.toContain("ProfileSection");
        expect(userCenterSource).not.toContain("user-authorized-apps");
        expect(userCenterSource).not.toContain("user.myAuthorizedApps");
    });

    it("sends user menu account actions directly to the configured web origin", () => {
        const userMenuSource = readFileSync(
            resolve(__dirname, "../../components/layout/AppUserMenu.vue"),
            "utf-8",
        );

        expect(userMenuSource).toContain("identityPortalURL");
        expect(userMenuSource).toContain("navigateToExternalURL");
        expect(userMenuSource).toContain("'account-profile': '/account/profile'");
        expect(userMenuSource).toContain("'account-security': '/account/security'");
        expect(userMenuSource).toContain("'open-platform-developer-apps': '/developers/apps'");
        expect(userMenuSource).toContain("'identity-verification': '/user/identity-verification'");
        expect(userMenuSource).toContain("'student-verification': '/user/student-verification'");
        expect(userMenuSource).toContain("'qq-binding': '/user/qq-binding'");
        expect(userMenuSource).not.toContain("void router.push({ name: routeName })");
        expect(userMenuSource).not.toContain("@click=\"goTo('account-profile')\"");
        expect(userMenuSource).not.toContain("@click=\"goTo('account-security')\"");
        expect(userMenuSource).not.toContain("@click=\"goTo('open-platform-developer-apps')\"");
        expect(userMenuSource).not.toContain("@click=\"goTo('identity-verification')\"");
        expect(userMenuSource).not.toContain("@click=\"goTo('student-verification')\"");
        expect(userMenuSource).not.toContain("@click=\"goTo('qq-binding')\"");
    });

    it("keeps profile completion actions on the account center", () => {
        const profileCompletionSource = readFileSync(
            resolve(__dirname, "../../modules/open-platform/views/ProfileCompletionPage.vue"),
            "utf-8",
        );

        expect(profileCompletionSource).toContain("identityPortalURLForHref");
        expect(profileCompletionSource).toContain("profileCompletionActionURL(field.actionURL)");
        expect(profileCompletionSource).toContain(
            "return identityPortalURLForHref(actionURL) ?? actionURL",
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
