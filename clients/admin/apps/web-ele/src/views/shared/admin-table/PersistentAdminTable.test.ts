// @vitest-environment happy-dom

import { mount } from '@vue/test-utils';
import { defineComponent, h } from 'vue';

import { beforeEach, describe, expect, it } from 'vitest';

import PersistentAdminTable from './PersistentAdminTable.vue';
import PersistentAdminTableColumn from './PersistentAdminTableColumn.vue';

const STORAGE_KEY = 'stuhelper.admin.tableColumns.spec';

const ElTableStub = defineComponent({
  name: 'ElTable',
  emits: ['header-dragend'],
  setup(_, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          'data-stub': 'el-table',
          style: attrs.style as Record<string, string>,
          type: 'button',
          onClick: () =>
            emit(
              'header-dragend',
              123.6,
              80,
              { columnKey: 'courseName' },
              new MouseEvent('mouseup'),
            ),
        },
        slots.default?.(),
      );
  },
});

const ElTableColumnStub = defineComponent({
  name: 'ElTableColumn',
  props: {
    columnKey: { type: String, required: true },
    width: { type: [Number, String], default: undefined },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-column-key': props.columnKey,
          'data-stub': 'el-table-column',
          'data-width': String(props.width ?? ''),
        },
        slots.default?.(),
      );
  },
});

const mountOptions = {
  global: {
    directives: { loading: { mounted() {}, updated() {} } },
    stubs: {
      ElTable: ElTableStub,
      ElTableColumn: ElTableColumnStub,
    },
  },
};

function createStorage(): Storage {
  const entries = new Map<string, string>();
  return {
    clear: () => entries.clear(),
    getItem: (key) => entries.get(key) ?? null,
    key: (index) => [...entries.keys()][index] ?? null,
    removeItem: (key) => entries.delete(key),
    setItem: (key, value) => entries.set(key, value),
    get length() {
      return entries.size;
    },
  };
}

describe('PersistentAdminTable', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: createStorage(),
    });
  });

  it('stores dragged column widths in localStorage', async () => {
    const wrapper = mount(PersistentAdminTable, {
      ...mountOptions,
      props: { tableKey: 'spec' },
    });

    await wrapper.find('[data-stub="el-table"]').trigger('click');

    expect(window.localStorage.getItem(STORAGE_KEY)).toBe('{"courseName":124}');
  });

  it('keeps the table root at container width for fixed columns', () => {
    const wrapper = mount(PersistentAdminTable, {
      ...mountOptions,
      props: { tableKey: 'spec' },
    });

    const style = wrapper.find('[data-stub="el-table"]').attributes('style');

    expect(style).toContain('width: 100%');
    expect(style).not.toContain('min-width');
  });

  it('hydrates column widths from localStorage', () => {
    window.localStorage.setItem(STORAGE_KEY, '{"courseName":220}');
    const wrapper = mount(
      {
        components: { PersistentAdminTable, PersistentAdminTableColumn },
        template: `
          <PersistentAdminTable table-key="spec">
            <PersistentAdminTableColumn
              column-key="courseName"
              :default-width="140"
              label="课程"
            />
          </PersistentAdminTable>
        `,
      },
      mountOptions,
    );

    expect(
      wrapper.find('[data-column-key="courseName"]').attributes(),
    ).toMatchObject({ 'data-width': '220' });
  });

  it('clears corrupt stored widths instead of crashing the table', () => {
    window.localStorage.setItem(STORAGE_KEY, '{bad-json');

    expect(() =>
      mount(PersistentAdminTable, {
        ...mountOptions,
        props: { tableKey: 'spec' },
      }),
    ).not.toThrow();
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
  });
});
