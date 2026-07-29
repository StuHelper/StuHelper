#!/usr/bin/env node

import { readdirSync, readFileSync, statSync } from 'node:fs';
import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const defaultRoots = [
  'clients/web/src',
  'clients/admin/apps/web-ele/src',
  'clients/uniappx/src',
  'bots/koishi/plugins/stuhelper-core/client',
];
const requestedRoots = process.argv.slice(2);
const roots = (requestedRoots.length > 0 ? requestedRoots : defaultRoots).map(
  (entry) => resolve(repositoryRoot, entry),
);
const uniAppXSourceRoot = resolve(repositoryRoot, 'clients/uniappx/src');
const nativeClickTargets = new Set([
  'a',
  'address',
  'article',
  'aside',
  'code',
  'dd',
  'div',
  'dt',
  'em',
  'figcaption',
  'figure',
  'footer',
  'header',
  'img',
  'label',
  'li',
  'main',
  'nav',
  'ol',
  'p',
  'pre',
  'section',
  'small',
  'span',
  'strong',
  'table',
  'tbody',
  'td',
  'tfoot',
  'th',
  'thead',
  'time',
  'tr',
  'ul',
]);
const failures = [];

for (const root of roots) {
  if (!statSync(root).isDirectory()) {
    failures.push(`${relative(repositoryRoot, root)}: expected a directory`);
    continue;
  }
  visitDirectory(root);
}

if (failures.length > 0) {
  console.error('Vue UI contract check failed:');
  for (const failure of failures) {
    console.error(`- ${failure}`);
  }
  process.exitCode = 1;
} else {
  console.log(`Vue UI contracts passed for ${roots.length} source root(s).`);
}

function visitDirectory(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const target = resolve(directory, entry.name);
    if (entry.isDirectory()) {
      visitDirectory(target);
      continue;
    }
    if (entry.isFile() && entry.name.endsWith('.vue')) {
      checkVueFile(target);
    }
  }
}

