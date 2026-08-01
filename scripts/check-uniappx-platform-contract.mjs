#!/usr/bin/env node

import { readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')

function readJSON(relativePath) {
  return JSON.parse(readFileSync(join(repositoryRoot, relativePath), 'utf8'))
}

const clientsPackage = readJSON('clients/package.json')
const uniappxPackage = readJSON('clients/uniappx/package.json')
const manifest = readJSON('clients/uniappx/src/manifest.json')
const violations = []

if (clientsPackage.scripts?.['build:uni:h5'] === undefined) {
  violations.push('clients/package.json must keep the supported build:uni:h5 target')
}
if (uniappxPackage.scripts?.['build:h5'] === undefined) {
  violations.push('clients/uniappx/package.json must keep the supported build:h5 target')
}

for (const script of ['build:uni:mp']) {
  if (clientsPackage.scripts?.[script] !== undefined) {
    violations.push(`clients/package.json must not advertise unsupported script ${script}`)
  }
}
for (const script of ['dev:mp-weixin', 'build:mp-weixin']) {
  if (uniappxPackage.scripts?.[script] !== undefined) {
    violations.push(`clients/uniappx/package.json must not advertise unsupported script ${script}`)
  }
}
if (manifest['mp-weixin'] !== undefined) {
  violations.push('clients/uniappx/src/manifest.json must not declare unsupported mp-weixin')
}

if (violations.length > 0) {
  console.error('UniAppX supported-target contract failed:')
  for (const violation of violations) console.error(`- ${violation}`)
  process.exitCode = 1
} else {
  console.log('UniAppX supported-target contract passed (H5 supported; mp-weixin not advertised).')
}
