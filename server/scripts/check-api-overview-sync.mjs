#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptPath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(scriptPath), '..', '..');
const bundledPath = path.join(repoRoot, 'server', 'api', 'openapi.bundled.yaml');
const overviewPath = path.join(repoRoot, 'docs', 'reference', 'api-overview.md');

export function collectSpecPaths(text) {
  const paths = new Set();
  const regex = /^  (\/[^:\n]+):\s*$/gm;
  for (const match of text.matchAll(regex)) {
    paths.add(match[1].trim());
  }
  return [...paths];
}

export function collectOverviewEntries(text) {
  const entries = [];
  const rowRegex = /^\|\s*([^|\n]+?)\s*\|\s*([^|\n]+?)\s*\|\s*([^|\n]+?)\s*\|$/gm;
  for (const match of text.matchAll(rowRegex)) {
    const module = match[1].trim();
    const prefixCell = match[2].trim();
    if (module === '模块' || module.startsWith('---')) {
      continue;
    }
    const patterns = [...prefixCell.matchAll(/`(\/[^`]+)`/g)].map((item) => item[1].trim());
    if (patterns.length === 0) {
      continue;
    }
    entries.push({ module, patterns });
  }
  return entries;
}

function escapeForRegExp(value) {
  return value.replace(/[|\\{}()[\]^$+?.]/g, '\\$&');
}

export function patternMatchesPath(pattern, specPath) {
  if (pattern.endsWith('/*')) {
    const base = escapeForRegExp(pattern.slice(0, -2));
    return new RegExp(`^${base}(?:/.*)?$`).test(specPath);
  }
  const regexPattern = escapeForRegExp(pattern).replaceAll('\\*', '.*');
  return new RegExp(`^${regexPattern}$`).test(specPath);
}

export function findCoverageIssues(specPaths, overviewEntries) {
  const normalizedPaths = [...specPaths];
  const documentedPatterns = overviewEntries.flatMap((entry) =>
    entry.patterns.map((pattern) => ({ module: entry.module, pattern })),
  );
  const unmatchedDocPatterns = documentedPatterns.filter(
    ({ pattern }) => !normalizedPaths.some((specPath) => patternMatchesPath(pattern, specPath)),
  );
  const uncoveredSpecPaths = normalizedPaths.filter(
    (specPath) => !documentedPatterns.some(({ pattern }) => patternMatchesPath(pattern, specPath)),
  );
  return { unmatchedDocPatterns, uncoveredSpecPaths, documentedPatterns };
}

export function checkApiOverviewSync({
  bundledSpecPath = bundledPath,
  apiOverviewPath = overviewPath,
} = {}) {
  const bundled = fs.readFileSync(bundledSpecPath, 'utf8');
  const overview = fs.readFileSync(apiOverviewPath, 'utf8');
  const specPaths = collectSpecPaths(bundled);
  const overviewEntries = collectOverviewEntries(overview);
  return {
    specPaths,
    overviewEntries,
    ...findCoverageIssues(specPaths, overviewEntries),
  };
}

function runCli() {
  const { specPaths, documentedPatterns, unmatchedDocPatterns, uncoveredSpecPaths } =
    checkApiOverviewSync();

  if (unmatchedDocPatterns.length === 0 && uncoveredSpecPaths.length === 0) {
    console.log(
      `[check-api-overview-sync] OK: ${specPaths.length} OpenAPI paths covered by ${documentedPatterns.length} documented prefixes`,
    );
    process.exit(0);
  }

  if (unmatchedDocPatterns.length > 0) {
    console.error('[check-api-overview-sync] Documented prefixes without matching OpenAPI paths:');
    for (const { module, pattern } of unmatchedDocPatterns) {
      console.error(`  - ${module}: ${pattern}`);
    }
  }

  if (uncoveredSpecPaths.length > 0) {
    console.error('[check-api-overview-sync] OpenAPI paths not covered by docs/reference/api-overview.md:');
    for (const specPath of uncoveredSpecPaths) {
      console.error(`  - ${specPath}`);
    }
  }

  process.exit(1);
}

if (process.argv[1] && path.resolve(process.argv[1]) === scriptPath) {
  runCli();
}
