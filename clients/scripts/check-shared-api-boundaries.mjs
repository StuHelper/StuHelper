import { readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const clientsRoot = path.resolve(scriptDir, '..');

const scanRoots = [
  'web/src',
  'admin/apps/web-ele/src',
  'uniappx/src',
];

const ignoredDirs = new Set([
  '__tests__',
  '__mocks__',
  '.turbo',
  'coverage',
  'dist',
  'node_modules',
  'storybook-static',
]);

const checkedExtensions = new Set([
  '.js',
  '.jsx',
  '.mjs',
  '.ts',
  '.tsx',
  '.vue',
]);

const allowedFiles = new Map([
  [
    'web/src/api/client.ts',
    'web shared API transport adapter',
  ],
  [
    'admin/apps/web-ele/src/api/shared-client.ts',
    'admin shared API transport adapter',
  ],
  [
    'uniappx/src/api/shared-client.ts',
    'UniAppX shared API transport adapter',
  ],
]);

const boundaryPatterns = [
  {
    kind: 'fetch',
    regex: /(?:^|[^\w$.])((?:(?:window|globalThis|self)(?:\.|\?\.)fetch|fetch)(?:\?\.)?)\s*\(/g,
  },
  {
    kind: 'sendBeacon',
    regex: /(navigator(?:\.|\?\.)sendBeacon(?:\?\.)?)\s*\(/g,
  },
  {
    kind: 'EventSource',
    regex: /(new\s+(?:window\.)?EventSource)\s*\(/g,
  },
  {
    kind: 'window.open navigation',
    regex: /(?:^|[^\w$.])((?:window|globalThis)(?:\.|\?\.)open(?:\?\.)?)\s*\(/g,
  },
  {
    kind: 'window.location navigation',
    regex: /(?:^|[^\w$.])((?:window\.)?location\.(?:href|assign|replace|reload))\s*(?:=|\()/g,
  },
  {
    kind: 'native openURL',
    regex: /(?:^|[^\w$.])((?:\w+\.)?runtime\.openURL)\s*\(/g,
  },
  {
    kind: 'uni.request',
    regex: /(?:^|[^\w$.])(uni\.request)\s*(?:\(|as\b)/g,
  },
  {
    kind: 'admin raw request client',
    regex: /(baseRequestClient\.instance\.request)\s*(?:<|\()/g,
  },
];

const commentLookbackLines = 4;
const boundaryCommentPattern =
  /(?:shared API client|clients\/shared|OpenAPI|SSE|EventSource|sendBeacon|keepalive|OIDC|OAuth|SSO|IdP|Cookie|WebView|deep link|chunk|account center|admin console|browser|navigation|reload|redirect|Open Platform|业务 API|浏览器|导航|跳转|重载|刷新|认证|回调|跨站|跨端|上游|系统浏览器|页面|卸载期|授权)/i;

const findings = [];
const skippedAllowedFiles = [];

for (const root of scanRoots) {
  walk(path.join(clientsRoot, root));
}

if (findings.length > 0) {
  for (const finding of findings) {
    console.error(
      `${finding.file}:${finding.line}:${finding.column}: undocumented ${finding.kind} boundary (${finding.target})`,
    );
  }
  console.error(
    '[check-shared-api-boundaries] Direct browser/API boundaries must use clients/shared, have a nearby explanatory comment, or be added to the explicit adapter allowlist.',
  );
  process.exit(1);
}

console.log(
  `[check-shared-api-boundaries] OK: scanned ${scanRoots.join(', ')}; skipped ${skippedAllowedFiles.length} adapter file(s)`,
);

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

  const relativeFile = path.relative(clientsRoot, target).replaceAll(path.sep, '/');
  const allowReason = allowedFiles.get(relativeFile);
  if (allowReason) {
    skippedAllowedFiles.push({ file: relativeFile, reason: allowReason });
    return;
  }

  scanFile(relativeFile, readFileSync(target, 'utf8'));
}

function scanFile(relativeFile, source) {
  const lines = source.split(/\r?\n/);
  for (let lineIndex = 0; lineIndex < lines.length; lineIndex += 1) {
    const line = lines[lineIndex];
    if (isStandaloneComment(line)) {
      continue;
    }

    for (const pattern of boundaryPatterns) {
      pattern.regex.lastIndex = 0;
      for (const match of line.matchAll(pattern.regex)) {
        const target = match[1] ?? match[0];
        const targetOffset = match[0].indexOf(target);
        const column = (match.index ?? 0) + Math.max(targetOffset, 0) + 1;
        if (hasNearbyBoundaryComment(lines, lineIndex)) {
          continue;
        }
        findings.push({
          column,
          file: relativeFile,
          kind: pattern.kind,
          line: lineIndex + 1,
          target,
        });
      }
    }
  }
}

function hasNearbyBoundaryComment(lines, lineIndex) {
  for (
    let index = lineIndex;
    index >= 0 && index >= lineIndex - commentLookbackLines;
    index -= 1
  ) {
    const comment = extractComment(lines[index]);
    if (comment && boundaryCommentPattern.test(comment)) {
      return true;
    }
  }
  return false;
}

function extractComment(line) {
  const trimmed = line.trim();
  if (trimmed.startsWith('//') || trimmed.startsWith('/*') || trimmed.startsWith('*')) {
    return trimmed;
  }

  const inlineCommentIndex = line.indexOf('//');
  if (inlineCommentIndex >= 0) {
    return line.slice(inlineCommentIndex).trim();
  }

  return '';
}

function isStandaloneComment(line) {
  return Boolean(extractComment(line)) && line.trim().startsWith(extractComment(line));
}
