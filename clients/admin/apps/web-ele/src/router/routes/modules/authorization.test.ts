import { describe, expect, it, vi } from 'vitest';

import routes from './authorization';

vi.mock('#/locales', () => ({
  $t: (key: string) => key,
}));

describe('authorization admin routes', () => {
  it('gates the ledger and projection controls with the global IAM capability', () => {
    const parent = routes[0];
    expect(parent?.path).toBe('/authorization');
    expect(parent?.meta?.authority).toEqual(['iam:grants:manage']);

    const ledger = parent?.children?.[0];
    expect(ledger?.name).toBe('AuthorizationGrants');
    expect(ledger?.path).toBe('/authorization/grants');
    expect(ledger?.meta?.authority).toEqual(['iam:grants:manage']);
    expect(ledger?.component).toBeTypeOf('function');
  });
});