function checkVueFile(filename) {
  const source = readFileSync(filename, 'utf8');
  const template = readTemplate(source);
  if (!template) return;
  const isUniAppXSource =
    filename === uniAppXSourceRoot ||
    filename.startsWith(`${uniAppXSourceRoot}${sep}`);
  const labelRanges = pairedElementRanges(template.content, 'label');
  const labelledControlIDs = new Set(
    openingTags(template.content, 'label')
      .map((tag) => staticAttributeValue(tag.text, 'for'))
      .filter(Boolean),
  );

  for (const tag of openingTags(template.content, 'button')) {
    if (isUniAppXSource) {
      if (staticAttributeValue(tag.text, 'role') !== 'button') {
        report(
          filename,
          source,
          template.offset + tag.offset,
          'UniAppX button 编译为跨端组件，必须显式声明 role="button"',
        );
      }
      if (!hasAttribute(tag.text, 'tabindex')) {
        report(
          filename,
          source,
          template.offset + tag.offset,
          'UniAppX button 必须显式声明 tabindex',
        );
      }
      if (
        !/(?:^|\s)(?:@|v-on:)(?:keydown|keypress|keyup)(?:[.\s=])/u.test(
          tag.text,
        )
      ) {
        report(
          filename,
          source,
          template.offset + tag.offset,
          'UniAppX button 必须提供键盘激活处理',
        );
      }
      continue;
    }
    if (!hasAttribute(tag.text, 'type')) {
      report(filename, source, template.offset + tag.offset, 'button 必须显式声明 type');
    }
  }

  for (const tag of openingTags(template.content, 'img')) {
    if (!hasAttribute(tag.text, 'alt')) {
      report(filename, source, template.offset + tag.offset, 'img 必须显式声明 alt');
    }
  }

  for (const elementName of ['input', 'select', 'textarea']) {
    for (const tag of openingTags(template.content, elementName)) {
      if (
        elementName === 'input' &&
        staticAttributeValue(tag.text, 'type') === 'hidden'
      ) {
        continue;
      }
      const controlID = staticAttributeValue(tag.text, 'id');
      const nestedInLabel = labelRanges.some(
        (range) => range.start < tag.offset && tag.offset < range.end,
      );
      const hasAccessibleName =
        hasAttribute(tag.text, 'aria-label') ||
        hasAttribute(tag.text, 'aria-labelledby') ||
        /(?:^|\s)v-bind\s*=/u.test(tag.text) ||
        nestedInLabel ||
        (controlID && labelledControlIDs.has(controlID));
      if (!hasAccessibleName) {
        report(
          filename,
          source,
          template.offset + tag.offset,
          `${elementName} 必须具有可访问名称`,
        );
      }
    }
  }

  for (const tag of openingTags(template.content)) {
    const elementName = tagName(tag.text);
    const isNativeInteractive =
      elementName === 'button' ||
      elementName === 'input' ||
      elementName === 'select' ||
      elementName === 'summary' ||
      elementName === 'textarea' ||
      (elementName === 'a' && hasAttribute(tag.text, 'href'));
    const isPresentation =
      staticAttributeValue(tag.text, 'role') === 'presentation' ||
      staticAttributeValue(tag.text, 'role') === 'none' ||
      staticAttributeValue(tag.text, 'aria-hidden') === 'true';
    const hasClickHandler = /(?:^|\s)(?:@|v-on:)click(?:[.\s=])/u.test(
      tag.text,
    );
    const hasKeyboardHandler =
      /(?:^|\s)(?:@|v-on:)(?:keydown|keypress|keyup)(?:[.\s=])/u.test(
        tag.text,
      );
    if (
      hasClickHandler &&
      nativeClickTargets.has(elementName) &&
      !isNativeInteractive &&
      !isPresentation &&
      !hasKeyboardHandler
    ) {
      report(
        filename,
        source,
        template.offset + tag.offset,
        `${elementName} 的点击行为必须提供键盘等价操作`,
      );
    }
    if (/(?:^|\s)tabindex\s*=\s*["']?\+?[1-9]\d*/u.test(tag.text)) {
      report(
        filename,
        source,
        template.offset + tag.offset,
        '禁止使用正数 tabindex 改写自然焦点顺序',
      );
    }
  }
}

function readTemplate(source) {
  const openingStart = source.indexOf('<template');
  if (openingStart < 0) return null;
  const contentStart = findTagEnd(source, openingStart) + 1;
  if (contentStart <= 0) return null;
  const scriptStart = source.indexOf('<script', contentStart);
  const searchEnd = scriptStart >= 0 ? scriptStart : source.length;
  const contentEnd = source.lastIndexOf('</template>', searchEnd);
  if (contentEnd < contentStart) return null;
  return {
    content: source.slice(contentStart, contentEnd),
    offset: contentStart,
  };
}

function openingTags(source, expectedTag) {
  const tags = [];
  for (let offset = 0; offset < source.length; offset += 1) {
    if (source[offset] !== '<') continue;
    if (
      source.startsWith('<!--', offset) ||
      source.startsWith('</', offset) ||
      source.startsWith('<!', offset) ||
      source.startsWith('<?', offset)
    ) {
      continue;
    }

    const match = source.slice(offset + 1).match(/^[A-Za-z][\w:-]*/u);
    if (!match) continue;
    const tagName = match[0];
    if (expectedTag && tagName !== expectedTag) continue;
    const end = findTagEnd(source, offset);
    if (end < 0) break;
    tags.push({
      offset,
      text: source.slice(offset, end + 1),
    });
    offset = end;
  }
  return tags;
}

function pairedElementRanges(source, expectedTag) {
  const ranges = [];
  const stack = [];
  for (let offset = 0; offset < source.length; offset += 1) {
    const openingToken = `<${expectedTag}`;
    const closingToken = `</${expectedTag}`;
    if (source.startsWith(openingToken, offset)) {
      const boundary = source[offset + openingToken.length];
      if (boundary && !/[\s/>]/u.test(boundary)) continue;
      const end = findTagEnd(source, offset);
      if (end < 0) break;
      if (source[end - 1] !== '/') stack.push(offset);
      offset = end;
      continue;
    }
    if (!source.startsWith(closingToken, offset)) continue;
    const boundary = source[offset + closingToken.length];
    if (boundary && !/[\s>]/u.test(boundary)) continue;
    const start = stack.pop();
    const end = findTagEnd(source, offset);
    if (start !== undefined && end >= 0) ranges.push({ end, start });
    if (end >= 0) offset = end;
  }
  return ranges;
}

function findTagEnd(source, start) {
  let quote = '';
  for (let index = start; index < source.length; index += 1) {
    const character = source[index];
    if (quote) {
      if (character === quote && source[index - 1] !== '\\') quote = '';
      continue;
    }
    if (character === '"' || character === "'") {
      quote = character;
      continue;
    }
    if (character === '>') return index;
  }
  return -1;
}

function hasAttribute(tag, name) {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/gu, '\\$&');
  return new RegExp(
    `(?:^|\\s)(?::|v-bind:)?${escapedName}\\s*=`,
    'u',
  ).test(tag);
}

function staticAttributeValue(tag, name) {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/gu, '\\$&');
  const match = tag.match(
    new RegExp(`(?:^|\\s)${escapedName}\\s*=\\s*([\"'])(.*?)\\1`, 'u'),
  );
  return match?.[2] ?? '';
}

function tagName(tag) {
  return tag.slice(1).match(/^[A-Za-z][\w:-]*/u)?.[0] ?? 'element';
}

function report(filename, source, offset, message) {
  const line = source.slice(0, offset).split('\n').length;
  failures.push(`${relative(repositoryRoot, filename)}:${line}: ${message}`);
}
