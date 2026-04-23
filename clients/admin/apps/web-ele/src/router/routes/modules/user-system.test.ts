import { describe, expect, it, vi } from 'vitest';

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
}));

import routes from './user-system';

describe('user-system routes', () => {
  it('lets scoped admin entries keep the parent menu visible', () => {
    const route = routes[0]!;

    expect(route.meta?.authority).toEqual(
      expect.arrayContaining([
        'user:identity:read',
        'user:identity:review',
        'user:student:review',
        'user:school:read',
        'user:system:read',
      ]),
    );
  });
});
