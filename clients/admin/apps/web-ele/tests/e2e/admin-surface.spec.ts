import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-05-24T04:00:00Z';

const capabilities = [
  'admin:dashboard:view',
  'admin:reviews:manage',
  'admin:reports:manage',
  'admin:teachers:manage',
  'admin:sensitive_words:manage',
  'admin:logs:view',
  'user:identity:read',
  'user:identity:review',
  'user:student:review',
  'user:school:read',
  'user:school:update',
  'user:system:read',
  'user:system:update',
  'admission:freshman:review',
  'admission:policy:update',
  'member_blacklist:read',
  'member_blacklist:manage',
  'open_platform:manage',
];

const adminUser = {
  id: '99',
  name: 'platform-admin',
  displayName: 'Platform Admin',
  email: 'admin@example.com',
  avatar: '',
  roles: ['platform_admin'],
  capabilities,
  globalCapabilities: capabilities,
  capabilityGrants: capabilities.map((name) => ({
    name,
    global: true,
    scopeRoles: [],
    scopeSchoolIDs: [],
    scopeSectionIDs: [],
  })),
  canAccessAdmin: true,
  isPlatformAdmin: true,
};

function json(data: unknown, status = 200) {
  return {
    status,
    contentType: 'application/json',
    body: JSON.stringify(data),
  };
}

function ok(data: unknown) {
  return json({ success: true, data });
}

function list<T>(items: T[]) {
  return ok({ list: items, total: items.length });
}

const adminApiRequests: string[] = [];

function hasAdminGetRequest(pathname: string, matches: (url: URL) => boolean) {
  return adminApiRequests.some((request) => {
    if (!request.startsWith('GET ')) return false;
    const url = new URL(request.slice('GET '.length), 'http://admin.e2e');
    return url.pathname === pathname && matches(url);
  });
}

async function waitForAdminGetRequest(
  page: Page,
  pathname: string,
  matches: (url: URL) => boolean,
  trigger: () => Promise<void>,
) {
  const responsePromise = page.waitForResponse((response) => {
    const request = response.request();
    const url = new URL(response.url());
    return (
      request.method() === 'GET' &&
      url.pathname === pathname &&
      matches(url) &&
      response.status() < 400
    );
  });

  await trigger();
  await responsePromise;

  await expect.poll(() => hasAdminGetRequest(pathname, matches)).toBe(true);
}

const stats = {
  totalReviews: 128,
  todayReviews: 7,
  weekReviews: 31,
  publishedReviews: 113,
  hiddenReviews: 4,
  deletedReviews: 11,
  totalReports: 18,
  pendingReports: 3,
};

const review = {
  id: 'review-101',
  courseID: 42,
  courseName: '数据结构与算法',
  teacherName: '王老师',
  title: '期末项目清晰',
  content: '课程资料完整，评分标准明确。',
  ratings: { recommendation: 5, clarity: 4 },
  status: 'published',
  createdAt: now,
};

const report = {
  id: 'report-9',
  reviewID: 'review-101',
  review,
  reason: 'spam',
  description: '疑似广告内容',
  resolutionNote: null,
  status: 'pending',
  createdAt: now,
};

const teacher = {
  id: 7,
  name: '李教授',
  departmentID: 1,
  departmentName: '计算机学院',
  reviewCount: 12,
  createdAt: now,
};

const sensitiveWord = {
  id: 'word-1',
  word: '违规词',
  category: 'review',
  level: 'block',
  isActive: true,
  createdAt: now,
  updatedAt: now,
};

const operationLog = {
  id: 'log-1',
  adminUserID: '99',
  adminUsername: 'platform-admin',
  action: 'review.hide',
  resourceType: 'review',
  resourceID: 'review-101',
  oldValue: { status: 'published' },
  newValue: { status: 'hidden' },
  ipAddress: '127.0.0.1',
  userAgent: 'Playwright',
  createdAt: now,
};

const identityReview = {
  userID: 12,
  realName: '张三',
  docType: 'MAINLAND_ID',
  verifyMethod: 'manual',
  verified: false,
  verifiedAt: null,
  reviewedAt: null,
  createdAt: now,
  updatedAt: now,
};

const studentVerification = {
  userID: 13,
  schoolID: 1001,
  activeStudentID: '20260001',
  verificationStatus: 'pending',
  verificationMethod: 'manual',
  createdAt: now,
  updatedAt: now,
};

