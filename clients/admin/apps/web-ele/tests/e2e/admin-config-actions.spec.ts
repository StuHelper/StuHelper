import type { Page, Route } from './fixtures';

import { expect, test } from './fixtures';

const capabilities = [
  'user:school:read',
  'user:school:update',
  'user:system:read',
  'user:system:update',
  'admission:policy:update',
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

const schoolConfig = {
  approvalPolicy: 'auto',
  schoolID: 4_111_010_006,
  schoolCode: '4111010006',
  schoolName: '测试大学',
  verificationMethod: 'ldap',
  enabled: true,
  schoolSsoEnabled: true,
  schoolEmailOtpEnabled: true,
  schoolEmailIdentityPolicy: {
    type: 'academic_student_email',
    studentIDEmailDomain: 'buaa.edu.cn',
    requireStudentName: true,
  },
  academicDbTable: 'academic_students',
  consentText: '仅用于学生身份认证',
  ldapConfig: {
    url: 'ldap://ldap.example.com',
    baseDN: 'dc=example,dc=com',
    systemBindDN: 'cn=reader,dc=example,dc=com',
    useTLS: true,
  },
};

const systemConfig = {
  key: 'review.retention_days',
  value: '90',
  description: '评课数据保留天数',
  updatedAt: '2026-06-03T00:00:00Z',
};

const admissionPolicy = {
  id: 'policy-qq-1',
  platform: 'qq',
  guildID: 'guild-1',
  guardEnabled: true,
  joinHandlingStrategy: 'post_join_guard',
  autoApproveJoin: true,
  autoApproveVerifiedJoin: true,
  autoApproveUnverifiedJoin: true,
  unverifiedJoinRejectReason: '请先完成 StuHelper 学生认证后再申请入群。',
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

interface CapturedMutation {
  body: unknown;
  method: string;
  path: string;
}

let nextSchoolConfigListErrorMessage: null | string = null;
let nextSchoolConfigUpdateErrorMessage: null | string = null;
let nextSystemConfigListErrorMessage: null | string = null;
let nextSystemConfigUpdateErrorMessage: null | string = null;
let nextAdmissionPolicyListErrorMessage: null | string = null;
let nextAdmissionPolicyUpdateErrorMessage: null | string = null;

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

async function mockAdminApi(page: Page, capturedMutations: CapturedMutation[]) {
  await page.route('**/api/v1/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();

    if (path === '/api/v1/auth/me') {
      await route.fulfill(ok(adminUser));
      return;
    }

    if (path === '/api/v1/admin/school-configs') {
      if (nextSchoolConfigListErrorMessage) {
        const message = nextSchoolConfigListErrorMessage;
        nextSchoolConfigListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_SCHOOL_CONFIG_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok([schoolConfig]));
      return;
    }
    if (path.startsWith('/api/v1/admin/school-configs/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      if (nextSchoolConfigUpdateErrorMessage) {
        const message = nextSchoolConfigUpdateErrorMessage;
        nextSchoolConfigUpdateErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_SCHOOL_CONFIG_UPDATE_REJECTED',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok(schoolConfig));
      return;
    }

    if (path === '/api/v1/admin/system-configs') {
      if (nextSystemConfigListErrorMessage) {
        const message = nextSystemConfigListErrorMessage;
        nextSystemConfigListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_SYSTEM_CONFIG_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok([systemConfig]));
      return;
    }
    if (path.startsWith('/api/v1/admin/system-configs/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      if (nextSystemConfigUpdateErrorMessage) {
        const message = nextSystemConfigUpdateErrorMessage;
        nextSystemConfigUpdateErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_SYSTEM_CONFIG_UPDATE_REJECTED',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok({ ...systemConfig, value: '180' }));
      return;
    }

    if (path === '/api/v1/admin/admission/policies' && method === 'POST') {
      const body = parseJsonBody(route) as { guildID?: string } | null;
      capturedMutations.push({ path, method, body });
      await route.fulfill(
        ok({
          ...admissionPolicy,
          id: `policy-${body?.guildID ?? 'new'}`,
          guildID: body?.guildID ?? 'guild-new',
          managementGuildIDs: [],
        }),
      );
      return;
    }

    if (path === '/api/v1/admin/admission/policies') {
      if (nextAdmissionPolicyListErrorMessage) {
        const message = nextAdmissionPolicyListErrorMessage;
        nextAdmissionPolicyListErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_ADMISSION_POLICY_LIST_UNAVAILABLE',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok([admissionPolicy]));
      return;
    }
    if (path.startsWith('/api/v1/admin/admission/policies/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      if (nextAdmissionPolicyUpdateErrorMessage) {
        const message = nextAdmissionPolicyUpdateErrorMessage;
        nextAdmissionPolicyUpdateErrorMessage = null;
        await route.fulfill(
          json({
            success: false,
            error: {
              code: 'E2E_ADMISSION_POLICY_UPDATE_REJECTED',
              message,
            },
          }),
        );
        return;
      }
      await route.fulfill(ok(admissionPolicy));
      return;
    }

    await route.fulfill(
      json(
        {
          success: false,
          error: {
            code: 'E2E_UNMOCKED',
            message: `unmocked config action request: ${method} ${path}`,
          },
        },
        500,
      ),
    );
  });
}

test.describe('Admin configuration actions', () => {
  let capturedMutations: CapturedMutation[];

  test.beforeEach(async ({ page }) => {
    capturedMutations = [];
    nextSchoolConfigListErrorMessage = null;
    nextSchoolConfigUpdateErrorMessage = null;
    nextSystemConfigListErrorMessage = null;
    nextSystemConfigUpdateErrorMessage = null;
    nextAdmissionPolicyListErrorMessage = null;
    nextAdmissionPolicyUpdateErrorMessage = null;
    await mockAdminApi(page, capturedMutations);
  });

  test('school config list failures show a persistent retry path', async ({
    page,
  }) => {
    nextSchoolConfigListErrorMessage = '学校配置目录暂不可用';

    await page.goto('/users/school-config');

    const loadError = page.locator('.admin-load-error', {
      hasText: '学校配置目录暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('测试大学')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('测试大学')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('school config dialog submits LDAP settings and password updates', async ({
    page,
  }) => {
    await page.goto('/users/school-config');

    await expect(page.getByText('测试大学')).toBeVisible();
    await page
      .getByRole('row', { name: /测试大学/ })
      .getByRole('button', { name: /编辑|Edit/ })
      .click();

    const dialog = page.getByRole('dialog', { name: /编辑学校配置/ });
    await dialog.getByPlaceholder('请输入学校名称').fill('测试大学新校区');
    await dialog.locator('.el-select').nth(1).click();
    await page.getByRole('option', { name: '人工审核' }).click();
    await dialog
      .getByPlaceholder('例如 ldaps://ldap.example.edu:636')
      .fill('ldaps://ldap.new.example.edu:636');
    await dialog
      .getByPlaceholder('例如 dc=example,dc=edu')
      .fill('dc=new,dc=edu');
    await dialog
      .getByPlaceholder('例如 cn=service,dc=example,dc=edu')
      .fill('cn=service,dc=new,dc=edu');
    await dialog.getByPlaceholder('留空表示不更新密码').fill('new-secret');
    await dialog
      .getByPlaceholder('academic_records')
      .fill('academic_records_new');
    await dialog.getByPlaceholder('用户同意文案...').fill('新同意书文案');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    await expect
      .poll(() => capturedMutations)
      .toContainEqual({
        path: '/api/v1/admin/school-configs/4111010006',
        method: 'PUT',
        body: {
          academicDbTable: 'academic_records_new',
          approvalPolicy: 'manual',
          consentText: '新同意书文案',
          enabled: true,
          ldapConfig: {
            baseDN: 'dc=new,dc=edu',
            systemBindDN: 'cn=service,dc=new,dc=edu',
            systemBindPassword: 'new-secret',
            url: 'ldaps://ldap.new.example.edu:636',
            useTLS: true,
          },
          schoolName: '测试大学新校区',
          verificationMethod: 'ldap',
        },
      });
  });

  test('school config update failures preserve backend error detail', async ({
    page,
  }) => {
    nextSchoolConfigUpdateErrorMessage = '学校配置保存被后台拒绝';

    await page.goto('/users/school-config');

    await expect(page.getByText('测试大学')).toBeVisible();
    await page
      .getByRole('row', { name: /测试大学/ })
      .getByRole('button', { name: /编辑|Edit/ })
      .click();

    const dialog = page.getByRole('dialog', { name: /编辑学校配置/ });
    await dialog.getByPlaceholder('请输入学校名称').fill('失败学校');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '学校配置保存被后台拒绝',
    });
    await expect(actionError).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(dialog.getByPlaceholder('请输入学校名称')).toHaveValue(
      '失败学校',
    );
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('system config list failures show a persistent retry path', async ({
    page,
  }) => {
    nextSystemConfigListErrorMessage = '系统配置列表暂不可用';

    await page.goto('/users/system-config');

    const loadError = page.locator('.admin-load-error', {
      hasText: '系统配置列表暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('review.retention_days')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('review.retention_days')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });

  test('system config update failures preserve backend error detail', async ({
    page,
  }) => {
    nextSystemConfigUpdateErrorMessage = '系统配置保存被后台拒绝';

    await page.goto('/users/system-config');

    const configRow = page.getByRole('row', { name: /review\.retention_days/ });
    await expect(configRow).toBeVisible();
    await configRow.getByRole('button', { name: /编辑|Edit/ }).click();

    const dialog = page.getByRole('dialog', { name: /编辑配置|Edit/ });
    await dialog.getByPlaceholder('请输入配置值').fill('180');
    await dialog.getByRole('button', { name: /保存|Save/ }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '系统配置保存被后台拒绝',
    });
    await expect(actionError).toBeVisible();
    await expect(dialog).toBeVisible();
    await expect(dialog.getByPlaceholder('请输入配置值')).toHaveValue('180');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('admission policy save serializes numeric fields and guild list', async ({
    page,
  }) => {
    await page.goto('/users/admission-policy');

    const policyForm = page
      .locator('form')
      .filter({ hasText: 'QQ 群 guild-1' });
    await expect(policyForm).toBeVisible();

    await policyForm
      .getByRole('spinbutton', { name: '绑定链接等待（秒）' })
      .fill('450');
    await policyForm
      .getByRole('spinbutton', { name: '失败入群上限' })
      .fill('5');
    await policyForm
      .getByRole('spinbutton', { name: '最大延期天数' })
      .fill('45');
    await policyForm
      .locator('.el-form-item', { hasText: '启用入群认证守卫' })
      .locator('.el-switch')
      .click();
    await policyForm.getByText('申请时审核', { exact: true }).click();
    await policyForm
      .getByRole('textbox', { name: '未认证拒绝理由' })
      .fill('请先完成 StuHelper 学生认证后再申请入群');
    await policyForm
      .getByPlaceholder('每行一个材料审核通知群号，可留空；这里不是目标认证群')
      .fill('guild-admin\nguild-ops');
    await policyForm.getByRole('button', { name: '保存' }).click();
    await expect(
      page.getByText('已保存 QQ 群 guild-1 入群认证策略'),
    ).toBeVisible();

    await expect
      .poll(() =>
        capturedMutations.find(
          (mutation) =>
            mutation.path === '/api/v1/admin/admission/policies/policy-qq-1',
        ),
      )
      .toMatchObject({
        path: '/api/v1/admin/admission/policies/policy-qq-1',
        method: 'PUT',
        body: expect.objectContaining({
          id: 'policy-qq-1',
          guardEnabled: false,
          joinHandlingStrategy: 'join_request_review',
          autoApproveJoin: false,
          autoApproveVerifiedJoin: true,
          autoApproveUnverifiedJoin: false,
          unverifiedJoinRejectReason:
            '请先完成 StuHelper 学生认证后再申请入群',
          freshmanChannelClosesAt: '2026-09-01T00:00:00Z',
          freshmanDefaultExpiresAt: '2026-10-01T00:00:00Z',
          linkWaitSeconds: 450,
          failedJoinLimit: 5,
          maxExtensionDays: 45,
          managementGuildIDs: ['guild-admin', 'guild-ops'],
        }),
      });
  });

  test('admission policy create posts new target guilds copied from an existing policy', async ({
    page,
  }) => {
    await page.goto('/users/admission-policy');

    await page.getByRole('button', { name: '新增目标认证群' }).click();
    const dialog = page.getByRole('dialog', { name: '新增目标认证群' });
    await expect(dialog).toBeVisible();
    await dialog
      .getByPlaceholder('每行一个需要开启入群认证的 QQ 群号')
      .fill('guild-2\nguild-3');
    await dialog.getByRole('button', { name: '创建' }).click();

    await expect(page.getByText('已创建 2 个新生认证群策略')).toBeVisible();
    await expect
      .poll(() =>
        capturedMutations.filter(
          (mutation) =>
            mutation.path === '/api/v1/admin/admission/policies' &&
            mutation.method === 'POST',
        ),
      )
      .toMatchObject([
        {
          body: {
            sourcePolicyID: 'policy-qq-1',
            platform: 'qq',
            guildID: 'guild-2',
          },
        },
        {
          body: {
            sourcePolicyID: 'policy-qq-1',
            platform: 'qq',
            guildID: 'guild-3',
          },
        },
      ]);
  });

  test('admission policy save failures preserve backend error detail', async ({
    page,
  }) => {
    nextAdmissionPolicyUpdateErrorMessage = '入群策略保存被后台拒绝';

    await page.goto('/users/admission-policy');

    const policyForm = page
      .locator('form')
      .filter({ hasText: 'QQ 群 guild-1' });
    await expect(policyForm).toBeVisible();
    await policyForm
      .getByRole('spinbutton', { name: '绑定链接等待（秒）' })
      .fill('450');
    await policyForm.getByRole('button', { name: '保存' }).click();

    const actionError = page.locator('.admin-load-error', {
      hasText: '入群策略保存被后台拒绝',
    });
    await expect(actionError).toBeVisible();
    await expect(policyForm).toBeVisible();
    await expect(
      policyForm.getByRole('spinbutton', { name: '绑定链接等待（秒）' }),
    ).toHaveValue('450');
    await expect(page.getByText('请求失败，请稍后重试')).toHaveCount(0);
  });

  test('admission policy list failures show a persistent retry path', async ({
    page,
  }) => {
    nextAdmissionPolicyListErrorMessage = '入群认证策略暂不可用';

    await page.goto('/users/admission-policy');

    const loadError = page.locator('.admin-load-error', {
      hasText: '入群认证策略暂不可用',
    });
    await expect(loadError).toBeVisible();
    await expect(page.getByText('QQ 群 guild-1')).toHaveCount(0);
    await loadError.getByRole('button', { name: /重试|Retry/ }).click();
    await expect(page.getByText('QQ 群 guild-1')).toBeVisible();
    await expect(loadError).toHaveCount(0);
  });
});
