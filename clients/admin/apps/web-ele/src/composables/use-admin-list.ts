import type { ShallowRef } from 'vue';
import type { LocationQuery } from 'vue-router';

import { onMounted, reactive, ref, shallowRef } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { $t } from '#/locales';
import { adminLogger } from '#/utils/admin-logger';

/**
 * Normalizes unknown failures into a user-facing message.
 *
 * The API layer throws typed `Error`s with translated messages; anything else
 * falls back to the generic request-failed copy.
 */
export function adminErrorMessage(error: unknown): string {
  return error instanceof Error && error.message
    ? error.message
    : $t('admin.result.requestFailed');
}

export interface AdminListQueryBase {
  page: number;
  pageSize: number;
}

export interface AdminListResult<TItem> {
  items: TItem[];
  total: number;
}

export interface UseAdminListOptions<TItem, TQuery extends AdminListQueryBase> {
  /**
   * Fetch the first page on mount. Default true. Views whose query requires
   * user-provided filters (e.g. consents) opt out and fetch on demand.
   */
  fetchOnMount?: boolean;
  /** Loads one page of data for the current query. */
  fetcher: (query: TQuery) => Promise<AdminListResult<TItem>>;
  /** Default query values; also defines which keys sync to the URL. */
  initialQuery: TQuery;
  /** Mirror filters/pagination into the route query. Default true. */
  syncRouteQuery?: boolean;
}

/**
 * List-page state machine shared by all admin list views.
 *
 * Encapsulates the fetch race guard (`fetchRequestSeq`), loading/loadError
 * state, pagination, and filter/pagination deep-linking via `router.replace`.
 * Error presentation stays in the view layer; this composable only records
 * the message.
 */
export function useAdminList<TItem, TQuery extends AdminListQueryBase>(
  options: UseAdminListOptions<TItem, TQuery>,
) {
  const route = useRoute();
  const router = useRouter();
  const syncRouteQuery = options.syncRouteQuery ?? true;
  const defaults: TQuery = { ...options.initialQuery };

  const query = reactive({
    ...defaults,
    ...(syncRouteQuery ? parseRouteQuery(route.query, defaults) : {}),
  }) as TQuery;
  const items = shallowRef<TItem[]>([]) as ShallowRef<TItem[]>;
  const total = ref(0);
  const loading = ref(false);
  const loadError = ref('');
  let fetchRequestSeq = 0;

  function buildRouteQuery(): LocationQuery {
    const preserved: LocationQuery = Object.fromEntries(
      Object.entries(route.query).filter(([key]) => !(key in defaults)),
    );
    const owned: LocationQuery = {};
    for (const key of Object.keys(defaults)) {
      const value = (query as Record<string, unknown>)[key];
      if (
        value === (defaults as Record<string, unknown>)[key] ||
        value === '' ||
        value === null ||
        value === undefined
      ) {
        continue;
      }
      owned[key] = String(value);
    }
    return { ...preserved, ...owned };
  }

  function syncQueryToRoute() {
    if (!syncRouteQuery) {
      return;
    }
    const next = buildRouteQuery();
    if (isSameRouteQuery(next, route.query)) {
      return;
    }
    router.replace({ query: next }).catch((error: unknown) => {
      adminLogger.warn('admin-list-route-sync-failed', error, {
        path: route.path,
      });
    });
  }

  async function fetchData() {
    const requestSeq = ++fetchRequestSeq;
    loading.value = true;
    loadError.value = '';
    syncQueryToRoute();
    try {
      const data = await options.fetcher(query);
      if (requestSeq !== fetchRequestSeq) return;
      items.value = data.items;
      total.value = data.total;
    } catch (error) {
      if (requestSeq !== fetchRequestSeq) return;
      loadError.value = adminErrorMessage(error);
    } finally {
      if (requestSeq === fetchRequestSeq) {
        loading.value = false;
      }
    }
  }

  function resetPageAndFetch() {
    query.page = 1;
    void fetchData();
  }

  onMounted(() => {
    if (options.fetchOnMount ?? true) {
      void fetchData();
    }
  });

  return {
    fetchData,
    items,
    loadError,
    loading,
    query,
    resetPageAndFetch,
    total,
  };
}

/**
 * Race-guarded loader for single-resource pages (e.g. dashboard stats).
 */
export function useAdminLoad<T>(loader: () => Promise<T>) {
  const data = shallowRef<null | T>(null) as ShallowRef<null | T>;
  const loading = ref(true);
  const loadError = ref('');
  let loadRequestSeq = 0;

  async function load() {
    const requestSeq = ++loadRequestSeq;
    loading.value = true;
    loadError.value = '';
    try {
      const result = await loader();
      if (requestSeq !== loadRequestSeq) return;
      data.value = result;
    } catch (error) {
      if (requestSeq !== loadRequestSeq) return;
      loadError.value = adminErrorMessage(error);
    } finally {
      if (requestSeq === loadRequestSeq) {
        loading.value = false;
      }
    }
  }

  onMounted(() => {
    void load();
  });

  return { data, load, loadError, loading };
}

function isSameRouteQuery(a: LocationQuery, b: LocationQuery): boolean {
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  return (
    aKeys.length === bKeys.length && aKeys.every((key) => a[key] === b[key])
  );
}

function parseRouteQuery<TQuery extends AdminListQueryBase>(
  routeQuery: LocationQuery,
  defaults: TQuery,
): Partial<TQuery> {
  const parsed: Record<string, unknown> = {};
  for (const [key, defaultValue] of Object.entries(defaults)) {
    const raw = routeQuery[key];
    if (typeof raw !== 'string' || raw === '') {
      continue;
    }
    const value = coerceQueryValue(raw, defaultValue);
    if (value !== undefined) {
      parsed[key] = value;
    }
  }

  const page = parsed.page;
  if (typeof page === 'number' && (!Number.isInteger(page) || page < 1)) {
    delete parsed.page;
  }
  const pageSize = parsed.pageSize;
  if (
    typeof pageSize === 'number' &&
    (!Number.isInteger(pageSize) || pageSize < 1)
  ) {
    delete parsed.pageSize;
  }

  return parsed as Partial<TQuery>;
}

function coerceQueryValue(raw: string, defaultValue: unknown): unknown {
  if (typeof defaultValue === 'number') {
    const numeric = Number(raw);
    return Number.isFinite(numeric) ? numeric : undefined;
  }
  if (typeof defaultValue === 'boolean') {
    if (raw === 'true') return true;
    if (raw === 'false') return false;
    return undefined;
  }
  if (typeof defaultValue === 'string') {
    return raw;
  }
  if (defaultValue === null) {
    const numeric = Number(raw);
    return Number.isFinite(numeric) ? numeric : raw;
  }
  return undefined;
}
