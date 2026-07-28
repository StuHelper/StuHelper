import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import {
  collectRelativeMarkdownLinks,
  parseFrontmatter,
  validateDocsTree,
} from './check-docs-hygiene.mjs';

function writeFile(root, relativePath, content) {
  const fullPath = path.join(root, relativePath);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, content);
}

test('parseFrontmatter extracts required metadata fields', () => {
  const doc = `---
type: design
audience: backend-dev, frontend-dev
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# Title
`;

  assert.deepEqual(parseFrontmatter(doc), {
    type: 'design',
    audience: 'backend-dev, frontend-dev',
    status: 'current',
    'authoritative-source': 'this file',
    'last-verified': '2026-04-19',
  });
});

test('collectRelativeMarkdownLinks skips anchors and external links', () => {
  const doc = [
    '[治理](../design/documentation-governance.md)',
    '[根锚点](#section)',
    '[外链](https://example.com)',
    '[目录](../guides/)',
  ].join('\n');

  assert.deepEqual(collectRelativeMarkdownLinks(doc), [
    '../design/documentation-governance.md',
    '../guides/',
  ]);
});

test('validateDocsTree reports metadata drift, broken links and retired paths', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-docs-'));

  writeFile(
    repoRoot,
    'docs/README.md',
    `---
type: reference
audience: all
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

[治理](design/documentation-governance.md)
`,
  );
  writeFile(
    repoRoot,
    'docs/design/documentation-governance.md',
    `---
type: design
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

[维护指南](../guides/documentation-maintenance.md)
`,
  );
  writeFile(
    repoRoot,
    'docs/guides/documentation-maintenance.md',
    `---
type: design
audience: maintainers
status: snapshot
authoritative-source: this file
last-verified: 2026-04-19
---

[旧入口](../BACKEND.md)
[丢失链接](../missing.md)
`,
  );

  const issues = validateDocsTree(repoRoot);

  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/guides/documentation-maintenance.md: frontmatter `type` must be `guide`'),
    ),
  );
  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/guides/documentation-maintenance.md: frontmatter `status` must be one of current, draft, deprecated'),
    ),
  );
  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/guides/documentation-maintenance.md: broken relative link -> ../missing.md'),
    ),
  );
  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/guides/documentation-maintenance.md: references retired path -> ../BACKEND.md'),
    ),
  );
});

test('validateDocsTree rejects retired superpowers docs paths', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-docs-'));

  writeFile(
    repoRoot,
    'docs/superpowers/specs/legacy.md',
    `---
type: design
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-05-07
---

# Legacy Spec
`,
  );

  const issues = validateDocsTree(repoRoot);

  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/superpowers/specs/legacy.md: unexpected top-level docs location'),
    ),
  );
  assert.ok(
    issues.some((issue) =>
      issue.includes('docs/superpowers/specs/legacy.md: retired docs location'),
    ),
  );
});

test('validateDocsTree skips ignored runtime directories', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-docs-'));
  const runtimeDir = path.join(repoRoot, '.deploy', 'vault', 'file');

  writeFile(
    repoRoot,
    'docs/README.md',
    `---
type: reference
audience: all
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# Documentation
`,
  );
  writeFile(repoRoot, '.deploy/vault/file/.DS_Store', 'runtime-only');
  fs.chmodSync(runtimeDir, 0o000);

  try {
    assert.deepEqual(validateDocsTree(repoRoot), []);
  } finally {
    fs.chmodSync(runtimeDir, 0o700);
    fs.rmSync(repoRoot, { force: true, recursive: true });
  }
});
