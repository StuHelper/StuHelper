#!/usr/bin/env node
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, '../..');
const requireFromWeb = createRequire(resolve(repoRoot, 'clients/web/package.json'));

let chromium;
try {
  ({ chromium } = requireFromWeb('@playwright/test'));
} catch (error) {
  console.error('[prod-parity-browser-smoke] failed to load @playwright/test.');
  console.error('Run infra/ops/bootstrap-dev-ubuntu2404.sh or install clients dependencies first.');
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}

const timeoutMs = Number(process.env.PROD_PARITY_BROWSER_SMOKE_TIMEOUT_MS || 30000);
const webBaseURL = normalizeBaseURL(process.env.WEB_BASE_URL || 'https://stuhelper.com');
const frontendDirectBaseURL = normalizeBaseURL(
  process.env.FRONTEND_DIRECT_BASE_URL || 'http://127.0.0.1:28000',
);
const identityBaseURL = normalizeBaseURL(
  process.env.IDENTITY_BASE_URL ||
    process.env.CASDOOR_PUBLIC_AUTH_BASE_URL ||
    'https://id.stuhelper.com',
);
const adminBaseURL = normalizeBaseURL(process.env.ADMIN_BASE_URL || 'https://stuhelper.com');
const casdoorLoginUsername = process.env.PROD_PARITY_CASDOOR_LOGIN_USERNAME || 'admin';
const casdoorLoginPassword = process.env.PROD_PARITY_CASDOOR_LOGIN_PASSWORD || '123';
const admissionToken = process.env.PROD_PARITY_ADMISSION_TOKEN || 'PROD-PARITY-ADMIT-LOGIN';
const admissionQQ = process.env.PROD_PARITY_ADMISSION_QQ || '990001';
const evidenceFile =
  process.env.PROD_PARITY_BROWSER_SMOKE_EVIDENCE_FILE ||
  resolve(repoRoot, '.run/prod-parity/browser-smoke-evidence.json');
const screenshotDir =
  process.env.PROD_PARITY_BROWSER_SMOKE_SCREENSHOT_DIR || dirname(evidenceFile);
const criticalResourceTypes = new Set(['document', 'font', 'image', 'script', 'stylesheet']);
const telemetryRoutePattern = /\/api\/v1\/metrics\/(?:frontend-errors|vitals)(?:\?|$)/;
const bootstrapFallbackTexts = ['应用启动失败', 'App startup failed'];
const viewportVariants = [
  {
    name: 'desktop',
    viewport: { width: 1365, height: 900 },
    deviceScaleFactor: 1,
    isMobile: false,
    hasTouch: false,
  },
  {
    name: 'mobile',
    viewport: { width: 390, height: 844 },
    deviceScaleFactor: 2,
    isMobile: true,
    hasTouch: true,
  },
];

