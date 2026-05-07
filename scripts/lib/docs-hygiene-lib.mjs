import fs from 'node:fs';
import path from 'node:path';

const REQUIRED_FRONTMATTER_KEYS = [
  'type',
  'audience',
  'status',
  'authoritative-source',
  'last-verified',
];
const ALLOWED_AUDIENCES = new Set([
  'all',
  'backend-dev',
  'frontend-dev',
  'ops',
  'product',
  'qa',
  'maintainers',
]);
const ALLOWED_TYPES = new Set(['guide', 'design', 'product-spec', 'reference', 'adr', 'internal']);
const LONG_TERM_STATUSES = new Set(['current', 'draft', 'deprecated']);
const INTERNAL_STATUSES = new Set(['current', 'snapshot', 'archived']);
const TOP_LEVEL_ALLOWED = new Set([
  'docs/README.md',
  'docs/QUICKSTART.md',
  'docs/adr',
  'docs/design',
  'docs/guides',
  'docs/internal',
  'docs/product-specs',
  'docs/reference',
]);
const LONG_TERM_ASSET_ALLOWLIST = new Set(['docs/design/openfga-model.fga']);
const RETIRED_PATH_PATTERNS = [
  /docs\/(BACKEND|FRONTEND|PRODUCT|SECURITY|QUALITY_SCORE)\.md/,
  /docs\/design-docs\//,
  /docs\/architecture\//,
  /docs\/operations\//,
  /docs\/references\//,
  /docs\/exec-plans\//,
  /docs\/superpowers\//,
  /docs\/internal\/superpowers\//,
  /docs\/product-specs\/(auth-sso|rbac-authorization|storage-driver-architecture)/,
];
const ABSOLUTE_PATH_PATTERN = /\/Users\/|\/home\/|\/root\/|C:\\/;
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const SKIP_DIRS = new Set(['.git', 'node_modules']);

function toPosix(value) {
  return value.split(path.sep).join(path.posix.sep);
}

function walkFiles(rootDir, predicate = () => true) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }
  const results = [];
  const stack = [rootDir];
  while (stack.length > 0) {
    const current = stack.pop();
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        if (!SKIP_DIRS.has(entry.name)) {
          stack.push(path.join(current, entry.name));
        }
        continue;
      }
      const fullPath = path.join(current, entry.name);
      if (predicate(fullPath)) {
        results.push(fullPath);
      }
    }
  }
  return results.sort();
}

function relativePath(root, target) {
  return toPosix(path.relative(root, target));
}

function expectedTypeFor(relativeFile) {
  if (relativeFile === 'docs/README.md') {
    return 'reference';
  }
  if (relativeFile === 'docs/QUICKSTART.md') {
    return 'guide';
  }
  if (relativeFile === 'docs/adr/README.md' || relativeFile === 'docs/adr/template.md') {
    return 'reference';
  }
  const [, dirName] = relativeFile.split('/');
  return {
    adr: 'adr',
    design: 'design',
    guides: 'guide',
    internal: 'internal',
    'product-specs': 'product-spec',
    reference: 'reference',
  }[dirName];
}

function expectedStatusesFor(relativeFile) {
  return relativeFile.startsWith('docs/internal/') ? INTERNAL_STATUSES : LONG_TERM_STATUSES;
}

function isValidDate(value) {
  if (!DATE_PATTERN.test(value)) {
    return false;
  }
  return !Number.isNaN(Date.parse(`${value}T00:00:00Z`));
}

export function parseFrontmatter(text) {
  if (!text.startsWith('---\n')) {
    return null;
  }
  const endMarker = text.indexOf('\n---\n', 4);
  if (endMarker === -1) {
    return null;
  }
  const block = text.slice(4, endMarker).trim();
  if (block.length === 0) {
    return {};
  }
  const metadata = {};
  for (const line of block.split('\n')) {
    const separator = line.indexOf(':');
    if (separator === -1) {
      continue;
    }
    const key = line.slice(0, separator).trim();
    const value = line.slice(separator + 1).trim();
    metadata[key] = value;
  }
  return metadata;
}

