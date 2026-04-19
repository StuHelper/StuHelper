#!/usr/bin/env node
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import {
  collectRelativeMarkdownLinks,
  formatIssues,
  parseFrontmatter,
  validateDocsTree,
} from './lib/docs-hygiene-lib.mjs';

const scriptPath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(scriptPath), '..');

function runCli() {
  const issues = validateDocsTree(repoRoot);
  const output = formatIssues(issues);
  const writer = issues.length === 0 ? console.log : console.error;
  writer(output);
  process.exit(issues.length === 0 ? 0 : 1);
}

export { collectRelativeMarkdownLinks, parseFrontmatter, validateDocsTree };

if (process.argv[1] && path.resolve(process.argv[1]) === scriptPath) {
  runCli();
}
