import type { InjectionKey } from 'vue';

export interface PersistentAdminTableContext {
  columnWidth: (columnKey: string) => number | undefined;
}

export const persistentAdminTableKey: InjectionKey<PersistentAdminTableContext> =
  Symbol('PersistentAdminTable');
