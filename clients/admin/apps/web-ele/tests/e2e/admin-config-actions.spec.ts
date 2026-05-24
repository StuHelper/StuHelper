import type { Page, Route } from '@playwright/test';

import { expect, test } from '@playwright/test';

const capabilities = [
  'user:school:read',
  'user:school:update',
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
      await route.fulfill(ok([schoolConfig]));
      return;
    }
    if (path.startsWith('/api/v1/admin/school-configs/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
      await route.fulfill(ok(schoolConfig));
      return;
    }

    if (path === '/api/v1/admin/admission/policies') {
      await route.fulfill(ok([admissionPolicy]));
      return;
    }
    if (path.startsWith('/api/v1/admin/admission/policies/')) {
      capturedMutations.push({ path, method, body: parseJsonBody(route) });
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
    await mockAdminApi(page, capturedMutations);
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
        path: '/api/v1/admin/school-configs/1001',
        method: 'PUT',
        body: {
          academicDbTable: 'academic_records_new',
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
      .getByPlaceholder('每行一个群号')
      .fill('guild-admin\nguild-ops');
    await policyForm.getByRole('button', { name: '保存' }).click();

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
          linkWaitSeconds: 450,
          failedJoinLimit: 5,
          maxExtensionDays: 45,
          managementGuildIDs: ['guild-admin', 'guild-ops'],
        }),
      });
  });
});
