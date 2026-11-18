import { createHash } from 'node:crypto'
import { readFileSync, writeFileSync } from 'node:fs'
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

export function installPatch() {
  const lock = JSON.parse(readFileSync(new URL('../package-lock.json', import.meta.url)))
  const copies = Object.keys(lock.packages).filter(path => path.endsWith('node_modules/picomatch'))
  if (copies.length !== 1 || copies[0] !== 'node_modules/picomatch') {
    throw new Error('Review picomatch security backport: dependency layout changed')
  }
  const root = new URL('../node_modules/picomatch/', import.meta.url)
  const manifest = JSON.parse(readFileSync(new URL('package.json', root)))
  if (manifest.version !== patch.version) throw new Error('Review picomatch security backport for the new version')
  const target = new URL('lib/parse.js', root)
  const source = readFileSync(target, 'utf8')
  const patched = patchParser(source)
  if (patched !== source) writeFileSync(target, patched)
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  installPatch()
  console.log('Verified picomatch repeated-extglob backport (upstream PR #262)')
}