function parseAudienceList(value) {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function validateMetadata(relativeFile, metadata) {
  const issues = [];
  if (!metadata) {
    return [`${relativeFile}: missing YAML frontmatter`];
  }
  for (const key of REQUIRED_FRONTMATTER_KEYS) {
    if (!metadata[key]) {
      issues.push(`${relativeFile}: missing frontmatter \`${key}\``);
    }
  }
  if (!metadata.type || !ALLOWED_TYPES.has(metadata.type)) {
    issues.push(`${relativeFile}: frontmatter \`type\` must be one of ${[...ALLOWED_TYPES].join(', ')}`);
  }
  const expectedType = expectedTypeFor(relativeFile);
  if (expectedType && metadata.type && metadata.type !== expectedType) {
    issues.push(`${relativeFile}: frontmatter \`type\` must be \`${expectedType}\``);
  }
  const audiences = metadata.audience ? parseAudienceList(metadata.audience) : [];
  if (audiences.length === 0) {
    issues.push(`${relativeFile}: frontmatter \`audience\` must not be empty`);
  }
  for (const audience of audiences) {
    if (!ALLOWED_AUDIENCES.has(audience)) {
      issues.push(`${relativeFile}: unsupported audience \`${audience}\``);
    }
  }
  const allowedStatuses = expectedStatusesFor(relativeFile);
  if (metadata.status && !allowedStatuses.has(metadata.status)) {
    issues.push(
      `${relativeFile}: frontmatter \`status\` must be one of ${[...allowedStatuses].join(', ')}`,
    );
  }
  if (metadata['last-verified'] && !isValidDate(metadata['last-verified'])) {
    issues.push(`${relativeFile}: frontmatter \`last-verified\` must use YYYY-MM-DD`);
  }
  return issues;
}

function normalizeLinkTarget(rawTarget) {
  let target = rawTarget.trim();
  if (target.startsWith('<') && target.endsWith('>')) {
    target = target.slice(1, -1);
  }
  const [firstToken] = target.split(/\s+/);
  return firstToken.split('#')[0].split('?')[0];
}

export function collectRelativeMarkdownLinks(text) {
  const links = [];
  const regex = /!?\[[^\]]*]\(([^)]+)\)/g;
  for (const match of text.matchAll(regex)) {
    const normalized = normalizeLinkTarget(match[1]);
    if (!normalized || normalized.startsWith('#')) {
      continue;
    }
    if (/^(https?:|mailto:|tel:)/.test(normalized)) {
      continue;
    }
    links.push(normalized);
  }
  return links;
}

function resolveLink(repoRootDir, relativeFile, target) {
  const sourceDir = path.dirname(path.join(repoRootDir, relativeFile));
  const absoluteTarget = path.resolve(sourceDir, target);
  if (!absoluteTarget.startsWith(repoRootDir)) {
    return null;
  }
  return absoluteTarget;
}

function validateLinkTargets(repoRootDir, relativeFile, text) {
  const issues = [];
  for (const linkTarget of collectRelativeMarkdownLinks(text)) {
    const resolved = resolveLink(repoRootDir, relativeFile, linkTarget);
    const resolvedRelative =
      resolved && resolved.startsWith(repoRootDir) ? relativePath(repoRootDir, resolved) : null;
    if (resolvedRelative && RETIRED_PATH_PATTERNS.some((pattern) => pattern.test(resolvedRelative))) {
      issues.push(`${relativeFile}: references retired path -> ${linkTarget}`);
    }
    if (!resolved || !fs.existsSync(resolved)) {
      issues.push(`${relativeFile}: broken relative link -> ${linkTarget}`);
    }
  }
  return issues;
}

function validateLongTermContent(relativeFile, text) {
  const issues = [];
  if (ABSOLUTE_PATH_PATTERN.test(text)) {
    issues.push(`${relativeFile}: contains local-machine absolute path`);
  }
  if (RETIRED_PATH_PATTERNS.some((pattern) => pattern.test(text))) {
    issues.push(`${relativeFile}: references retired path -> update to current docs layout`);
  }
  return issues;
}

function validateDocsPlacement(relativeEntries) {
  const issues = [];
  for (const entry of relativeEntries) {
    const firstSegment = entry.split('/').slice(0, 2).join('/');
    if (RETIRED_PATH_PATTERNS.some((pattern) => pattern.test(entry))) {
      issues.push(`${entry}: retired docs location`);
    }
    if (entry.startsWith('docs/')) {
      if (!TOP_LEVEL_ALLOWED.has(firstSegment) && !TOP_LEVEL_ALLOWED.has(entry)) {
        issues.push(`${entry}: unexpected top-level docs location`);
      }
    }
    if (!entry.endsWith('.md') && entry.startsWith('docs/') && !entry.startsWith('docs/internal/')) {
      if (!LONG_TERM_ASSET_ALLOWLIST.has(entry)) {
        issues.push(`${entry}: unexpected long-term non-Markdown file`);
      }
    }
  }
  return issues;
}

function collectDocsEntries(repoRootDir) {
  return walkFiles(path.join(repoRootDir, 'docs')).map((file) => relativePath(repoRootDir, file));
}

function collectDSStoreIssues(repoRootDir) {
  return walkFiles(repoRootDir, (file) => path.basename(file) === '.DS_Store').map(
    (file) => `${relativePath(repoRootDir, file)}: .DS_Store is forbidden`,
  );
}

export function validateDocsTree(repoRootDir) {
  const issues = [];
  issues.push(...collectDSStoreIssues(repoRootDir));
  const relativeEntries = collectDocsEntries(repoRootDir);
  issues.push(...validateDocsPlacement(relativeEntries));
  for (const relativeFile of relativeEntries.filter((file) => file.endsWith('.md'))) {
    const text = fs.readFileSync(path.join(repoRootDir, relativeFile), 'utf8');
    issues.push(...validateMetadata(relativeFile, parseFrontmatter(text)));
    if (!relativeFile.startsWith('docs/internal/')) {
      issues.push(...validateLongTermContent(relativeFile, text));
      issues.push(...validateLinkTargets(repoRootDir, relativeFile, text));
    }
  }
  return issues;
}

export function formatIssues(issues) {
  if (issues.length === 0) {
    return 'docs-hygiene check PASSED.';
  }
  return ['docs-hygiene check FAILED.', ...issues.map((issue) => `- ${issue}`)].join('\n');
}
