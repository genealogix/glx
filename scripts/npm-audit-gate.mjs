// npm audit gate with narrow, verified exceptions.
//
// Usage (from an npm tree, e.g. website/):
//   npm audit --json --audit-level=moderate > audit.json
//   node ../scripts/npm-audit-gate.mjs audit.json
//
// Exit codes: 0 no unaccepted finding at moderate or above, 1 unaccepted
// findings (or an accepted entry that no longer verifies), 3 the input is not
// an npm audit report (network failure; the workflow retries on this).
//
// An exception lives in <tree>/audit-accepted.json and names one advisory, the
// one package and exact version it is accepted for, why, and a command that
// proves the mitigation is in place. The gate runs that command first and
// fails if it fails, so an exception can never outlive its mitigation. A
// finding is accepted only when every advisory behind it, followed through
// npm's dependency chain, is an accepted one: a new advisory anywhere in the
// chain fails the run as before.

import { execSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const SEVERITY = { info: 0, low: 1, moderate: 2, high: 3, critical: 4 }

export const ghsaOf = url => (/GHSA(-[a-z0-9]{4}){3}/i.exec(url || '') || [''])[0].toUpperCase()

// Returns { failing: [names], accepted: [names], stale: [advisory ids] }.
export function evaluate(report, accepted, level = 'moderate') {
  const vulns = report.vulnerabilities || {}
  const ok = new Map()
  const isAcceptedAdvisory = (pkg, via) =>
    accepted.some(a => a.package === pkg && a.advisory.toUpperCase() === ghsaOf(via.url))

  const resolved = (name, seen = new Set()) => {
    if (ok.has(name)) return ok.get(name)
    // A dependency cycle cannot hide a real advisory: that advisory's own
    // package node is evaluated on its own and fails there.
    if (seen.has(name)) return true
    seen.add(name)
    const v = vulns[name]
    const result = !v || v.via.every(via =>
      typeof via === 'string' ? resolved(via, seen) : isAcceptedAdvisory(name, via))
    ok.set(name, result)
    return result
  }

  const failing = []
  const acceptedHits = []
  for (const [name, v] of Object.entries(vulns)) {
    if ((SEVERITY[v.severity] ?? 4) < SEVERITY[level]) continue
    ;(resolved(name) ? acceptedHits : failing).push(name)
  }
  const seenAdvisories = new Set(Object.values(vulns).flatMap(v =>
    v.via.filter(via => typeof via !== 'string').map(via => ghsaOf(via.url))))
  const stale = accepted.map(a => a.advisory.toUpperCase()).filter(id => !seenAdvisories.has(id))
  return { failing, accepted: acceptedHits, stale }
}

// Every installed copy of the package must be the accepted version; a bump
// means the exception has to be reviewed again.
export function checkVersions(treeDir, accepted) {
  if (accepted.length === 0) return []
  const lock = JSON.parse(readFileSync(join(treeDir, 'package-lock.json'), 'utf8'))
  const problems = []
  for (const a of accepted) {
    const copies = Object.entries(lock.packages || {})
      .filter(([path]) => path === `node_modules/${a.package}` || path.endsWith(`/node_modules/${a.package}`))
    if (copies.length === 0) problems.push(`${a.advisory}: ${a.package} is not installed; remove the entry`)
    for (const [path, meta] of copies) {
      if (meta.version !== a.version) {
        problems.push(`${a.advisory}: ${path} is ${meta.version}, accepted only for ${a.version}`)
      }
    }
  }
  return problems
}

function main(argv) {
  const treeDir = process.cwd()
  let report
  try {
    report = JSON.parse(readFileSync(argv[0], 'utf8'))
  } catch {
    return 3
  }
  if (!report || typeof report.vulnerabilities !== 'object') return 3

  const file = join(treeDir, 'audit-accepted.json')
  const accepted = existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')).accepted || [] : []

  for (const a of accepted) {
    for (const key of ['advisory', 'package', 'version', 'reason', 'verify']) {
      if (!a[key]) {
        console.log(`::error::audit-accepted.json: entry ${a.advisory || '?'} has no "${key}"`)
        return 1
      }
    }
  }
  const versionProblems = checkVersions(treeDir, accepted)
  for (const p of versionProblems) console.log(`::error::${p}`)
  if (versionProblems.length) return 1
  for (const a of accepted) {
    try {
      execSync(a.verify, { cwd: treeDir, stdio: 'inherit' })
    } catch {
      console.log(`::error::${a.advisory} is accepted only while "${a.verify}" passes, and it failed`)
      return 1
    }
  }

  const { failing, accepted: hits, stale } = evaluate(report, accepted)
  for (const id of stale) {
    console.log(`::warning::${id} is in audit-accepted.json but npm audit no longer reports it; remove the entry`)
  }
  if (hits.length) {
    const ids = accepted.map(a => a.advisory).join(', ')
    console.log(`::warning::Accepted with a verified mitigation (${ids}): ${hits.join(', ')}`)
  }
  if (failing.length) {
    console.log(`::error::npm audit reported vulnerabilities at or above the moderate level: ${failing.join(', ')}`)
    return 1
  }
  return 0
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.exit(main(process.argv.slice(2)))
}
