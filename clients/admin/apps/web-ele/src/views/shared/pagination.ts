/**
 * Shared pagination contract for admin list pages.
 *
 * Every list view renders the same ElPagination layout (with a page-size
 * selector) so paging behaves identically across the admin surface.
 */

export const ADMIN_PAGE_SIZES: number[] = [10, 20, 50, 100];

export const ADMIN_DEFAULT_PAGE_SIZE = 20;

export const ADMIN_PAGINATION_LAYOUT = 'total, prev, pager, next, sizes';
