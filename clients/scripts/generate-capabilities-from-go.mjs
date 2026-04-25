#!/usr/bin/env node

import { readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const goPath = path.resolve(__dirname, '../../server/internal/pkg/capability/capability.go')
const outPath = path.resolve(__dirname, '../shared/src/constants/capabilities.gen.ts')

const source = readFileSync(goPath, 'utf8')

function extractBlock(startPattern) {
  const start = source.indexOf(startPattern)
  if (start === -1) {
    throw new Error(`Unable to find block: ${startPattern}`)
  }
  const braceStart = source.indexOf('{', start)
  if (braceStart === -1) {
    throw new Error(`Unable to find opening brace for: ${startPattern}`)
  }
  let depth = 0
  for (let i = braceStart; i < source.length; i += 1) {
    const char = source[i]
    if (char === '{') depth += 1
    if (char === '}') {
      depth -= 1
      if (depth === 0) {
        return source.slice(braceStart + 1, i)
      }
    }
  }
  throw new Error(`Unable to find closing brace for: ${startPattern}`)
}

function toUpperSnakeCase(name) {
  return name
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1_$2')
    .toUpperCase()
}

const constBlockMatch = source.match(/const\s*\(([\s\S]*?)\n\)/)
if (!constBlockMatch) {
  throw new Error('Unable to parse Go const block')
}

const constEntries = [...constBlockMatch[1].matchAll(/^\s*([A-Za-z0-9_]+)\s*=\s*"([^"]+)"\s*$/gm)].map(
  ([, goName, value]) => ({
    goName,
    tsName: toUpperSnakeCase(goName),
    value,
  }),
)

if (constEntries.length === 0) {
  throw new Error('No capability constants found in Go source')
}

const adminEntryBlock = extractBlock('var AdminEntryCapabilities = []string{')
const adminEntries = [...adminEntryBlock.matchAll(/([A-Za-z0-9_]+)/g)].map(([, name]) => name).filter(Boolean)

const roleCapabilitiesBlock = extractBlock('var roleCapabilities = map[string][]string{')
const roleEntries = [...roleCapabilitiesBlock.matchAll(/"([^"]+)"\s*:\s*\{([\s\S]*?)\n\s*\},?/g)].map(
  ([, role, body]) => ({
    role,
    capabilities: [...body.matchAll(/([A-Za-z0-9_]+)/g)].map(([, capability]) => capability),
  }),
)

const header = `/**
 * AUTO-GENERATED FILE. DO NOT EDIT.
 *
 * Source of truth:
 *   /Users/zxy/Code/StuHelper/server/internal/pkg/capability/capability.go
 *
 * Regenerate with:
 *   cd /Users/zxy/Code/StuHelper/clients && pnpm run generate:capabilities
 */
`

const body = [
  ...constEntries.map(({ tsName, value }) => `export const ${tsName} = '${value}' as const`),
  '',
  'export const ALL_CAPABILITIES = [',
  ...constEntries.map(({ tsName }) => `  ${tsName},`),
  '] as const',
  '',
  'export const ADMIN_ENTRY_CAPABILITIES = [',
  ...adminEntries.map((goName) => `  ${toUpperSnakeCase(goName)},`),
  '] as const',
  '',
  `export const ROLE_NAMES = [${roleEntries.map(({ role }) => JSON.stringify(role)).join(', ')}] as const`,
  '',
  'export const ROLE_CAPABILITIES = {',
  ...roleEntries.map(
    ({ role, capabilities }) => `  ${JSON.stringify(role)}: [${capabilities.map((name) => toUpperSnakeCase(name)).join(', ')}],`,
  ),
  '} as const',
  '',
].join('\n')

writeFileSync(outPath, `${header}\n${body}`)