const checks = [
  {
    name: 'web-home',
    url: joinURL(webBaseURL, '/'),
    expectedTexts: ['StuHelper'],
  },
  {
    name: 'web-login',
    url: joinURL(webBaseURL, '/login'),
    expectedTexts: ['StuHelper'],
  },
  {
    name: 'frontend-direct-login-redirect',
    url: joinURL(frontendDirectBaseURL, '/login'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login')],
  },
  {
    name: 'frontend-direct-login-session-refresh',
    url: joinURL(frontendDirectBaseURL, '/'),
    flow: 'frontend-direct-login-session-refresh',
    expectedTexts: ['StuHelper'],
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/auth/refresh',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-developer-login',
    url: joinURL(identityBaseURL, '/developers/apps'),
    flow: 'identity-portal-shell',
    expectedTexts: ['登录', 'Login'],
    requiredTexts: ['身份中心', '个人资料', 'Connect', '授权应用', '开发者应用'],
    forbiddenTexts: ['课程', '教师', '评课'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/developers/apps'],
  },
  {
    name: 'identity-root-login',
    url: joinURL(identityBaseURL, '/'),
    flow: 'identity-portal-shell',
    expectedTexts: ['登录', 'Login'],
    requiredTexts: ['身份中心', '个人资料', 'Connect', '授权应用', '开发者应用'],
    forbiddenTexts: ['课程', '教师', '评课'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/identity'],
  },
  {
    name: 'identity-connect-public',
    url: joinURL(identityBaseURL, '/connect'),
    expectedTexts: ['StuHelper ID Connect'],
    requiredTexts: [
      joinURL(identityBaseURL, '/.well-known/openid-configuration'),
      joinURL(identityBaseURL, '/oauth2/authorize'),
      joinURL(identityBaseURL, '/oauth2/token'),
      joinURL(identityBaseURL, '/oidc/userinfo'),
    ],
    forbiddenTexts: ['课程', '教师', '评课'],
    expectedURLIncludes: joinURL(identityBaseURL, '/connect'),
  },
  {
    name: 'web-identity-connect-redirect',
    url: joinURL(webBaseURL, '/connect'),
    expectedTexts: ['StuHelper ID Connect'],
    requiredTexts: [
      joinURL(identityBaseURL, '/.well-known/openid-configuration'),
      joinURL(identityBaseURL, '/oauth2/token'),
    ],
    expectedURLIncludes: joinURL(identityBaseURL, '/connect'),
  },
  {
    name: 'identity-home-authenticated',
    url: joinURL(identityBaseURL, '/identity'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['身份中心', 'Identity Hub'],
    requiredTexts: ['个人资料', '账号安全', 'Connect', '授权应用', '开发者应用'],
    expectedURLIncludes: joinURL(identityBaseURL, '/identity'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-account-profile-authenticated',
    url: joinURL(identityBaseURL, '/account/profile'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['个人资料', 'Profile'],
    requiredTexts: ['联系信息', '授权披露字段', '实名认证'],
    expectedURLIncludes: joinURL(identityBaseURL, '/account/profile'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-account-security-authenticated',
    url: joinURL(identityBaseURL, '/account/security'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['账号安全', 'Account Security'],
    requiredTexts: ['当前浏览器会话', '退出当前会话', '打开账号设置'],
    expectedURLIncludes: joinURL(identityBaseURL, '/account/security'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-authorized-apps-authenticated',
    url: joinURL(identityBaseURL, '/user/authorized-apps'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['授权应用', 'Authorized Apps'],
    requiredTexts: ['授权应用'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/authorized-apps'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-developer-apps-authenticated',
    url: joinURL(identityBaseURL, '/developers/apps'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['开发者应用', 'Developer Apps'],
    requiredTexts: ['创建应用'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/developers/apps'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-consent-missing-token-authenticated',
    url: joinURL(identityBaseURL, '/consent'),
    flow: 'identity-connect-error-refresh',
    expectedTexts: ['StuHelper ID Connect'],
    requiredTexts: ['授权请求加载失败', '返回身份中心'],
    expectedURLIncludes: joinURL(identityBaseURL, '/consent'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
  },
  {
    name: 'identity-profile-completion-missing-token-authenticated',
    url: joinURL(identityBaseURL, '/complete-profile'),
    flow: 'identity-connect-error-refresh',
    expectedTexts: ['StuHelper ID Connect'],
    requiredTexts: ['资料补全请求加载失败', '返回身份中心'],
    expectedURLIncludes: joinURL(identityBaseURL, '/complete-profile'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
  },
  {
    name: 'identity-identity-verification-authenticated',
    url: joinURL(identityBaseURL, '/user/identity-verification'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['实名认证', 'Identity Verification'],
    requiredTexts: ['实名认证'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/identity-verification'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-student-verification-authenticated',
    url: joinURL(identityBaseURL, '/user/student-verification'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['学生认证', 'Student Verification'],
    requiredTexts: ['学生认证'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/student-verification'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-phone-binding-authenticated',
    url: joinURL(identityBaseURL, '/user/phone-binding'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['绑定手机', 'Phone Binding'],
    requiredTexts: ['绑定手机'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/phone-binding'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-qq-binding-authenticated',
    url: joinURL(identityBaseURL, '/user/qq-binding'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['绑定 QQ', 'QQ Binding'],
    requiredTexts: ['绑定 QQ'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/qq-binding'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'identity-academic-info-authenticated',
    url: joinURL(identityBaseURL, '/user/academic-info'),
    flow: 'identity-authenticated-refresh',
    expectedTexts: ['学业信息', 'Academic Info'],
    requiredTexts: ['学业信息'],
    forbiddenTexts: ['我的评价', '我的点赞', '我的收藏', 'My Reviews', 'My Votes', 'My Favorites'],
    expectedURLIncludes: joinURL(identityBaseURL, '/user/academic-info'),
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/profile/academic-info',
        statuses: [403, 404],
      },
    ],
  },
  {
    name: 'identity-main-route-redirect',
    url: joinURL(identityBaseURL, '/courses'),
    expectedTexts: ['评课社区@BUAA', 'Browse Courses'],
    expectedURLIncludes: joinURL(webBaseURL, '/courses'),
  },
  {
    name: 'web-login-session-refresh',
    url: joinURL(webBaseURL, '/'),
    flow: 'web-login-session-refresh',
    expectedTexts: ['StuHelper'],
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
      {
        url: 'https://cdn.casbin.org/flag-icons/**',
        contentType: 'image/svg+xml',
        body: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"></svg>\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/auth/refresh',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/user/profile',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/qq-binding',
        statuses: [404],
      },
      {
        urlIncludes: '/api/v1/user/identity',
        statuses: [404],
      },
    ],
  },
  {
    name: 'web-auth-callback-missing-code',
    url: joinURL(webBaseURL, '/auth/callback?state=prod-parity-smoke'),
    expectedTexts: ['登录失败', 'Login Failed'],
    requiredTexts: ['缺少授权码'],
  },
  {
    name: 'web-admission-login',
    url: joinURL(
      webBaseURL,
      `/admission/a/${encodeURIComponent(admissionToken)}?qq=${encodeURIComponent(admissionQQ)}`,
    ),
    expectedTexts: ['登录 StuHelper'],
    requiredTexts: ['入群身份认证', `QQ：${admissionQQ}`],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
      {
        urlIncludes: '/api/v1/auth/refresh',
        statuses: [401],
      },
    ],
  },
  {
    name: 'web-about',
    url: joinURL(webBaseURL, '/about'),
    expectedTexts: ['关于 StuHelper', 'About StuHelper'],
  },
  {
    name: 'web-privacy',
    url: joinURL(webBaseURL, '/privacy'),
    expectedTexts: ['隐私政策', 'Privacy Policy'],
  },
  {
    name: 'web-terms',
    url: joinURL(webBaseURL, '/terms'),
    expectedTexts: ['服务条款', 'Terms of Service'],
  },
  {
    name: 'web-course-hub',
    url: joinURL(webBaseURL, '/courses'),
    expectedTexts: ['评课社区@BUAA', 'Browse Courses'],
  },
  {
    name: 'web-course-list',
    url: joinURL(webBaseURL, '/courses/list'),
    expectedTexts: ['课程列表', 'Course List'],
    requiredTexts: ['生产等价课程'],
  },
  {
    name: 'web-course-about',
    url: joinURL(webBaseURL, '/courses/about'),
    expectedTexts: ['关于评课社区@BUAA'],
  },
  {
    name: 'web-course-detail',
    url: joinURL(webBaseURL, '/courses/900001'),
    expectedTexts: ['生产等价课程'],
    requiredTexts: ['生产等价课程', '生产等价学院'],
  },
  {
    name: 'web-course-detail-reviews',
    url: joinURL(webBaseURL, '/courses/900001/reviews'),
    expectedTexts: ['生产等价课程'],
    requiredTexts: ['生产等价课程', '生产等价评课'],
  },
  {
    name: 'web-review-feed',
    url: joinURL(webBaseURL, '/courses/reviews'),
    expectedTexts: ['最新', 'Latest'],
    requiredTexts: ['生产等价评课'],
  },
  {
    name: 'web-search',
    url: joinURL(webBaseURL, '/search'),
    expectedTexts: ['高级搜索', 'Advanced Search'],
  },
  {
    name: 'web-teacher-hub',
    url: joinURL(webBaseURL, '/teachers'),
    expectedTexts: ['教师主页', 'Teacher'],
    requiredTexts: ['生产等价教师'],
  },
  {
    name: 'web-teacher-profile',
    url: joinURL(webBaseURL, '/teachers/900001'),
    expectedTexts: ['生产等价教师'],
    requiredTexts: ['生产等价教师', '生产等价课程'],
  },
  {
    name: 'web-protected-review-post',
    url: joinURL(webBaseURL, '/courses/reviews/post'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [
      joinURL(identityBaseURL, '/login'),
      webRedirectQuery('/courses/reviews/post'),
    ],
  },
  {
    name: 'web-protected-user-reviews',
    url: joinURL(webBaseURL, '/user/reviews'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [
      joinURL(identityBaseURL, '/login'),
      webRedirectQuery('/user/reviews'),
    ],
  },
  {
    name: 'web-protected-user-votes',
    url: joinURL(webBaseURL, '/user/votes'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [
      joinURL(identityBaseURL, '/login'),
      webRedirectQuery('/user/votes'),
    ],
  },
  {
    name: 'web-protected-user-favorites',
    url: joinURL(webBaseURL, '/user/favorites'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [
      joinURL(identityBaseURL, '/login'),
      webRedirectQuery('/user/favorites'),
    ],
  },
  {
    name: 'web-protected-identity-home',
    url: joinURL(webBaseURL, '/identity'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/identity'],
  },
  {
    name: 'web-protected-account-profile',
    url: joinURL(webBaseURL, '/account/profile'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/account/profile'],
  },
  {
    name: 'web-protected-account-security',
    url: joinURL(webBaseURL, '/account/security'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/account/security'],
  },
  {
    name: 'web-protected-user-authorized-apps',
    url: joinURL(webBaseURL, '/user/authorized-apps'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/authorized-apps'],
  },
  {
    name: 'web-protected-identity-verification',
    url: joinURL(webBaseURL, '/user/identity-verification'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/identity-verification'],
  },
  {
    name: 'web-protected-student-verification',
    url: joinURL(webBaseURL, '/user/student-verification'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/student-verification'],
  },
  {
    name: 'web-protected-phone-binding',
    url: joinURL(webBaseURL, '/user/phone-binding'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/phone-binding'],
  },
  {
    name: 'web-protected-qq-binding',
    url: joinURL(webBaseURL, '/user/qq-binding'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/qq-binding'],
  },
  {
    name: 'web-protected-academic-info',
    url: joinURL(webBaseURL, '/user/academic-info'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/user/academic-info'],
  },
  {
    name: 'web-protected-notifications',
    url: joinURL(webBaseURL, '/notifications'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [
      joinURL(identityBaseURL, '/login'),
      webRedirectQuery('/notifications'),
    ],
  },
  {
    name: 'web-protected-developer-apps',
    url: joinURL(webBaseURL, '/developers/apps'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/developers/apps'],
  },
  {
    name: 'web-protected-open-platform-consent',
    url: joinURL(webBaseURL, '/consent?token=smoke'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/consent?token=smoke'],
  },
  {
    name: 'web-protected-profile-completion',
    url: joinURL(webBaseURL, '/complete-profile?token=smoke'),
    expectedTexts: ['登录', 'Login'],
    expectedURLIncludes: [joinURL(identityBaseURL, '/login'), 'redirect=/complete-profile?token=smoke'],
  },
  {
    name: 'web-not-found',
    url: joinURL(webBaseURL, '/this-route-does-not-exist'),
    expectedTexts: ['页面不存在', 'Page Not Found'],
  },
  {
    name: 'admin-login-redirect',
    url: joinURL(adminBaseURL, '/admin/'),
    expectedTexts: ['Sign In', 'Password', '登录'],
    expectedURLIncludes: '/login/oauth/authorize',
    stubbedResources: [
      {
        url: 'https://fonts.googleapis.com/**',
        contentType: 'text/css',
        body: '/* prod-parity smoke uses system fonts for the Casdoor login page. */\n',
      },
    ],
    allowedAPIResponses: [
      {
        urlIncludes: '/api/v1/auth/me',
        statuses: [401],
      },
    ],
  },
];

const results = [];
let passed = false;
let browser;

try {
  await mkdir(screenshotDir, { recursive: true });
  browser = await chromium.launch({ headless: process.env.PLAYWRIGHT_HEADLESS !== '0' });

  for (const viewportVariant of viewportVariants) {
    for (const check of checks) {
      results.push(await runCheck(browser, check, viewportVariant));
    }
  }

  passed = results.every((result) => result.passed);
} catch (error) {
  results.push({
    name: 'fatal',
    passed: false,
    error: error instanceof Error ? error.message : String(error),
  });
} finally {
  if (browser) {
    await browser.close();
  }

  const evidence = {
    generatedAt: new Date().toISOString(),
    passed,
    webBaseURL,
    identityBaseURL,
    adminBaseURL,
    viewportVariants,
    checks: results,
  };
  await writeFile(evidenceFile, `${JSON.stringify(evidence, null, 2)}\n`);
}

if (!passed) {
  console.error(`[prod-parity-browser-smoke] failed; evidence: ${evidenceFile}`);
  process.exit(1);
}

console.log(`[prod-parity-browser-smoke] passed; evidence: ${evidenceFile}`);

async function runCheck(browser, check, viewportVariant) {
  const context = await browser.newContext({
    viewport: viewportVariant.viewport,
    deviceScaleFactor: viewportVariant.deviceScaleFactor,
    isMobile: viewportVariant.isMobile,
    hasTouch: viewportVariant.hasTouch,
    ignoreHTTPSErrors: true,
  });
  const page = await context.newPage();
  const assetFailures = [];
  const apiFailures = [];
  const consoleErrors = [];
  const ignoredConsoleErrors = [];
  const ignoredAPIResponses = [];
  const pageErrors = [];
  const suppressedTelemetryRequests = [];
  const stubbedExternalResources = [];
  const checkName = `${check.name}-${viewportVariant.name}`;
  const screenshotFile = resolve(screenshotDir, `${checkName}.png`);

  for (const resourceStub of resourceStubsForCheck(check)) {
    await page.route(resourceStub.url, async (route) => {
      const request = route.request();
      const status = resourceStub.status || 200;
      const contentType = resourceStub.contentType || 'text/plain';
      stubbedExternalResources.push({
        url: request.url(),
        resourceType: request.resourceType(),
        status,
        contentType,
      });
      await route.fulfill({
        status,
        contentType,
        body: resourceStub.body || '',
      });
    });
  }

  await page.route(telemetryRoutePattern, async (route) => {
    suppressedTelemetryRequests.push({
      method: route.request().method(),
      url: route.request().url(),
    });
    await route.fulfill({ status: 204, body: '' });
  });

  page.on('console', (message) => {
    if (message.type() === 'error') {
      const described = describeConsoleMessage(message);
      if (isIgnoredConsoleError(message.text())) {
        ignoredConsoleErrors.push(described);
      } else {
        consoleErrors.push(described);
      }
    }
  });
  page.on('pageerror', (error) => {
    pageErrors.push(error.message);
  });
  page.on('requestfailed', (request) => {
    if (criticalResourceTypes.has(request.resourceType())) {
      assetFailures.push({
        url: request.url(),
        resourceType: request.resourceType(),
        failure: request.failure()?.errorText || 'unknown',
      });
    }
  });
  page.on('response', (response) => {
    const request = response.request();
    const resourceType = request.resourceType();
    if (
      ['fetch', 'xhr'].includes(resourceType) &&
      response.status() >= 400
    ) {
      const described = {
        url: response.url(),
        resourceType,
        status: response.status(),
        statusText: response.statusText(),
      };
      if (isAllowedAPIResponse(check, response)) {
        ignoredAPIResponses.push(described);
      } else {
        apiFailures.push(described);
      }
    }
    if (
      criticalResourceTypes.has(resourceType) &&
      response.status() >= 400
    ) {
      assetFailures.push({
        url: response.url(),
        resourceType,
        status: response.status(),
        statusText: response.statusText(),
      });
    }
  });

  try {
    const response = await page.goto(check.url, {
      timeout: timeoutMs,
      waitUntil: 'domcontentloaded',
    });

    if (!response) {
      throw new Error(`no response for ${check.url}`);
    }
    if (response.status() >= 400) {
      throw new Error(`unexpected HTTP ${response.status()} for ${check.url}`);
    }

    const flowResult = await runCheckFlow(page, check, viewportVariant);

    const expectedURLIncludes = toArray(check.expectedURLIncludes);
    if (expectedURLIncludes.length > 0) {
      await page.waitForURL(
        (url) => expectedURLIncludes.every((text) => url.href.includes(text)),
        { timeout: timeoutMs },
      );
    }

    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);
    const title = await page.title();
    const bodyText = await page.locator('body').innerText({ timeout: timeoutMs });
    const matchedText = flowResult
      ? flowResult.matchedText
      : check.expectedTexts.find((text) => bodyText.includes(text));
    const requiredTexts = toArray(check.requiredTexts);
    const missingRequiredTexts = requiredTexts.filter((text) => !bodyText.includes(text));
    const forbiddenTexts = [
      ...bootstrapFallbackTexts,
      ...toArray(check.forbiddenTexts),
    ];
    const presentForbiddenTexts = forbiddenTexts.filter((text) => bodyText.includes(text));

    if (!matchedText) {
      throw new Error(
        `missing expected text; expected one of: ${check.expectedTexts.join(', ')}`,
      );
    }
    if (missingRequiredTexts.length > 0) {
      throw new Error(
        `missing required text: ${missingRequiredTexts.join(', ')}`,
      );
    }
    if (presentForbiddenTexts.length > 0) {
      throw new Error(
        `present forbidden text: ${presentForbiddenTexts.join(', ')}`,
      );
    }
    if (
      expectedURLIncludes.length > 0 &&
      expectedURLIncludes.some((text) => !page.url().includes(text))
    ) {
      throw new Error(
        `final URL ${page.url()} does not include ${expectedURLIncludes.join(', ')}`,
      );
    }
    if (pageErrors.length > 0) {
      throw new Error(`page errors: ${pageErrors.join(' | ')}`);
    }
    if (assetFailures.length > 0) {
      throw new Error(`asset failures: ${JSON.stringify(assetFailures)}`);
    }
    if (apiFailures.length > 0) {
      throw new Error(`api failures: ${JSON.stringify(apiFailures)}`);
    }
    if (consoleErrors.length > 0) {
      throw new Error(`console errors: ${consoleErrors.join(' | ')}`);
    }

    await page.screenshot({ path: screenshotFile, fullPage: true });

    return {
      name: check.name,
      checkName,
      viewport: viewportVariant.name,
      viewportSize: viewportVariant.viewport,
      passed: true,
      url: check.url,
      finalURL: page.url(),
      status: response.status(),
      title,
      matchedText,
      requiredTexts,
      forbiddenTexts,
      flowResult,
      ignoredConsoleErrors,
      ignoredAPIResponses,
      suppressedTelemetryRequests,
      stubbedExternalResources,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } catch (error) {
    await page.screenshot({ path: screenshotFile, fullPage: true }).catch(() => undefined);
    return {
      name: check.name,
      checkName,
      viewport: viewportVariant.name,
      viewportSize: viewportVariant.viewport,
      passed: false,
      url: check.url,
      finalURL: page.url(),
      error: error instanceof Error ? error.message : String(error),
      pageErrors,
      consoleErrors,
      ignoredConsoleErrors,
      ignoredAPIResponses,
      suppressedTelemetryRequests,
      stubbedExternalResources,
      assetFailures,
      apiFailures,
      screenshot: relative(repoRoot, screenshotFile),
    };
  } finally {
    await context.close();
  }
}

function normalizeBaseURL(value) {
  return value.replace(/\/+$/, '');
}

function joinURL(base, path) {
  return `${normalizeBaseURL(base)}${path}`;
}

function webRedirectQuery(path) {
  return `redirect=${encodeURIComponent(joinURL(webBaseURL, path))}`;
}

function toArray(value) {
  if (Array.isArray(value)) return value;
  if (typeof value === 'string') return [value];
  return [];
}

async function runCheckFlow(page, check, viewportVariant) {
  if (check.flow === 'web-login-session-refresh') {
    return runLoginSessionRefreshFlow(page, [webBaseURL], 'web');
  }
  if (check.flow === 'frontend-direct-login-session-refresh') {
    return runLoginSessionRefreshFlow(
      page,
      [frontendDirectBaseURL, webBaseURL],
      'frontend direct',
    );
  }
  if (check.flow === 'identity-portal-shell') {
    return runIdentityPortalShellFlow(page, viewportVariant);
  }
  if (check.flow === 'identity-authenticated-refresh') {
    return runIdentityAuthenticatedRefreshFlow(page, check, viewportVariant);
  }
  if (check.flow === 'identity-connect-error-refresh') {
    return runIdentityConnectErrorRefreshFlow(page, check);
  }
  return null;
}

async function runLoginSessionRefreshFlow(page, expectedFinalBaseURLs, label) {
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);
  await page.getByRole('link', { name: /登录|Login/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.pathname === '/login', { timeout: timeoutMs });
  await page.getByRole('button', { name: /SSO|统一身份/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.pathname.includes('/login/oauth/authorize'), {
    timeout: timeoutMs,
  });

  await page.getByRole('textbox', { name: /username|email|phone/i }).fill(casdoorLoginUsername);
  await page.getByRole('textbox', { name: /password/i }).fill(casdoorLoginPassword);
  await page.getByRole('button', { name: /sign in|登录/i }).click({ timeout: timeoutMs });
  await page.waitForURL(
    (url) => expectedFinalBaseURLs.some((baseURL) => url.href.startsWith(`${baseURL}/`)),
    { timeout: timeoutMs },
  );
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const beforeRefresh = await authMe(page);
  if (beforeRefresh.status !== 200) {
    throw new Error(`auth/me after login returned ${beforeRefresh.status}: ${beforeRefresh.body}`);
  }
  await expectAuthenticatedHeader(page);

  await page.reload({ waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const afterRefresh = await authMe(page);
  if (afterRefresh.status !== 200) {
    throw new Error(`auth/me after browser refresh returned ${afterRefresh.status}: ${afterRefresh.body}`);
  }
  await expectAuthenticatedHeader(page);

  return {
    matchedText: `${label} authenticated session survived refresh`,
    username: casdoorLoginUsername,
    beforeRefreshStatus: beforeRefresh.status,
    afterRefreshStatus: afterRefresh.status,
  };
}

async function runIdentityPortalShellFlow(page, viewportVariant) {
  const header = appShellHeader(page);
  if (viewportVariant.isMobile) {
    await header.locator('[aria-controls="app-mobile-nav"]').click({ timeout: timeoutMs });
    await page.locator('#app-mobile-nav').waitFor({ state: 'visible', timeout: timeoutMs });
  }

  const headerText = await header.innerText({ timeout: timeoutMs });
  const requiredLabels = ['身份中心', '个人资料', 'Connect', '授权应用', '开发者应用'];
  const missingLabels = requiredLabels.filter((label) => !headerText.includes(label));
  if (missingLabels.length > 0) {
    throw new Error(`identity header missing labels: ${missingLabels.join(', ')}`);
  }

  const forbiddenLabels = ['课程', '教师', '评课'];
  const presentForbiddenLabels = forbiddenLabels.filter((label) => headerText.includes(label));
  if (presentForbiddenLabels.length > 0) {
    throw new Error(`identity header contains main-site labels: ${presentForbiddenLabels.join(', ')}`);
  }

  const linkPaths = await header.locator('a[href]').evaluateAll((links) =>
    links.map((link) => new URL(link.href).pathname),
  );
  const requiredPaths = ['/identity', '/account/profile', '/connect', '/user/authorized-apps', '/developers/apps'];
  const missingPaths = requiredPaths.filter((path) => !linkPaths.includes(path));
  if (missingPaths.length > 0) {
    throw new Error(`identity header missing links: ${missingPaths.join(', ')}`);
  }

  const forbiddenPaths = ['/courses', '/teachers', '/courses/reviews'];
  const presentForbiddenPaths = forbiddenPaths.filter((path) => linkPaths.includes(path));
  if (presentForbiddenPaths.length > 0) {
    throw new Error(`identity header contains main-site links: ${presentForbiddenPaths.join(', ')}`);
  }

  return {
    matchedText: 'identity portal shell navigation',
    requiredLabels,
    requiredPaths,
  };
}

async function runIdentityAuthenticatedRefreshFlow(page, check, viewportVariant) {
  const targetPath = new URL(check.url).pathname;
  await page.waitForURL((url) => url.pathname === '/login', { timeout: timeoutMs });
  await page.getByRole('button', { name: /SSO|统一身份/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.pathname.includes('/login/oauth/authorize'), {
    timeout: timeoutMs,
  });

  await page.getByRole('textbox', { name: /username|email|phone/i }).fill(casdoorLoginUsername);
  await page.getByRole('textbox', { name: /password/i }).fill(casdoorLoginPassword);
  await page.getByRole('button', { name: /sign in|登录/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.href.startsWith(joinURL(identityBaseURL, targetPath)), {
    timeout: timeoutMs,
  });
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const beforeRefresh = await authMe(page);
  if (beforeRefresh.status !== 200) {
    throw new Error(`auth/me before identity refresh returned ${beforeRefresh.status}`);
  }
  const beforeHeader = await expectIdentityAuthenticatedHeader(page, viewportVariant);

  await page.reload({ waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForURL((url) => url.href.startsWith(joinURL(identityBaseURL, targetPath)), {
    timeout: timeoutMs,
  });
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const afterRefresh = await authMe(page);
  if (afterRefresh.status !== 200) {
    throw new Error(`auth/me after identity refresh returned ${afterRefresh.status}`);
  }
  const afterHeader = await expectIdentityAuthenticatedHeader(page, viewportVariant);

  return {
    matchedText: `identity ${targetPath} authenticated session survived refresh`,
    username: casdoorLoginUsername,
    beforeRefreshStatus: beforeRefresh.status,
    afterRefreshStatus: afterRefresh.status,
    beforeHeader,
    afterHeader,
  };
}

async function runIdentityConnectErrorRefreshFlow(page, check) {
  const targetPath = new URL(check.url).pathname;
  await page.waitForURL((url) => url.pathname === '/login', { timeout: timeoutMs });
  await page.getByRole('button', { name: /SSO|统一身份/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.pathname.includes('/login/oauth/authorize'), {
    timeout: timeoutMs,
  });

  await page.getByRole('textbox', { name: /username|email|phone/i }).fill(casdoorLoginUsername);
  await page.getByRole('textbox', { name: /password/i }).fill(casdoorLoginPassword);
  await page.getByRole('button', { name: /sign in|登录/i }).click({ timeout: timeoutMs });
  await page.waitForURL((url) => url.href.startsWith(joinURL(identityBaseURL, targetPath)), {
    timeout: timeoutMs,
  });
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const beforeRefresh = await authMe(page);
  if (beforeRefresh.status !== 200) {
    throw new Error(`auth/me before identity connect refresh returned ${beforeRefresh.status}`);
  }

  await page.reload({ waitUntil: 'domcontentloaded', timeout: timeoutMs });
  await page.waitForURL((url) => url.href.startsWith(joinURL(identityBaseURL, targetPath)), {
    timeout: timeoutMs,
  });
  await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => undefined);

  const afterRefresh = await authMe(page);
  if (afterRefresh.status !== 200) {
    throw new Error(`auth/me after identity connect refresh returned ${afterRefresh.status}`);
  }

  return {
    matchedText: `identity ${targetPath} connect error page survived refresh`,
    username: casdoorLoginUsername,
    beforeRefreshStatus: beforeRefresh.status,
    afterRefreshStatus: afterRefresh.status,
  };
}

async function expectIdentityAuthenticatedHeader(page, viewportVariant) {
  await runIdentityPortalShellFlow(page, viewportVariant);

  const header = appShellHeader(page);
  const notificationBellCount = await header.locator('.notification-bell').count();
  if (notificationBellCount > 0) {
    throw new Error('identity header should not render the main-site notification bell');
  }

  const userMenuButton = header.getByRole('button', { name: /用户|User/i });
  await userMenuButton.waitFor({ state: 'visible', timeout: timeoutMs });
  await userMenuButton.click({ timeout: timeoutMs });
  const userMenu = page.getByRole('menu', { name: /用户|User/i });
  await userMenu.waitFor({ state: 'visible', timeout: timeoutMs });
  const menuText = await userMenu.innerText({ timeout: timeoutMs });

  if (!/身份中心|Identity Hub/i.test(menuText)) {
    throw new Error(`identity user menu does not expose identity home: ${menuText}`);
  }
  if (/个人中心|Profile/i.test(menuText)) {
    throw new Error(`identity user menu exposes main-site profile entry: ${menuText}`);
  }

  await page.keyboard.press('Escape').catch(() => undefined);

  return {
    matchedText: 'identity authenticated header actions',
    notificationBellCount,
  };
}

function appShellHeader(page) {
  return page.getByRole('banner').first();
}

async function authMe(page) {
  return page.evaluate(async () => {
    const response = await fetch('/api/v1/auth/me', { credentials: 'include' });
    return {
      status: response.status,
      body: await response.text(),
    };
  });
}

async function expectAuthenticatedHeader(page) {
  const header = appShellHeader(page);
  const headerText = await header.innerText({ timeout: timeoutMs });
  if (/登录|Login/i.test(headerText)) {
    throw new Error(`header still shows login after authentication: ${headerText}`);
  }

  const userMenuButton = header.getByRole('button', { name: /用户|User/i });
  await userMenuButton.waitFor({ state: 'visible', timeout: timeoutMs });
  await userMenuButton.click({ timeout: timeoutMs });
  const userMenu = page.getByRole('menu', { name: /用户|User/i });
  await userMenu.waitFor({ state: 'visible', timeout: timeoutMs });
  await userMenu.getByRole('menuitem', { name: /退出登录|Logout/i }).waitFor({
    state: 'visible',
    timeout: timeoutMs,
  });
  await page.keyboard.press('Escape').catch(() => undefined);
}

function resourceStubsForCheck(check) {
  if (!Array.isArray(check.stubbedResources)) return [];
  return check.stubbedResources.filter((resourceStub) => (
    resourceStub &&
    typeof resourceStub.url === 'string' &&
    resourceStub.url.length > 0
  ));
}

function isAllowedAPIResponse(check, response) {
  const rules = Array.isArray(check.allowedAPIResponses) ? check.allowedAPIResponses : [];
  return rules.some((rule) => {
    const statuses = Array.isArray(rule.statuses) ? rule.statuses : [];
    return (
      statuses.includes(response.status()) &&
      typeof rule.urlIncludes === 'string' &&
      response.url().includes(rule.urlIncludes)
    );
  });
}

function describeConsoleMessage(message) {
  const location = message.location();
  const locationText =
    location.url && location.lineNumber > 0
      ? ` (${location.url}:${location.lineNumber}:${location.columnNumber})`
      : '';
  return `${message.text()}${locationText}`;
}

function isIgnoredConsoleError(text) {
  return (
    /^Failed to load resource: the server responded with a status of [45]\d\d \([^)]*\)$/.test(
      text,
    ) ||
    text.includes('The Cross-Origin-Opener-Policy header has been ignored')
  );
}
