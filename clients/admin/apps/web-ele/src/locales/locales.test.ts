import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const appRoot = process.cwd();
const localeRoot = join(appRoot, 'src/locales/langs');

describe('admin locale source', () => {
  it('loads only active application locale namespaces', () => {
    for (const locale of ['zh-CN', 'en-US'] as const) {
      const files = readdirSync(join(localeRoot, locale))
        .filter((file) => file.endsWith('.json'))
        .toSorted();

      expect(files).toEqual(['admin.json', 'page.json']);
    }
  });

  it('localizes every student verification method emitted by the API', () => {
    for (const locale of ['zh-CN', 'en-US'] as const) {
      const messages = JSON.parse(
        readFileSync(join(localeRoot, locale, 'admin.json'), 'utf8'),
      ) as {
        users?: { studentVerification?: { method?: Record<string, unknown> } };
      };
      const methods = messages.users?.studentVerification?.method;

      for (const method of [
        'ldap',
        'manual',
        'school_email_otp',
        'school_sso',
      ]) {
        expect(methods?.[method], `${locale} ${method}`).toEqual(
          expect.any(String),
        );
      }
    }
  });
});