const schoolConfig = {
  schoolID: 1001,
  schoolName: '测试大学',
  verificationMethod: 'ldap',
  enabled: true,
  academicDbTable: 'academic_students',
  consentText: '仅用于学生身份认证',
  ldapConfig: {
    url: 'ldap://ldap.example.com',
    baseDN: 'dc=example,dc=com',
    systemBindDN: 'cn=reader,dc=example,dc=com',
    useTLS: true,
  },
};

const freshmanApplication = {
  id: 'freshman-1',
  status: 'pending',
  schoolID: 1001,
  qqID: '10001',
  applicantNameMasked: '赵*',
  materialURL: 'https://example.com/material.jpg',
  failureCount: 1,
  createdAt: now,
};

const admissionPolicy = {
  id: 'policy-qq-1',
  platform: 'qq',
  guildID: 'guild-1',
  freshmanChannelEnabled: true,
  freshmanChannelClosesAt: '2026-09-01T00:00:00Z',
  freshmanDefaultExpiresAt: '2026-10-01T00:00:00Z',
  initialMuteDurationSeconds: 60,
  linkWaitSeconds: 300,
  submissionWaitSeconds: 600,
  manualReviewTimeoutSeconds: 3600,
  reminderIntervalSeconds: 120,
  failedJoinLimit: 3,
  blacklistDurationSeconds: 86_400,
  maxMaterialBytes: 5_242_880,
  maxExtensionDays: 30,
  managementGuildIDs: ['guild-admin'],
  forwardRawMaterialToQQ: false,
};

const blacklistEntry = {
  id: 'entry-active',
  platform: 'qq',
  subjectType: 'qq_user',
  subjectID: '10001',
  scopeType: 'guild',
  guildID: 'guild-1',
  source: 'admission_failure',
  reasonCode: 'admission_timeout_limit',
  reasonText: 'too many failures',
  createdFrom: 'admin_console',
  createdByType: 'admin_user',
  createdByID: 'admin-1',
  createdAt: now,
  updatedAt: now,
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
};

const openPlatformApp = {
  app: {
    id: 42,
    clientID: 'campus-connector',
    displayName: 'Campus Connector',
    description: 'Campus integration client',
    homepageURL: 'https://connector.example.com',
    privacyPolicyURL: 'https://connector.example.com/privacy',
    redirectURIs: ['https://connector.example.com/callback'],
    status: 'approved',
    createdAt: now,
    updatedAt: now,
  },
  scopes: [
    {
      id: 1,
      scope: 'profile.basic.read',
      displayName: 'Basic profile',
      sensitivity: 'low',
      fields: ['name', 'avatar'],
      reason: 'Show the signed-in user',
      status: 'approved',
      reviewerUserID: 99,
      reviewedAt: now,
      decisionNote: 'standard profile access',
      createdAt: now,
      updatedAt: now,
    },
  ],
  redirectURIRequests: [],
};

const auditEvent = {
  id: 88,
  appID: 42,
  userID: 12,
  eventType: 'open_platform.scope.approved',
  scope: 'profile.basic.read',
  requestID: 'req-admin-surface',
  metadata: { source: 'e2e' },
  createdAt: now,
};

const consent = {
  userID: 12,
  app: openPlatformApp.app,
  scopes: [
    {
      scope: 'profile.basic.read',
      displayName: 'Basic profile',
      grantedAt: now,
      lastUsedAt: now,
    },
  ],
};

const tokenProbeEvidence = {
  id: 77,
  appID: 42,
  reviewerUserID: 99,
  clientID: 'campus-connector',
  redirectURI: 'https://connector.example.com/callback',
  result: 'passed',
  probeMethod: 'authorization_code',
  inspectedClaims: ['sub', 'aud'],
  businessClaims: [],
  error: null,
  tokenClaims: { sub: ['user-1'] },
  requestID: 'req-token-probe',
  metadata: { nonceVerified: true },
  createdAt: now,
};

