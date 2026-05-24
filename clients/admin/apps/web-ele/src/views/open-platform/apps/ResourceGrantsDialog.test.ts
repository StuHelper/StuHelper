// @vitest-environment happy-dom

import type {
  OpenPlatformAppWithScopes,
  OpenPlatformResourceGrant,
} from '#/api/admin';

import { mount } from '@vue/test-utils';

import { ElMessage, ElMessageBox } from 'element-plus';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getOpenPlatformResourceGrants,
  grantOpenPlatformResourceAccess,
  revokeOpenPlatformResourceAccess,
} from '#/api/admin';

import ResourceGrantsDialog from './ResourceGrantsDialog.vue';

vi.mock('#/locales', () => ({
  $t: (key: string, params?: Record<string, unknown>) =>
    params ? `${key}:${JSON.stringify(params)}` : key,
}));

vi.mock('#/api/admin', () => ({
  getOpenPlatformResourceGrants: vi.fn(),
  grantOpenPlatformResourceAccess: vi.fn(),
  revokeOpenPlatformResourceAccess: vi.fn(),
}));

vi.mock('element-plus', () => ({
  ElAlert: {
    template: '<div><slot /></div>',
  },
  ElButton: {
    template: '<button type="button"><slot /></button>',
  },
  ElCheckbox: {
    template: '<label><slot /></label>',
  },
  ElCheckboxGroup: {
    template: '<div><slot /></div>',
  },
  ElDialog: {
    template: '<div data-stub="el-dialog"><slot /><slot name="footer" /></div>',
  },
  ElForm: {
    template: '<form><slot /></form>',
  },
  ElFormItem: {
    template: '<div><slot /></div>',
  },
  ElInput: {
    template: '<input />',
  },
  ElMessage: {
    success: vi.fn(),
    warning: vi.fn(),
  },
  ElMessageBox: {
    prompt: vi.fn(),
  },
  ElOption: {
    template: '<option />',
  },
  ElSelect: {
    template: '<select><slot /></select>',
  },
  ElTable: {
    template: '<div><slot /></div>',
  },
  ElTableColumn: {
    template: '<div />',
  },
  ElTag: {
    template: '<span><slot /></span>',
  },
}));

const baseApp: OpenPlatformAppWithScopes = {
  app: {
    id: 12,
    displayName: 'Resource Client',
    description: 'Resource client',
    clientID: 'resource-client',
    homepageURL: 'https://client.example.com',
    privacyPolicyURL: 'https://client.example.com/privacy',
    redirectURIs: ['https://client.example.com/callback'],
    status: 'approved',
    createdAt: '2026-05-01T00:00:00Z',
    updatedAt: '2026-05-01T00:00:00Z',
  },
  redirectURIRequests: [],
  scopes: [
    {
      id: 1,
      scope: 'resource.read',
      displayName: 'Read resources',
      sensitivity: 'high',
      fields: [],
      status: 'approved',
      reason: 'read dataset',
      reviewerUserID: 1,
      reviewedAt: '2026-05-01T00:00:00Z',
      decisionNote: null,
      createdAt: '2026-05-01T00:00:00Z',
      updatedAt: '2026-05-01T00:00:00Z',
    },
  ],
};

const grant: OpenPlatformResourceGrant = {
  appID: 12,
  resourceType: 'resource_item',
  resourceID: 'resource-42',
  action: 'read',
  relation: 'can_read_by_app',
};

function mountDialog() {
  return mount(ResourceGrantsDialog, {
    props: {
      app: baseApp,
      visible: true,
    },
    global: {
      directives: { loading: { mounted() {}, updated() {} } },
    },
  });
}

describe('ResourceGrantsDialog', () => {
  beforeEach(() => {
    vi.mocked(getOpenPlatformResourceGrants).mockResolvedValue({
      app: baseApp.app,
      grants: [],
    });
    vi.mocked(grantOpenPlatformResourceAccess).mockResolvedValue({
      app: baseApp.app,
      grants: [grant],
    });
    vi.mocked(revokeOpenPlatformResourceAccess).mockResolvedValue({
      app: baseApp.app,
      grants: [],
    });
    vi.mocked(ElMessage.success).mockClear();
    vi.mocked(ElMessage.warning).mockClear();
    vi.mocked(ElMessageBox.prompt).mockReset();
  });

  it('requires a reason before granting resource access', async () => {
    const wrapper = mountDialog();
    const vm = wrapper.vm as unknown as {
      draft: {
        actions: string[];
        reason: string;
        resourceID: string;
      };
      submitGrant: () => Promise<void>;
    };
    vm.draft.resourceID = 'resource-42';
    vm.draft.actions = ['read'];
    vm.draft.reason = ' ';

    await vm.submitGrant();

    expect(ElMessage.warning).toHaveBeenCalledWith(
      'admin.openPlatform.apps.reasonRequired',
    );
    expect(grantOpenPlatformResourceAccess).not.toHaveBeenCalled();
  });

  it('trims and sends the audit reason when granting resource access', async () => {
    const wrapper = mountDialog();
    const vm = wrapper.vm as unknown as {
      draft: {
        actions: string[];
        reason: string;
        resourceID: string;
      };
      submitGrant: () => Promise<void>;
    };
    vm.draft.resourceID = 'resource-42';
    vm.draft.actions = ['read'];
    vm.draft.reason = '  approved dataset sharing  ';

    await vm.submitGrant();

    expect(grantOpenPlatformResourceAccess).toHaveBeenCalledWith(12, {
      actions: ['read'],
      reason: 'approved dataset sharing',
      resourceID: 'resource-42',
      resourceType: 'resource_item',
    });
  });

  it('requires and sends an audit reason when revoking a resource grant', async () => {
    vi.mocked(ElMessageBox.prompt).mockResolvedValue({
      value: '  cleanup expired sharing  ',
    } as Awaited<ReturnType<typeof ElMessageBox.prompt>>);
    const wrapper = mountDialog();
    const vm = wrapper.vm as unknown as {
      revokeGrant: (item: OpenPlatformResourceGrant) => Promise<void>;
    };

    await vm.revokeGrant(grant);

    const promptOptions = vi.mocked(ElMessageBox.prompt).mock.calls[0]?.[2];
    expect(promptOptions?.inputValidator?.(' ')).toBe(
      'admin.openPlatform.apps.reasonRequired',
    );
    expect(revokeOpenPlatformResourceAccess).toHaveBeenCalledWith(12, {
      actions: ['read'],
      reason: 'cleanup expired sharing',
      resourceID: 'resource-42',
      resourceType: 'resource_item',
    });
  });
});
