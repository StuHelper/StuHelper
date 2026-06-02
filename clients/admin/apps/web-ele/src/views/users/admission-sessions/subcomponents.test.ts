// @vitest-environment happy-dom

import type { AdmissionSession } from '#/api/admin';

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';

import { describe, expect, it } from 'vitest';

import AdmissionSessionFilters from './AdmissionSessionFilters.vue';
import AdmissionSessionTable from './AdmissionSessionTable.vue';
import { statusLabel, statusTagType } from './options';

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

const tableStubs = {
  ElPagination: defineComponent({
    name: 'ElPagination',
    template: '<div data-stub="el-pagination" />',
  }),
  PersistentAdminTable: PersistentAdminTableStub,
  PersistentAdminTableColumn: PersistentAdminTableColumnStub,
};

const baseSession: AdmissionSession = {
  id: 'session-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: '178037297',
  channelID: '178037297',
  qqID: '1390191645',
  userID: '42',
  status: 'linked',
  tokenExpiresAt: '2026-06-02T06:00:00Z',
  tokenConsumedAt: '2026-06-02T05:00:00Z',
  linkWaitDeadlineAt: '2026-06-02T06:00:00Z',
  submissionWaitDeadlineAt: '2026-06-02T07:00:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-07-02T05:00:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: 'unmute failed',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/token?qq=1390191645',
};

describe('AdmissionSessionFilters', () => {
  it('emits search/reset and includes runtime filter fields', async () => {
    const wrapper = mount(AdmissionSessionFilters, {
      props: {
        botSelfID: '',
        guildID: '',
        platform: 'qq',
        qqID: '',
        status: '',
      },
    });

    expect(wrapper.find('[data-field="qqID"]').exists()).toBe(true);
    expect(wrapper.find('[data-field="guildID"]').exists()).toBe(true);
    expect(wrapper.find('[data-field="botSelfID"]').exists()).toBe(true);
    expect(wrapper.find('[data-field="platform"]').exists()).toBe(true);
    expect(wrapper.find('[data-field="status"]').exists()).toBe(true);

    const buttons = wrapper.findAll('button');
    const search = buttons.find((btn) => btn.text() === '查询');
    const reset = buttons.find((btn) => btn.text() === '重置');
    if (!search || !reset) {
      throw new Error('expected search and reset buttons');
    }

    await search.trigger('click');
    await reset.trigger('click');

    expect(wrapper.emitted('search')).toHaveLength(1);
    expect(wrapper.emitted('reset')).toHaveLength(1);
  });
});

describe('AdmissionSessionTable', () => {
  it('renders session diagnostics and emits copyAuthURL for canonical links', async () => {
    const wrapper = mount(AdmissionSessionTable, {
      props: {
        items: [baseSession],
        loading: false,
        page: 1,
        pageSize: 20,
        total: 1,
      },
      global: { stubs: tableStubs },
    });

    expect(wrapper.text()).toContain('1390191645');
    expect(wrapper.text()).toContain('178037297');
    expect(wrapper.text()).toContain('2118785781');
    expect(wrapper.text()).toContain('unmute failed');

    const copy = wrapper.find('[data-action="copyAuthURL"]');
    expect(copy.exists()).toBe(true);
    await copy.trigger('click');

    expect(wrapper.emitted('copyAuthURL')?.[0]).toEqual([baseSession.authURL]);
  });

  it('uses operator-facing status labels and tag severity', () => {
    expect(statusLabel('joined_muted')).toBe('已入群禁言');
    expect(statusTagType('joined_muted')).toBe('danger');
    expect(statusLabel('verified')).toBe('已通过');
    expect(statusTagType('verified')).toBe('success');
  });
});