const disclosureReport = {
  summary: {
    total: 10,
    granted: 8,
    denied: 1,
    rateLimited: 1,
    replayDetected: 1,
    windowHours: 24,
  },
  endpoints: [
    {
      endpoint: '/oidc/userinfo',
      total: 10,
      granted: 8,
      denied: 1,
      rateLimited: 1,
      replayDetected: 1,
    },
  ],
  denialReasons: [{ reason: 'fga_denied', total: 1 }],
  rateLimitDimensions: [{ dimension: 'app:42', total: 1 }],
  recentReplayEvents: [
    {
      id: 'replay-1',
      detectedAt: now,
      appID: 42,
      userID: 12,
      endpoint: '/oidc/userinfo',
      result: 'replay_detected',
      count: 1,
      scopes: ['profile.basic.read'],
      metadata: { nonce: 'redacted' },
    },
  ],
};

async function mockAdminApi(page: Page) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    adminApiRequests.push(`${method} ${path}${url.search}`);

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }

    if (path === '/api/v1/course/review/admin/stats') {
      await route.fulfill(ok(stats));
      return;
    }

    if (path === '/api/v1/course/review/admin/reviews') {
      await route.fulfill(list([review]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reviews/')) {
      await route.fulfill(ok(review));
      return;
    }

    if (path === '/api/v1/course/review/admin/reports') {
      await route.fulfill(list([report]));
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/reports/')) {
      await route.fulfill(ok({ ...report, status: 'resolved' }));
      return;
    }

    if (path === '/api/v1/course/review/admin/teachers') {
      await route.fulfill(
        method === 'POST' ? ok({ ...teacher, id: 8 }) : list([teacher]),
      );
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/teachers/')) {
      await route.fulfill(ok(teacher));
      return;
    }

    if (path === '/api/v1/course/review/admin/sensitive-words') {
      await route.fulfill(
        method === 'POST'
          ? ok({ ...sensitiveWord, id: 'word-2' })
          : list([sensitiveWord]),
      );
      return;
    }
    if (path.startsWith('/api/v1/course/review/admin/sensitive-words/')) {
      await route.fulfill(ok(sensitiveWord));
      return;
    }

    if (path === '/api/v1/course/review/admin/logs') {
      const pageNumber = Number(url.searchParams.get('page') ?? '1');
      await route.fulfill(
        ok({
          list: [
            {
              ...operationLog,
              id: `log-${pageNumber}`,
            },
          ],
          total: 40,
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/identities') {
      await route.fulfill(list([identityReview]));
      return;
    }
    if (path.startsWith('/api/v1/admin/identities/')) {
      await route.fulfill(ok({ ...identityReview, verified: true }));
      return;
    }

    if (path === '/api/v1/admin/student-verifications') {
      await route.fulfill(list([studentVerification]));
      return;
    }
    if (path.startsWith('/api/v1/admin/student-verifications/')) {
      await route.fulfill(
        ok({ ...studentVerification, verificationStatus: 'verified' }),
      );
      return;
    }

    if (path === '/api/v1/admin/school-configs') {
      await route.fulfill(ok([schoolConfig]));
      return;
    }
    if (path.startsWith('/api/v1/admin/school-configs/')) {
      await route.fulfill(ok(schoolConfig));
      return;
    }

    if (path === '/api/v1/admin/system-configs') {
      await route.fulfill(
        ok([
          {
            key: 'review.retention_days',
            value: '365',
            description: '评课保留天数',
            updatedAt: now,
          },
        ]),
      );
      return;
    }
    if (path.startsWith('/api/v1/admin/system-configs/')) {
      await route.fulfill(ok({ key: 'review.retention_days', value: '180' }));
      return;
    }

    if (path === '/api/v1/admin/freshman-verifications') {
      await route.fulfill(list([freshmanApplication]));
      return;
    }
    if (path.startsWith('/api/v1/admin/freshman-verifications/')) {
      await route.fulfill(ok({ ...freshmanApplication, status: 'approved' }));
      return;
    }

    if (path === '/api/v1/admin/admission/policies') {
      await route.fulfill(ok([admissionPolicy]));
      return;
    }
    if (path.startsWith('/api/v1/admin/admission/policies/')) {
      await route.fulfill(ok(admissionPolicy));
      return;
    }

    if (path === '/api/v1/admin/member-blacklist') {
      await route.fulfill(
        method === 'POST'
          ? ok({ ...blacklistEntry, id: 'entry-created' })
          : list([blacklistEntry]),
      );
      return;
    }
    if (
      path.includes('/member-blacklist/') ||
      path.endsWith('/release-by-subject')
    ) {
      await route.fulfill(ok({ ...blacklistEntry, releasedAt: now }));
      return;
    }

    if (path === '/api/v1/admin/open-platform/apps') {
      await route.fulfill(list([openPlatformApp]));
      return;
    }
    if (path.includes('/api/v1/admin/open-platform/apps/')) {
      await route.fulfill(ok(openPlatformApp));
      return;
    }

    if (path === '/api/v1/admin/open-platform/audit-events') {
      await route.fulfill(list([auditEvent]));
      return;
    }

    if (path === '/api/v1/admin/open-platform/consents') {
      await route.fulfill(list([consent]));
      return;
    }

    if (path === '/api/v1/admin/open-platform/token-probe-evidence') {
      await route.fulfill(list([tokenProbeEvidence]));
      return;
    }

    if (path === '/api/v1/admin/open-platform/disclosure-report') {
      await route.fulfill(ok(disclosureReport));
      return;
    }

    await fulfillUnexpected(route, path, method);
  });
}

async function fulfillUnexpected(route: Route, path: string, method: string) {
  await route.fulfill(
    json(
      {
        success: false,
        error: {
          code: 'E2E_UNMOCKED',
          message: `unmocked admin e2e request: ${method} ${path}`,
        },
      },
      500,
    ),
  );
}

test.describe('Admin management surfaces', () => {
  test.beforeEach(async ({ page }) => {
    adminApiRequests.length = 0;
    await mockAdminApi(page);
  });

  test('dashboard pages render live moderation statistics', async ({
    page,
  }) => {
    await page.goto('/analytics');
    await expect(
      page.getByRole('heading', { name: /分析页|Analytics/ }),
    ).toBeVisible();
    await expect(page.getByText('128')).toBeVisible();
    await expect(
      page
        .locator('.admin-dashboard__kpi')
        .filter({ hasText: '待处理举报' })
        .getByText('3', { exact: true }),
    ).toBeVisible();

    await page.goto('/workspace');
    await expect(page.getByText('处理队列')).toBeVisible();
    await expect(page.getByText('评课总量')).toBeVisible();
    await expect(page.getByText('待处理举报')).toBeVisible();
  });

  test('content management pages render review operations data', async ({
    page,
  }) => {
    await page.goto('/content/reviews');
    await expect(page.getByText('数据结构与算法').first()).toBeVisible();
    await expect(page.getByText('期末项目清晰')).toBeVisible();

    await page.goto('/content/reports');
    await expect(page.getByText('疑似广告内容')).toBeVisible();
    await expect(page.getByText('期末项目清晰')).toBeVisible();

    await page.goto('/content/teachers');
    await expect(page.getByText('李教授')).toBeVisible();
    await expect(page.getByText('计算机学院')).toBeVisible();

    await page.goto('/content/sensitive-words');
    await expect(page.getByText('违规词')).toBeVisible();
    await expect(page.getByText('review')).toBeVisible();

    await page.goto('/content/logs');
    await expect(page.getByText('platform-admin')).toBeVisible();
    await expect(page.getByText('review.hide')).toBeVisible();
  });

  test('operation logs pagination requests the next page with current page size', async ({
    page,
  }) => {
    await page.goto('/content/logs');
    await expect(page.getByText('platform-admin')).toBeVisible();

    await page.locator('.el-pagination .btn-next').click();

    await expect
      .poll(() =>
        adminApiRequests.some((request) => {
          const url = new URL(request.slice('GET '.length), 'http://admin.e2e');
          return (
            url.pathname === '/api/v1/course/review/admin/logs' &&
            url.searchParams.get('page') === '2' &&
            url.searchParams.get('pageSize') === '20'
          );
        }),
      )
      .toBe(true);
  });

  test('content filters pass status, category, and level query params', async ({
    page,
  }) => {
    await page.goto('/content/reviews');
    await waitForAdminGetRequest(
      page,
      '/api/v1/course/review/admin/reviews',
      (url) =>
        url.searchParams.get('status') === 'pending_review' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '待审核' }).click();
      },
    );

    await page.goto('/content/reports');
    await waitForAdminGetRequest(
      page,
      '/api/v1/course/review/admin/reports',
      (url) =>
        url.searchParams.get('status') === 'resolved' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '已处理' }).click();
      },
    );

    await page.goto('/content/sensitive-words');
    await page.getByPlaceholder('按分类筛选...').fill('comment');
    await waitForAdminGetRequest(
      page,
      '/api/v1/course/review/admin/sensitive-words',
      (url) =>
        url.searchParams.get('category') === 'comment' &&
        url.searchParams.get('level') === 'review' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '复核' }).click();
      },
    );
  });

  test('teacher and app status filters pass query params', async ({ page }) => {
    await page.goto('/content/teachers');
    await page.getByPlaceholder('搜索教师姓名...').fill('李教授');
    await page.getByPlaceholder('按院系 ID 筛选...').fill('1');
    await waitForAdminGetRequest(
      page,
      '/api/v1/course/review/admin/teachers',
      (url) =>
        url.searchParams.get('search') === '李教授' &&
        url.searchParams.get('departmentID') === '1' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('button', { name: '查询' }).click();
      },
    );

    await page.goto('/open-platform/apps');
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/open-platform/apps',
      (url) =>
        url.searchParams.get('status') === 'approved' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page
          .getByRole('main')
          .locator('.el-select.admin-toolbar-control')
          .click();
        await page.getByRole('option', { name: '已批准' }).click();
      },
    );
  });

  test('user system pages render identity, admission, and blacklist data', async ({
    page,
  }) => {
    await page.goto('/users/identity-review');
    await expect(page.getByText('张三')).toBeVisible();

    await page.goto('/users/student-verification');
    await expect(page.getByText('20260001')).toBeVisible();

    await page.goto('/users/school-config');
    await expect(page.getByText('测试大学')).toBeVisible();
    await expect(page.getByText('ldap://ldap.example.com')).toBeVisible();

    await page.goto('/users/freshman-verification');
    await expect(page.getByText('赵*')).toBeVisible();
    await expect(page.getByText('10001').first()).toBeVisible();

    await page.goto('/users/admission-policy');
    await expect(
      page.getByRole('heading', { name: '入群认证策略' }),
    ).toBeVisible();
    await expect(page.getByText('QQ 群 guild-1')).toBeVisible();

    await page.goto('/users/member-blacklist');
    await expect(
      page.getByRole('heading', { name: '成员黑名单' }),
    ).toBeVisible();
    await expect(page.getByText('too many failures')).toBeVisible();

    await page.goto('/users/system-config');
    await expect(page.getByText('review.retention_days')).toBeVisible();
    await expect(page.getByText('评课保留天数')).toBeVisible();
  });

  test('user system filters pass review and blacklist query params', async ({
    page,
  }) => {
    await page.goto('/users/identity-review');
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/identities',
      (url) =>
        url.searchParams.get('status') === 'verified' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '已认证' }).click();
      },
    );

    await page.goto('/users/student-verification');
    await page.getByPlaceholder('按学校ID筛选...').fill('1001');
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/student-verifications',
      (url) =>
        url.searchParams.get('status') === 'verified' &&
        url.searchParams.get('schoolID') === '1001' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '已认证' }).click();
      },
    );

    await page.goto('/users/freshman-verification');
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/freshman-verifications',
      (url) =>
        url.searchParams.get('status') === 'rejected' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('main').locator('.el-select').click();
        await page.getByRole('option', { name: '已驳回' }).click();
      },
    );

    await page.goto('/users/member-blacklist');
    await page.getByPlaceholder('QQ / 主体 ID').fill('10001');
    await page.getByPlaceholder('群号').fill('guild-filter');
    await page.getByPlaceholder('平台').fill('qq');
    await page.locator('.el-select[data-field="scopeType"]').click();
    await page.getByRole('option', { name: '单群' }).click();
    await page.locator('.el-select[data-field="source"]').click();
    await page.getByRole('option', { name: '审核处置' }).click();
    await page.locator('.el-select[data-field="status"]').click();
    await page.getByRole('option', { name: '已解除' }).click();
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/member-blacklist',
      (url) =>
        url.searchParams.get('subjectID') === '10001' &&
        url.searchParams.get('guildID') === 'guild-filter' &&
        url.searchParams.get('platform') === 'qq' &&
        url.searchParams.get('scopeType') === 'guild' &&
        url.searchParams.get('source') === 'moderation_action' &&
        url.searchParams.get('status') === 'released' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('button', { name: '查询' }).click();
      },
    );
  });

  test('open platform admin pages render app review and audit evidence', async ({
    page,
  }) => {
    await page.goto('/open-platform/apps');
    await expect(page.getByText('Campus Connector').first()).toBeVisible();
    await expect(page.getByText('campus-connector').first()).toBeVisible();

    await page.goto('/open-platform/audit-events');
    await expect(page.getByText('open_platform.scope.approved')).toBeVisible();
    await expect(page.getByText('req-admin-surface')).toBeVisible();

    await page.goto('/open-platform/consents?appID=42');
    await page.getByRole('spinbutton').first().fill('42');
    await page.getByRole('button', { name: /查询|Query/ }).click();
    await expect(page.getByText('Campus Connector').first()).toBeVisible();
    await expect(page.getByText('profile.basic.read').first()).toBeVisible();

    await page.goto('/open-platform/token-probe-evidence');
    await expect(page.getByText('campus-connector').first()).toBeVisible();
    await expect(page.getByText('authorization_code')).toBeVisible();

    await page.goto('/open-platform/disclosure-report');
    await expect(
      page.getByRole('row', { name: /\/oidc\/userinfo\s+10\s+8/ }),
    ).toBeVisible();
    await expect(page.getByRole('row', { name: /fga_denied/ })).toBeVisible();
  });

  test('open platform audit and consent filters pass query params', async ({
    page,
  }) => {
    await page.goto('/open-platform/audit-events');
    await page.getByPlaceholder('按应用 ID').fill('42');
    await page.getByPlaceholder('按用户 ID').fill('12');
    await page
      .getByRole('main')
      .locator('.el-select.admin-toolbar-control--wide')
      .first()
      .click();
    await page.getByRole('option', { name: '用户授权' }).click();
    await page
      .getByRole('main')
      .locator('.el-select.admin-toolbar-control--wide')
      .nth(1)
      .click();
    await page.getByRole('option', { name: 'email.read' }).click();
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/open-platform/audit-events',
      (url) =>
        url.searchParams.get('appID') === '42' &&
        url.searchParams.get('userID') === '12' &&
        url.searchParams.get('eventType') === 'open_platform.consent.granted' &&
        url.searchParams.get('scope') === 'email.read' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('button', { name: '查询' }).click();
      },
    );

    await page.goto('/open-platform/consents');
    await page.getByPlaceholder('按应用 ID').fill('42');
    await page.getByPlaceholder('按用户 ID').fill('12');
    await waitForAdminGetRequest(
      page,
      '/api/v1/admin/open-platform/consents',
      (url) =>
        url.searchParams.get('appID') === '42' &&
        url.searchParams.get('userID') === '12' &&
        url.searchParams.get('page') === '1' &&
        url.searchParams.get('pageSize') === '20',
      async () => {
        await page.getByRole('button', { name: '查询' }).click();
      },
    );
  });

  test('open platform token probe evidence filters request by app, reviewer, result, and client', async ({
    page,
  }) => {
    await page.goto('/open-platform/token-probe-evidence');

    await page.getByPlaceholder('按应用 ID').fill('42');
    await page.getByPlaceholder('按审核人 ID').fill('99');
    await page.getByRole('main').locator('.el-select').click();
    await page.getByRole('option', { name: '失败' }).click();
    await page.getByPlaceholder('按 Client ID').fill('campus-filter');
    await page.getByRole('button', { name: '查询' }).click();

    await expect
      .poll(() =>
        adminApiRequests.some((request) => {
          const url = new URL(request.slice('GET '.length), 'http://admin.e2e');
          return (
            url.pathname ===
              '/api/v1/admin/open-platform/token-probe-evidence' &&
            url.searchParams.get('appID') === '42' &&
            url.searchParams.get('reviewerUserID') === '99' &&
            url.searchParams.get('result') === 'failed' &&
            url.searchParams.get('clientID') === 'campus-filter' &&
            url.searchParams.get('page') === '1' &&
            url.searchParams.get('pageSize') === '20'
          );
        }),
      )
      .toBe(true);
  });

  test('open platform disclosure report query updates the reporting window', async ({
    page,
  }) => {
    await page.goto('/open-platform/disclosure-report');

    await page.getByPlaceholder('统计窗口（小时）').fill('6');
    await page.getByRole('button', { name: '查询' }).click();

    await expect
      .poll(() =>
        adminApiRequests.some((request) => {
          const url = new URL(request.slice('GET '.length), 'http://admin.e2e');
          return (
            url.pathname === '/api/v1/admin/open-platform/disclosure-report' &&
            url.searchParams.get('windowHours') === '6'
          );
        }),
      )
      .toBe(true);
  });
});
