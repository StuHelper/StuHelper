// @vitest-environment happy-dom

import type { MemberBlacklistEntry } from '#/api/admin';

import { mount } from '@vue/test-utils';

import { describe, expect, it, vi } from 'vitest';

import CreateBlacklistDialog from './CreateBlacklistDialog.vue';
import ReleaseBlacklistDialog from './ReleaseBlacklistDialog.vue';

vi.mock('#/locales', () => ({
  $t: (key: string, params?: Record<string, unknown>) =>
    params ? `${key}:${Object.values(params).join('/')}` : key,
}));

const dialogStubs = {
  ElDialog: {
    template: '<div data-stub="el-dialog"><slot /><slot name="footer" /></div>',
  },
  ElPopconfirm: {
    template: '<div data-stub="el-popconfirm"><slot name="reference" /></div>',
  },
};

const baseEntry: MemberBlacklistEntry = {
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
  createdAt: '2026-04-01T00:00:00Z',
  updatedAt: '2026-04-01T00:00:00Z',
  expiresAt: null,
  releasedAt: null,
  releasedByType: null,
  releasedByID: null,
  releaseReasonCode: null,
  releaseReason: null,
  metadata: {},
};

describe('CreateBlacklistDialog', () => {
  it('emits a guild-scope create payload that matches the API contract', async () => {
    const wrapper = mount(CreateBlacklistDialog, {
      props: { visible: true, submitting: false },
      global: { stubs: dialogStubs },
    });

    const vm = wrapper.vm as unknown as {
      draft: {
        expiresAt: Date | string;
        guildID: string;
        platform: string;
        reasonText: string;
        scopeType: 'global' | 'guild';
        subjectID: string;
      };
    };
    vm.draft.platform = 'qq';
    vm.draft.subjectID = '20002';
    vm.draft.scopeType = 'guild';
    vm.draft.guildID = 'guild-42';
    vm.draft.reasonText = 'spamming';
    vm.draft.expiresAt = '';
    await wrapper.vm.$nextTick();

    const submit = wrapper.find('[data-action="submitCreate"]');
    expect(submit.exists()).toBe(true);
    await submit.trigger('click');

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      platform: 'qq',
      subjectType: 'qq_user',
      subjectID: '20002',
      scopeType: 'guild',
      guildID: 'guild-42',
      source: 'manual_admin',
      reasonCode: 'manual_blacklist',
      reasonText: 'spamming',
      expiresAt: undefined,
      metadata: {
        operatorInput: '20002',
        scopeSelectionContext: 'admin_console_form_guild',
      },
    });
  });

  it('omits guildID and tags global context for global blacklists', async () => {
    const wrapper = mount(CreateBlacklistDialog, {
      props: { visible: true, submitting: false },
      global: { stubs: dialogStubs },
    });

    const vm = wrapper.vm as unknown as {
      draft: {
        expiresAt: Date | string;
        guildID: string;
        platform: string;
        reasonText: string;
        scopeType: 'global' | 'guild';
        subjectID: string;
      };
      submit: () => void;
    };
    vm.draft.platform = 'qq';
    vm.draft.subjectID = '30003';
    vm.draft.scopeType = 'global';
    vm.draft.guildID = 'leftover-should-be-ignored';
    vm.draft.reasonText = 'global ban';
    vm.draft.expiresAt = '';
    await wrapper.vm.$nextTick();

    vm.submit();

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      platform: 'qq',
      subjectType: 'qq_user',
      subjectID: '30003',
      scopeType: 'global',
      guildID: undefined,
      source: 'manual_admin',
      reasonCode: 'manual_blacklist',
      reasonText: 'global ban',
      expiresAt: undefined,
      metadata: {
        operatorInput: '30003',
        scopeSelectionContext: 'admin_console_form_global',
      },
    });
  });

  it('does not emit when required fields are missing', async () => {
    const wrapper = mount(CreateBlacklistDialog, {
      props: { visible: true, submitting: false },
      global: { stubs: dialogStubs },
    });

    await wrapper.find('[data-action="submitCreate"]').trigger('click');
    expect(wrapper.emitted('submit')).toBeUndefined();
  });

  it('does not emit while a create request is already submitting', async () => {
    const wrapper = mount(CreateBlacklistDialog, {
      props: { visible: true, submitting: true },
      global: { stubs: dialogStubs },
    });

    const vm = wrapper.vm as unknown as {
      draft: {
        guildID: string;
        platform: string;
        reasonText: string;
        scopeType: 'global' | 'guild';
        subjectID: string;
      };
    };
    vm.draft.platform = 'qq';
    vm.draft.subjectID = '20002';
    vm.draft.scopeType = 'guild';
    vm.draft.guildID = 'guild-42';
    vm.draft.reasonText = 'spamming';
    await wrapper.vm.$nextTick();

    const submit = wrapper.find('[data-action="submitCreate"]');
    expect((submit.element as HTMLButtonElement).disabled).toBe(true);
    await submit.trigger('click');

    expect(wrapper.emitted('submit')).toBeUndefined();
  });

  it('reset() restores the draft to defaults', async () => {
    const wrapper = mount(CreateBlacklistDialog, {
      props: { visible: true, submitting: false },
      global: { stubs: dialogStubs },
    });

    const exposed = wrapper.vm as unknown as {
      draft: { reasonText: string; scopeType: string; subjectID: string };
      reset: () => void;
    };
    exposed.draft.subjectID = '99999';
    exposed.draft.reasonText = 'noise';
    exposed.draft.scopeType = 'global';
    exposed.reset();
    await wrapper.vm.$nextTick();

    expect(exposed.draft.subjectID).toBe('');
    expect(exposed.draft.reasonText).toBe('');
    expect(exposed.draft.scopeType).toBe('guild');
  });
});

