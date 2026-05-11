// @vitest-environment happy-dom

import type { MemberBlacklistEntry } from '#/api/admin';

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';

import { describe, expect, it } from 'vitest';

import BlacklistFilters from './BlacklistFilters.vue';
import BlacklistTable from './BlacklistTable.vue';
import CreateBlacklistDialog from './CreateBlacklistDialog.vue';
import ReleaseBlacklistDialog from './ReleaseBlacklistDialog.vue';

const dialogStubs = {
  ElDialog: {
    template: '<div data-stub="el-dialog"><slot /><slot name="footer" /></div>',
  },
  ElPopconfirm: {
    template: '<div data-stub="el-popconfirm"><slot name="reference" /></div>',
  },
};

// Element Plus tables drive scoped slots from the inside; in happy-dom we
// reimplement the row→column slot relationship with a tiny pair of stubs that
// faithfully calls each column's default slot once per row.
const ElTableStub = defineComponent({
  name: 'ElTable',
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () => {
      const cols = slots.default?.() ?? [];
      return h(
        'div',
        { 'data-stub': 'el-table' },
        (props.data as object[]).map((row, rowIndex) =>
          h(
            'div',
            { 'data-stub': 'el-row', key: rowIndex },
            cols.map((col: any) => {
              const colSlot = col?.children?.default;
              return colSlot ? colSlot({ row, $index: rowIndex }) : null;
            }),
          ),
        ),
      );
    };
  },
});
const ElTableColumnStub = defineComponent({
  name: 'ElTableColumn',
  setup(_, { slots }) {
    return () =>
      h('div', { 'data-stub': 'el-table-column' }, slots.default?.());
  },
});
const ElPaginationStub = defineComponent({
  name: 'ElPagination',
  template: '<div data-stub="el-pagination" />',
});

