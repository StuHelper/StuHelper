import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const now = '2026-08-05T08:00:00Z';
const schoolCode = '4111010006';

const capabilities = [
  'student:manual_review:read',
  'student:manual_review:decide',
  'student:manual_material:access',
  'student:verification_config:read',
  'student:verification_config:update',
  'student:roster:read',
  'student:roster:activate',
  'student:credential:read',
  'student:credential:revoke',
  'student:subject_conflict:read',
  'student:subject_conflict:resolve',
  'campus_connector:health:read',
  'campus_connector:manage',
];

const adminUser = {
  id: '99',
  name: 'platform-admin',
  displayName: 'Platform Admin',
  email: 'admin@example.com',
  avatar: '',
  roles: ['super_admin'],
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

const method = {
  method: 'real_name_identity_check',
  displayName: '实名信息校验',
  description: '由服务端校验提交信息是否一致。',
  adapterID: 'buaa',
  adapterVersion: 'v1',
  rosterDependency: 'required',
  conditionalPolicy: {},
  publicFormSchema: {},
  riskPolicy: {},
  credentialTTLSeconds: 31_536_000,
  connectorOperationKey: null,
  secretConfigured: false,
  privacyNoticeVersion: '2026-08-05',
  privacyNotice: { summary: '仅用于本次学生身份校验。' },
  enabled: true,
  validationStatus: 'valid',
  validationCode: null,
  healthStatus: 'healthy',
  healthCode: null,
  healthCheckedAt: now,
  configRevision: 3,
  updatedAt: now,
};

const school = {
  schoolCode,
  schoolName: '北京航空航天大学',
  location: '北京',
  adapterID: 'buaa',
  adapterVersion: 'v1',
  emailDomains: ['buaa.edu.cn'],
  studentIDPolicy: { kind: 'buaa_code_adapter' },
  nameMatchPolicy: { normalizeWhitespace: true },
  enrollmentPolicy: { currentMarker: '1' },
  manualFormSchema: { version: 1 },
  snapshotSyncIntervalSeconds: 604_800,
  snapshotWarningAfterSeconds: 691_200,
  snapshotHardExpirySeconds: 1_209_600,
  snapshotGraceSeconds: 0,
  snapshotAutoActivate: false,
  enabled: true,
  validationStatus: 'valid',
  validationCode: null,
  configRevision: 5,
  methods: [method],
  updatedAt: now,
};

const snapshot = {
  id: '00000000-0000-4000-8000-000000000101',
  schoolCode,
  schoolName: school.schoolName,
  sourceKind: 'campus_connector',
  sourceVersion: 'oracle-full-20260805T080000Z',
  importMode: 'full',
  schemaVersion: 1,
  mappingVersion: 'buaa-v1',
  status: 'active',
  sourceCutoffAt: now,
  importStartedAt: now,
  importCompletedAt: now,
  activatedAt: now,
  rowCount: 31_204,
  eligibleRowCount: 30_981,
  deletedRowCount: 0,
  checksum: 'a'.repeat(64),
  failureCode: '',
  activationRevision: 7,
  isCurrent: true,
  qualityChecks: [
    {
      checkName: 'row_count_bounds',
      status: 'passed',
      observedValue: { rowCount: 31_204 },
      detailCode: '',
      checkedAt: now,
    },
  ],
  createdAt: now,
  updatedAt: now,
};

const connector = {
  id: '00000000-0000-4000-8000-000000000201',
  displayName: 'BUAA campus connector',
  status: 'active',
  protocolVersion: 'v1',
  softwareVersion: '2026.08.05',
  certificateNotAfter: '2027-08-05T00:00:00Z',
  maxConcurrency: 4,
  heartbeatIntervalSeconds: 30,
  lastHeartbeatAt: now,
  lastHealthCode: null,
  revision: 2,
  operations: [
    {
      schoolCode,
      operationKey: 'buaa.roster.full_snapshot',
      operationType: 'roster_snapshot_upload',
      adapterID: 'buaa_oracle_roster',
      adapterVersion: 'v1',
      enabled: true,
      validationStatus: 'valid',
      healthStatus: 'healthy',
      healthCode: null,
      healthCheckedAt: now,
      configRevision: 2,
    },
  ],
};

const rosterSyncRequest = {
  id: '00000000-0000-4000-8000-000000000202',
  schoolCode,
  operationKey: 'buaa.roster.full_snapshot',
  adapterID: 'buaa_oracle_roster',
  adapterVersion: 'v1',
  status: 'pending',
  resultCode: null,
  requestedByUserID: 99,
  reason: '上线前完整快照复核',
  deadlineAt: '2026-08-06T08:00:00Z',
  claimedAt: null,
  claimAttempts: 0,
  completedAt: null,
  latencyMilliseconds: null,
  createdAt: now,
  updatedAt: now,
};

const material = {
  id: '00000000-0000-4000-8000-000000000301',
  contentType: 'image/jpeg',
  sizeBytes: 220_000,
  width: 1600,
  height: 1200,
  capturedAt: now,
};

const manualCase = {
  id: '00000000-0000-4000-8000-000000000302',
  applicationID: '00000000-0000-4000-8000-000000000303',
  school: { code: schoolCode, name: school.schoolName },
  status: 'pending',
  materialType: 'student_card',
  applicantNameMasked: '张*',
  studentIDMasked: '20****01',
  emailMasked: '20****01@buaa.edu.cn',
  emailVerified: true,
  emailVerificationRequired: false,
  materials: [material],
  userVisibleReason: null,
  credentialClass: null,
  credentialExpiresAt: null,
  revision: 2,
  nextActions: ['wait_for_review'],
  submittedAt: now,
  reviewedAt: null,
  createdAt: now,
  updatedAt: now,
};

const credential = {
  id: '00000000-0000-4000-8000-000000000401',
  userID: 42,
  schoolCode,
  schoolName: school.schoolName,
  method: 'real_name_identity_check',
  status: 'active',
  credentialClass: 'formal_student',
  subjectDisplay: '20****01',
  rosterDependency: 'required',
  assurance: 'standard',
  verifiedAt: now,
  expiresAt: '2027-08-05T08:00:00Z',
  reviewRequiredAt: null,
  revokedReason: null,
  revision: 4,
};

const conflict = {
  id: '00000000-0000-4000-8000-000000000402',
  schoolCode,
  schoolName: school.schoolName,
  claimantUserID: 43,
  incumbentUserID: 42,
  applicationID: '00000000-0000-4000-8000-000000000403',
  status: 'open',
  reasonCode: 'subject_already_bound',
  resolutionCode: null,
  resolvedAt: null,
  createdAt: now,
  updatedAt: now,
};

interface CapturedMutation {
  body: unknown;
  method: string;
  path: string;
}

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

function parseJsonBody(route: Route) {
  const postData = route.request().postData();
  return postData ? (JSON.parse(postData) as unknown) : null;
}

async function mockAdminApi(page: Page, captured: CapturedMutation[]) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const methodName = request.method();

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }
    if (path === '/api/v1/admin/student-verification/schools') {
      await route.fulfill(ok([school]));
      return;
    }
    if (
      path ===
      `/api/v1/admin/student-verification/schools/${schoolCode}/roster-snapshots`
    ) {
      await route.fulfill(ok([snapshot]));
      return;
    }
    if (path === '/api/v1/admin/student-verification/connectors') {
      await route.fulfill(ok([connector]));
      return;
    }
    if (
      path ===
      `/api/v1/admin/student-verification/schools/${schoolCode}/roster-sync-requests`
    ) {
      if (methodName === 'POST') {
        captured.push({ path, method: methodName, body: parseJsonBody(route) });
        await route.fulfill(
          ok({
            ...rosterSyncRequest,
            reason: (
              parseJsonBody(route) as {
                reason: string;
              }
            ).reason,
          }),
        );
        return;
      }
      await route.fulfill(ok([]));
      return;
    }
    if (path === '/api/v1/admin/student-verification/manual-reviews') {
      await route.fulfill(ok([manualCase]));
      return;
    }
    if (
      path ===
      `/api/v1/admin/student-verification/manual-reviews/${manualCase.id}`
    ) {
      await route.fulfill(
        ok({
          case: manualCase,
          formValues: {
            college: '计算机学院',
            studentID: '20990001',
          },
        }),
      );
      return;
    }
    if (
      path ===
      `/api/v1/admin/student-verification/manual-reviews/${manualCase.id}/materials/${material.id}/access`
    ) {
      await route.fulfill(
        ok({
          url: 'https://materials.example.invalid/audited-preview',
          expiresAt: '2026-08-05T08:05:00Z',
        }),
      );
      return;
    }
    if (
      path ===
      `/api/v1/admin/student-verification/manual-reviews/${manualCase.id}/decision`
    ) {
      captured.push({ path, method: methodName, body: parseJsonBody(route) });
      await route.fulfill(ok({ ...manualCase, status: 'approved' }));
      return;
    }
    if (path === '/api/v1/admin/student-verification/credentials') {
      await route.fulfill(ok([credential]));
      return;
    }
    if (path === '/api/v1/admin/student-verification/subject-conflicts') {
      await route.fulfill(ok([conflict]));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked student verification admin request: ${methodName} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin student verification platform', () => {
  let captured: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    captured = [];
    await mockAdminApi(page, captured);
  });

  test('school control plane shows independent methods, active snapshot, and connector health', async ({
    page,
  }) => {
    await page.goto('/users/school-config');

    await expect(page.getByText(school.schoolName).first()).toBeVisible();
    await expect(page.getByText('实名信息校验')).toBeVisible();
    await expect(page.getByText('buaa@v1').first()).toBeVisible();

    await page.getByRole('tab', { name: '学籍快照' }).click();
    await expect(page.getByText(snapshot.sourceVersion)).toBeVisible();
    await expect(page.getByText('31204 / 30981')).toBeVisible();

    await page.getByRole('tab', { name: '连接器健康' }).click();
    await expect(page.getByText(connector.displayName)).toBeVisible();
    await expect(page.getByText(/buaa\.roster\.full_snapshot/)).toBeVisible();

    await page.getByRole('tab', { name: '策略摘要' }).click();
    await expect(page.getByText('7 天', { exact: true })).toBeVisible();
    await expect(page.getByText('8 天', { exact: true })).toBeVisible();
    await expect(page.getByText('14 天', { exact: true })).toBeVisible();
  });

  test('manual full roster sync requires an audited reason and queues a persistent job', async ({
    page,
  }) => {
    await page.goto('/users/school-config');
    await page.getByRole('tab', { name: '学籍快照' }).click();
    await page.getByRole('button', { name: '立即完整同步' }).click();

    const dialog = page.getByRole('dialog', {
      name: '立即执行完整学籍同步',
    });
    await dialog.getByRole('textbox').fill('上线前完整快照复核');
    await dialog.getByRole('button', { name: '确认并审计' }).click();

    await expect
      .poll(() => captured)
      .toContainEqual({
        path: `/api/v1/admin/student-verification/schools/${schoolCode}/roster-sync-requests`,
        method: 'POST',
        body: { reason: '上线前完整快照复核' },
      });
    await expect(page.getByText(/最近手动同步：pending/)).toBeVisible();
  });

  test('manual review is school-scoped and records a user-visible audited decision', async ({
    page,
  }) => {
    await page.goto('/users/student-verification');
    await page.getByPlaceholder('学校代码，例如 4111010006').fill(schoolCode);
    await page.getByRole('button', { name: '刷新队列' }).click();

    const row = page.getByRole('row', { name: /张\*/ });
    await expect(row).toContainText('20****01');
    await expect(row).toContainText('学生证');
    await row.getByRole('button', { name: '查看详情' }).click();
    await expect(page.getByText('计算机学院')).toBeVisible();

    await page.getByRole('button', { name: '批准并签发凭据' }).click();
    const dialog = page.getByRole('dialog', { name: '记录审核决定' });
    await dialog
      .getByPlaceholder('说明通过依据、需要补充的材料，或可执行的申诉/重试路径')
      .fill('材料与学校记录一致，可在有效期内使用。');
    await dialog.getByRole('button', { name: '确认并审计' }).click();

    await expect
      .poll(() => captured)
      .toContainEqual({
        path: `/api/v1/admin/student-verification/manual-reviews/${manualCase.id}/decision`,
        method: 'POST',
        body: {
          action: 'approve',
          expiresInDays: 365,
          userVisibleReason: '材料与学校记录一致，可在有效期内使用。',
        },
      });
  });

  test('credential and conflict governance do not expose raw identity evidence', async ({
    page,
  }) => {
    await page.goto('/users/student-credentials');
    await page.getByPlaceholder('学校代码').fill(schoolCode);
    await page.getByRole('button', { name: '刷新' }).click();

    await expect(page.getByText('20****01')).toBeVisible();
    await expect(page.getByText('实名信息校验')).toBeVisible();
    await expect(page.getByText('身份证号')).toHaveCount(0);
    await expect(page.getByText('证件原图')).toHaveCount(0);

    await page.getByRole('tab', { name: '主体冲突' }).click();
    await expect(page.getByText('subject_already_bound')).toBeVisible();
    await expect(page.getByRole('cell', { name: '43' })).toBeVisible();
    await expect(page.getByRole('cell', { name: '42' })).toBeVisible();
  });
});
