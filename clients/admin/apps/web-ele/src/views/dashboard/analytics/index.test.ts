import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { describe, expect, it } from 'vitest';

const appRoot = process.cwd();
const analyticsDir = join(appRoot, 'src/views/dashboard/analytics');

function readAdminLocale(locale: 'en-US' | 'zh-CN') {
  return JSON.parse(
    readFileSync(
      join(appRoot, `src/locales/langs/${locale}/admin.json`),
      'utf8',
    ),
  ) as {
    dashboard: Record<string, unknown> & {
      analytics: Record<string, unknown>;
    };
    layout?: unknown;
  };
}

describe('admin analytics dashboard source', () => {
  it('keeps only the active StuHelper analytics page in the dashboard directory', () => {
    const vueFiles = readdirSync(analyticsDir)
      .filter((file) => file.endsWith('.vue'))
      .toSorted();

    expect(vueFiles).toEqual(['index.vue']);
  });

  it('keeps analytics locale keys scoped to live moderation metrics', () => {
    for (const locale of ['zh-CN', 'en-US'] as const) {
      const analytics = readAdminLocale(locale).dashboard.analytics;

      expect(Object.keys(analytics).toSorted()).toEqual(['overview']);
    }
  });

  it('does not keep unused upstream dashboard and notification locale demos', () => {
    for (const locale of ['zh-CN', 'en-US'] as const) {
      const admin = readAdminLocale(locale);

      expect(admin.layout).toBeUndefined();
      expect(Object.keys(admin.dashboard).toSorted()).toEqual([
        'analytics',
        'quickActions',
        'summary',
      ]);
    }
  });
});
