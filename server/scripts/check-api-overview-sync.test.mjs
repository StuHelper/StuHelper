import assert from 'node:assert/strict';
import test from 'node:test';

import {
  collectOverviewEntries,
  findCoverageIssues,
  patternMatchesPath,
} from './check-api-overview-sync.mjs';

test('patternMatchesPath accepts collection endpoints and nested paths for terminal star patterns', () => {
  assert.equal(
    patternMatchesPath(
      '/api/v1/course/review/user/notifications/*',
      '/api/v1/course/review/user/notifications',
    ),
    true,
  );
  assert.equal(
    patternMatchesPath(
      '/api/v1/course/review/user/notifications/*',
      '/api/v1/course/review/user/notifications/stream',
    ),
    true,
  );
  assert.equal(
    patternMatchesPath('/api/v1/course/review/user/notifications/*', '/api/v1/course/review/user'),
    false,
  );
});

test('collectOverviewEntries extracts module prefixes from the summary table', () => {
  const overview = `| 模块 | 前缀 | 权威规格 |
|------|------|----------|
| 资源共享 | \`/api/v1/resources/*\` | ref |
| 通知 | \`/api/v1/course/review/user/notifications/*\` | ref |`;

  assert.deepEqual(collectOverviewEntries(overview), [
    { module: '资源共享', patterns: ['/api/v1/resources/*'] },
    { module: '通知', patterns: ['/api/v1/course/review/user/notifications/*'] },
  ]);
});

test('findCoverageIssues reports stale prefixes and uncovered OpenAPI paths', () => {
  const specPaths = [
    '/api/v1/resources',
    '/api/v1/resources/42/download-url',
    '/api/v1/course/review/user/notifications',
    '/api/v1/course/review/user/notifications/stream',
    '/api/v1/metrics/vitals',
  ];
  const overviewEntries = [
    { module: '资源共享', patterns: ['/api/v1/resource/*'] },
    { module: '通知', patterns: ['/api/v1/course/review/user/notifications/*'] },
    { module: '审计', patterns: ['/api/v1/admin/audit/*'] },
    { module: '指标采集', patterns: ['/api/v1/metrics/*'] },
  ];

  const issues = findCoverageIssues(specPaths, overviewEntries);

  assert.deepEqual(issues.unmatchedDocPatterns, [
    { module: '资源共享', pattern: '/api/v1/resource/*' },
    { module: '审计', pattern: '/api/v1/admin/audit/*' },
  ]);
  assert.deepEqual(issues.uncoveredSpecPaths, [
    '/api/v1/resources',
    '/api/v1/resources/42/download-url',
  ]);
});
