import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { createRequire } from 'node:module'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDirectory = dirname(fileURLToPath(import.meta.url))
const clientsDirectory = resolve(scriptDirectory, '..')
const virtualStores = [
  join(clientsDirectory, 'node_modules', '.pnpm'),
  join(clientsDirectory, 'admin', 'node_modules', '.pnpm'),
]

let checkedPackages = 0

function isSupportedBraceExpansionVersion(version) {
  const match = /^(\d+)\.(\d+)\.(\d+)(?:-|$)/.exec(version)
  if (!match) return false

  const [, major, minor, patch] = match.map(Number)
  return major === 5 && (minor > 0 || patch >= 8)
}

for (const virtualStore of virtualStores) {
  if (!existsSync(virtualStore)) {
    throw new Error(`missing pnpm virtual store: ${virtualStore}`)
  }

  const workspaceDirectory = resolve(virtualStore, '..', '..')
  const lockfile = readFileSync(join(workspaceDirectory, 'pnpm-lock.yaml'), 'utf8')
  const minimatchEntries = readdirSync(virtualStore)
    .filter(entry => entry.startsWith('minimatch@'))
    .filter(entry => lockfile.includes(`\n  ${entry}:\n`))
    .sort()

  for (const entry of minimatchEntries) {
    const packageDirectory = join(virtualStore, entry, 'node_modules', 'minimatch')
    const requireFromPackage = createRequire(join(packageDirectory, 'package.json'))
    const minimatchModule = requireFromPackage(packageDirectory)
    const minimatch =
      typeof minimatchModule === 'function' ? minimatchModule : minimatchModule.minimatch

    if (
      typeof minimatch !== 'function' ||
      !minimatch('src/app.ts', 'src/{app,test}.ts')
    ) {
      throw new Error(`${entry} cannot perform brace expansion`)
    }

    const bracePackagePath = requireFromPackage.resolve('brace-expansion/package.json')
    const bracePackage = requireFromPackage(bracePackagePath)
    const braceModule = requireFromPackage('brace-expansion')
    const hasCompatibleExports =
      typeof braceModule?.expand === 'function' &&
      // 5.0.8 needs the repository patch to stay callable for minimatch < 10.
      (bracePackage.version !== '5.0.8' || typeof braceModule === 'function')

    if (
      !isSupportedBraceExpansionVersion(bracePackage.version) ||
      !hasCompatibleExports
    ) {
      throw new Error(
        `${entry} is not using a compatible brace-expansion >= 5.0.8 export`,
      )
    }

    checkedPackages += 1
  }
}

if (checkedPackages === 0) {
  throw new Error('no minimatch packages were checked')
}

console.log(
  `[check-brace-expansion-compat] OK: ${checkedPackages} minimatch installations use compatible brace-expansion >= 5.0.8`,
)
