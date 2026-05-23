import { readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const clientsRoot = path.resolve(scriptDir, '..');

const roots = ['web', 'uniappx', 'shared', 'admin/apps/web-ele'];
const ignoredDirs = new Set([
  '.turbo',
  '_archived',
  'coverage',
  'dist',
  'node_modules',
  'storybook-static',
]);
const checkedExtensions = new Set([
  '.cjs',
  '.js',
  '.jsx',
  '.mjs',
  '.ts',
  '.tsx',
  '.vue',
]);
const emptyCatchPattern = /catch\s*(?:\([^)]*\))?\s*\{\s*\}/g;

const findings = [];

for (const root of roots) {
  walk(path.join(clientsRoot, root));
}

if (findings.length > 0) {
  for (const finding of findings) {
    console.error(`${finding.file}:${finding.line}:${finding.column}: empty catch block`);
  }
  console.error('[check-no-empty-catch] empty catch block detected');
  process.exit(1);
}

console.log('[check-no-empty-catch] OK: no empty catch blocks found');

function walk(target) {
  let stats;
  try {
    stats = statSync(target);
  } catch {
    return;
  }

  if (stats.isDirectory()) {
    if (ignoredDirs.has(path.basename(target))) {
      return;
    }
    for (const entry of readdirSync(target)) {
      walk(path.join(target, entry));
    }
    return;
  }

  if (!stats.isFile() || !checkedExtensions.has(path.extname(target))) {
    return;
  }

  const source = readFileSync(target, 'utf8');
  for (const match of source.matchAll(emptyCatchPattern)) {
    const position = lineColumn(source, match.index ?? 0);
    findings.push({
      file: path.relative(clientsRoot, target).replaceAll(path.sep, '/'),
      line: position.line,
      column: position.column,
    });
  }
}

function lineColumn(source, index) {
  let line = 1;
  let column = 1;
  for (let i = 0; i < index; i += 1) {
    if (source.charCodeAt(i) === 10) {
      line += 1;
      column = 1;
    } else {
      column += 1;
    }
  }
  return { line, column };
}
