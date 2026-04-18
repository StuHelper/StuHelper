#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..', '..');
const bundledPath = path.join(repoRoot, 'server', 'api', 'openapi.bundled.yaml');
const overviewPath = path.join(repoRoot, 'docs', 'references', 'api-overview.md');

function collectSpecPaths(text) {
  const paths = new Set();
  const regex = /^  (\/[^:\n]+):\s*$/gm;
  for (const match of text.matchAll(regex)) {
    paths.add(match[1].trim());
  }
  return paths;
}

function collectOverviewPaths(text) {
  const paths = new Set();
  const regex = /^\|\s*`?(\/[^|`]+?)`?\s*\|/gm;
  for (const match of text.matchAll(regex)) {
    paths.add(match[1].trim());
  }
  return paths;
}

function sortedDiff(left, right) {
  return [...left].filter((item) => !right.has(item)).sort();
}

const bundled = fs.readFileSync(bundledPath, 'utf8');
const overview = fs.readFileSync(overviewPath, 'utf8');

const specPaths = collectSpecPaths(bundled);
const overviewPaths = collectOverviewPaths(overview);

const missingFromDoc = sortedDiff(specPaths, overviewPaths);
const extraInDoc = sortedDiff(overviewPaths, specPaths);

if (missingFromDoc.length === 0 && extraInDoc.length === 0) {
  console.log(`[check-api-overview-sync] OK: ${specPaths.size} routes in sync`);
  process.exit(0);
}

if (missingFromDoc.length > 0) {
  console.error('[check-api-overview-sync] Missing from docs/references/api-overview.md:');
  for (const item of missingFromDoc) console.error(`  - ${item}`);
}
if (extraInDoc.length > 0) {
  console.error('[check-api-overview-sync] Present in docs/references/api-overview.md but absent from OpenAPI:');
  for (const item of extraInDoc) console.error(`  - ${item}`);
}
process.exit(1);
