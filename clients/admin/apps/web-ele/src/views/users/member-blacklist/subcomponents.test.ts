// @vitest-environment happy-dom

import type { MemberBlacklistEntry } from '#/api/admin';

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';

import { describe, expect, it, vi } from 'vitest';

import BlacklistFilters from './BlacklistFilters.vue';
import BlacklistTable from './BlacklistTable.vue';

vi.mock('#/locales', () => ({
  $t: (key: string, params?: Record<string, unknown>) =>
    params ? `${key}:${Object.values(params).join('/')}` : key,
}));

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
const PersistentAdminTableStub = defineComponent({
  name: 'PersistentAdminTable',
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () => {
      const cols = slots.default?.() ?? [];
      return h(
        'div',
        { 'data-stub': 'persistent-admin-table' },
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
const PersistentAdminTableColumnStub = defineComponent({
  name: 'PersistentAdminTableColumn',
  setup(_, { slots }) {
    return () =>
      h(
        'div',
        { 'data-stub': 'persistent-admin-table-column' },
        slots.default?.(),
      );
  },
});
const ElPaginationStub = defineComponent({
  name: 'ElPagination',
  template: '<div data-stub="el-pagination" />',
});

const tableStubs = {
  ElTable: ElTableStub,
  ElTableColumn: ElTableColumnStub,
  PersistentAdminTable: PersistentAdminTableStub,
  PersistentAdminTableColumn: PersistentAdminTableColumnStub,
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
    const search = buttons.find((btn) => btn.text() === 'admin.common.query');
    const reset = buttons.find((btn) => btn.text() === 'admin.common.reset');
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
    const [scopeSelect, sourceSelect, statusSelect] = selects;
    if (!scopeSelect || !sourceSelect || !statusSelect) {
      throw new Error('expected three select filters');
    }

    await scopeSelect.vm.$emit('change', 'global');
    await sourceSelect.vm.$emit('change', 'manual_admin');
    await statusSelect.vm.$emit('change', 'released');

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
          PersistentAdminTable: PersistentAdminTableStub,
          PersistentAdminTableColumn: PersistentAdminTableColumnStub,
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
          PersistentAdminTable: PersistentAdminTableStub,
          PersistentAdminTableColumn: PersistentAdminTableColumnStub,
          ElPagination: ElPaginationStub,
        },
        directives: { loading: tableStubs.vLoading },
      },
    });

    expect(wrapper.find('[data-action="release"]').exists()).toBe(false);
  });
});