describe('ReleaseBlacklistDialog', () => {
  it('defaults to manual_pardon when releasing an admission_failure entry', async () => {
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: { visible: true, submitting: false, target: null },
      global: { stubs: dialogStubs },
    });

    await wrapper.setProps({
      target: { ...baseEntry, source: 'admission_failure' },
    });
    await wrapper.vm.$nextTick();

    const vm = wrapper.vm as unknown as {
      draft: { releaseReason: string; releaseReasonCode: string };
    };
    expect(vm.draft.releaseReasonCode).toBe('manual_pardon');

    await wrapper.find('[data-action="submitRelease"]').trigger('click');

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      id: 'entry-active',
      request: { releaseReasonCode: 'manual_pardon' },
    });
  });

  it('defaults to release_only for non-admission sources and trims note', async () => {
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: { visible: true, submitting: false, target: null },
      global: { stubs: dialogStubs },
    });

    await wrapper.setProps({
      target: { ...baseEntry, source: 'manual_admin' },
    });
    await wrapper.vm.$nextTick();

    const vm = wrapper.vm as unknown as {
      draft: { releaseReason: string; releaseReasonCode: string };
    };
    expect(vm.draft.releaseReasonCode).toBe('release_only');
    vm.draft.releaseReason = '   appeal accepted   ';
    await wrapper.vm.$nextTick();

    await wrapper.find('[data-action="submitRelease"]').trigger('click');

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      id: 'entry-active',
      request: {
        releaseReasonCode: 'release_only',
        releaseReason: 'appeal accepted',
      },
    });
  });

  it('does not emit when target is null', async () => {
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: { visible: true, submitting: false, target: null },
      global: { stubs: dialogStubs },
    });

    await wrapper.find('[data-action="submitRelease"]').trigger('click');
    expect(wrapper.emitted('submit')).toBeUndefined();
  });

  it('does not emit while a release request is already submitting', async () => {
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: {
        visible: true,
        submitting: true,
        target: { ...baseEntry, source: 'admission_failure' },
      },
      global: { stubs: dialogStubs },
    });

    const submit = wrapper.find('[data-action="submitRelease"]');
    expect((submit.element as HTMLButtonElement).disabled).toBe(true);
    await submit.trigger('click');

    expect(wrapper.emitted('submit')).toBeUndefined();
  });

  it('resets the draft when reopening the same target after cancelling', async () => {
    const target = { ...baseEntry, source: 'admission_failure' as const };
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: { visible: false, submitting: false, target: null },
      global: { stubs: dialogStubs },
    });

    await wrapper.setProps({ target });
    await wrapper.setProps({ visible: true });
    await wrapper.vm.$nextTick();

    const vm = wrapper.vm as unknown as {
      draft: { releaseReason: string; releaseReasonCode: string };
    };
    expect(vm.draft.releaseReasonCode).toBe('manual_pardon');

    vm.draft.releaseReasonCode = 'release_only';
    vm.draft.releaseReason = 'mistaken edit';
    await wrapper.setProps({ visible: false });
    await wrapper.vm.$nextTick();

    await wrapper.setProps({ visible: true });
    await wrapper.vm.$nextTick();

    expect(vm.draft.releaseReasonCode).toBe('manual_pardon');
    expect(vm.draft.releaseReason).toBe('');
  });
});
