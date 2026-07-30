#!/usr/bin/env node

import { readFile, stat } from 'node:fs/promises'
import { dirname, join, posix, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const appRoot = join(repositoryRoot, 'clients', 'uniappx')
const sourceRoot = join(appRoot, 'src')
const outputRoot = join(appRoot, 'dist', 'build', 'h5')
const pagesPath = join(sourceRoot, 'pages.json')
const pages = JSON.parse(await readFile(pagesPath, 'utf8'))
const tabBarItems = pages?.tabBar?.list

if (!Array.isArray(tabBarItems) || tabBarItems.length === 0) {
  throw new Error('UniAppX pages.json must declare at least one tabBar item')
}

const assetPaths = new Set()
const failures = []

for (const [index, item] of tabBarItems.entries()) {
  for (const field of ['iconPath', 'selectedIconPath']) {
    const assetPath = item?.[field]
    if (typeof assetPath !== 'string' || assetPath.trim() === '') {
      failures.push(`tabBar.list[${index}].${field} must be a non-empty relative path`)
      continue
    }
    const normalized = posix.normalize(assetPath)
    if (
      normalized === '..' ||
      normalized.startsWith('../') ||
      normalized.startsWith('/') ||
      normalized.includes('\\')
    ) {
      failures.push(`tabBar.list[${index}].${field} escapes the app root: ${assetPath}`)
      continue
    }
    assetPaths.add(normalized)
  }
}

for (const assetPath of assetPaths) {
  await requireRegularFile(sourceRoot, assetPath, 'source', failures)
  await requireRegularFile(outputRoot, assetPath, 'H5 output', failures)
}

if (failures.length > 0) {
  throw new Error(`UniAppX tabBar asset contract failed:\n- ${failures.join('\n- ')}`)
}

console.log(`UniAppX tabBar asset contract passed (${assetPaths.size} files).`)

async function requireRegularFile(root, assetPath, label, errors) {
  const filePath = resolve(root, assetPath)
  const relativePath = relative(root, filePath)
  if (relativePath === '..' || relativePath.startsWith(`..${sep}`)) {
    errors.push(`${label} path escapes its root: ${assetPath}`)
    return
  }
  try {
    const fileStat = await stat(filePath)
    if (!fileStat.isFile()) {
      errors.push(`${label} is not a regular file: ${assetPath}`)
    }
  } catch (error) {
    if (error?.code === 'ENOENT') {
      errors.push(`${label} is missing: ${assetPath}`)
      return
    }
    throw error
  }
}
