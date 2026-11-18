import { createHash } from 'node:crypto'
import { existsSync, readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const patch = JSON.parse(readFileSync(new URL('../patches/picomatch-4.0.7.json', import.meta.url)))
const sha256 = source => createHash('sha256').update(source).digest('hex')

export function patchParser(source) {
  if (sha256(source) === patch.afterSha256) return source
  if (sha256(source) !== patch.beforeSha256) {
    throw new Error('Unexpected picomatch parser; review the security backport before updating dependencies')
  }
  for (const { before, after } of patch.edits) {
    if (source.split(before).length !== 2) throw new Error('Picomatch patch context is not unique')
    source = source.replace(before, () => after)
  }
  if (sha256(source) !== patch.afterSha256) throw new Error('Picomatch patch output hash mismatch')
  return source
}

// Patches every installed copy of picomatch at the version the backport
// targets, and leaves every other copy alone. Deciding whether a different
// version is acceptable (a fixed release, a new nested copy) is the audit
// gate's job (scripts/npm-audit-gate.mjs checks the lockfile against
// audit-accepted.json), so an upgrade or a dev-less install never breaks
// `npm ci` here. A copy at the targeted version whose parser does not hash to
// the known original or patched source still fails closed.
export function installPatch(treeDir = new URL('../', import.meta.url), log = console.log) {
  const lockUrl = new URL('package-lock.json', treeDir)
  const lock = existsSync(lockUrl) ? JSON.parse(readFileSync(lockUrl)) : {}
  const copies = Object.keys(lock.packages || {})
    .filter(path => path === 'node_modules/picomatch' || path.endsWith('/node_modules/picomatch'))
  const patched = []
  for (const path of copies) {
    const root = new URL(`${path}/`, treeDir)
    const manifestUrl = new URL('package.json', root)
    if (!existsSync(manifestUrl)) continue // not installed, e.g. npm ci --omit=dev
    const { version } = JSON.parse(readFileSync(manifestUrl))
    if (version !== patch.version) {
      log(`picomatch backport: ${path} is ${version}, not ${patch.version}; left unpatched for the audit gate to review`)
      continue
    }
    const target = new URL('lib/parse.js', root)
    const source = readFileSync(target, 'utf8')
    const result = patchParser(source)
    if (result !== source) writeFileSync(target, result)
    patched.push(path)
  }
  return patched
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const patched = installPatch()
  if (patched.length) {
    console.log(`Verified picomatch repeated-extglob backport (upstream PR #262): ${patched.join(', ')}`)
  }
}
