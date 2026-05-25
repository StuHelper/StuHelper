import { readdirSync } from 'node:fs';
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
});