const tableStubs = {
  ElTable: ElTableStub,
  ElTableColumn: ElTableColumnStub,
  ElPagination: ElPaginationStub,
  // suppress the v-loading directive resolver
  vLoading: { mounted() {}, updated() {} },
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

describe('BlacklistFilters', () => {
  it('emits search/reset/openCreate when its action buttons are clicked', async () => {
    const wrapper = mount(BlacklistFilters, {
      props: {
        canManage: true,
        platform: '',
        scopeType: '',
        source: '',
        status: 'active',
        guildID: '',
        subjectID: '',
      },
    });

    const buttons = wrapper.findAll('button');
    const search = buttons.find((btn) => btn.text() === '查询');
    const reset = buttons.find((btn) => btn.text() === '重置');
    const create = wrapper.find('[data-action="openCreate"]');

    expect(search).toBeDefined();
    expect(reset).toBeDefined();
    expect(create.exists()).toBe(true);
    if (!search || !reset) {
      throw new Error('expected search and reset buttons');
    }

    await search.trigger('click');
    await reset.trigger('click');
    await create.trigger('click');

    expect(wrapper.emitted('search')).toHaveLength(1);
    expect(wrapper.emitted('reset')).toHaveLength(1);
    expect(wrapper.emitted('openCreate')).toHaveLength(1);
  });

  it('emits search immediately when select filters change', async () => {
    const wrapper = mount(BlacklistFilters, {
      props: {
        canManage: true,
        platform: '',
        scopeType: '',
        source: '',
        status: 'active',
        guildID: '',
        subjectID: '',
      },
    });

    const selects = wrapper.findAllComponents({ name: 'ElSelect' });
    expect(selects).toHaveLength(3);

    await selects[0]!.vm.$emit('change', 'global');
    await selects[1]!.vm.$emit('change', 'manual_admin');
    await selects[2]!.vm.$emit('change', 'released');

    expect(wrapper.emitted('search')).toHaveLength(3);
    for (const select of selects) {
      expect(select.props('teleported')).toBe(false);
    }
  });

  it('hides the create button when canManage is false', () => {
    const wrapper = mount(BlacklistFilters, {
      props: {
        canManage: false,
        platform: '',
        scopeType: '',
        source: '',
        status: 'active',
        guildID: '',
        subjectID: '',
      },
    });

    expect(wrapper.find('[data-action="openCreate"]').exists()).toBe(false);
  });
});

describe('BlacklistTable', () => {
  it('emits release with the row payload when the release action is clicked', async () => {
    const wrapper = mount(BlacklistTable, {
      props: {
        loading: false,
        items: [baseEntry],
        total: 1,
        canManage: true,
        page: 1,
        pageSize: 20,
      },
      global: {
        stubs: {
          ElTable: ElTableStub,
          ElTableColumn: ElTableColumnStub,
          ElPagination: ElPaginationStub,
        },
        directives: { loading: tableStubs.vLoading },
      },
    });

    const releaseBtn = wrapper.find('[data-action="release"]');
    expect(releaseBtn.exists()).toBe(true);
    await releaseBtn.trigger('click');

    const events = wrapper.emitted('release');
    expect(events).toHaveLength(1);
    expect(events?.[0]?.[0]).toEqual(
      expect.objectContaining({ id: 'entry-active', subjectID: '10001' }),
    );
  });

  it('hides the release action once the entry has expired', () => {
    const expired = { ...baseEntry, expiresAt: '2020-01-01T00:00:00Z' };
    const wrapper = mount(BlacklistTable, {
      props: {
        loading: false,
        items: [expired],
        total: 1,
        canManage: true,
        page: 1,
        pageSize: 20,
      },
      global: {
        stubs: {
          ElTable: ElTableStub,
          ElTableColumn: ElTableColumnStub,
          ElPagination: ElPaginationStub,
        },
        directives: { loading: tableStubs.vLoading },
      },
    });

    expect(wrapper.find('[data-action="release"]').exists()).toBe(false);
  });
});

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

    const events = wrapper.emitted('submit');
    expect(events).toHaveLength(1);
    expect(events?.[0]?.[0]).toEqual({
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

    // Global scope renders the submit inside ElPopconfirm, which is stubbed
    // and will not forward @confirm. Drive submission directly.
    vm.submit();

    const events = wrapper.emitted('submit');
    expect(events).toHaveLength(1);
    expect(events?.[0]?.[0]).toEqual({
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

    const submit = wrapper.find('[data-action="submitCreate"]');
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

    const submit = wrapper.find('[data-action="submitRelease"]');
    expect(submit.exists()).toBe(true);
    await submit.trigger('click');

    const events = wrapper.emitted('submit');
    expect(events).toHaveLength(1);
    expect(events?.[0]?.[0]).toEqual({
      id: 'entry-active',
      request: { releaseReasonCode: 'manual_pardon' },
    });
  });

  it('defaults to release_only for non-admission sources and trims the reason note', async () => {
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

  it('resets the draft when reopening the same target after cancelling', async () => {
    const target = { ...baseEntry, source: 'admission_failure' as const };
    const wrapper = mount(ReleaseBlacklistDialog, {
      props: { visible: false, submitting: false, target: null },
      global: { stubs: dialogStubs },
    });

    // First open: parent assigns the target then flips visible to true.
    await wrapper.setProps({ target });
    await wrapper.setProps({ visible: true });
    await wrapper.vm.$nextTick();

    const vm = wrapper.vm as unknown as {
      draft: { releaseReason: string; releaseReasonCode: string };
    };
    expect(vm.draft.releaseReasonCode).toBe('manual_pardon');

    // Operator changes both fields then cancels (parent does NOT clear target).
    vm.draft.releaseReasonCode = 'release_only';
    vm.draft.releaseReason = 'mistaken edit';
    await wrapper.setProps({ visible: false });
    await wrapper.vm.$nextTick();

    // Reopen the same target: stale draft must not bleed across opens.
    await wrapper.setProps({ visible: true });
    await wrapper.vm.$nextTick();

    expect(vm.draft.releaseReasonCode).toBe('manual_pardon');
    expect(vm.draft.releaseReason).toBe('');
  });
});
