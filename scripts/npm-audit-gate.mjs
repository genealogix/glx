// npm audit gate with narrow, verified exceptions.
//
// Usage (from an npm tree, e.g. website/):
//   npm audit --json --fetch-timeout=60000 > audit.json
//   node ../scripts/npm-audit-gate.mjs audit.json --audit-level=moderate
//
// The threshold is the gate's --audit-level (default moderate). npm's own
// --audit-level only sets npm's exit code, which the workflow ignores, so it is
// not passed to npm at all.
//
// Exit codes: 0 no unaccepted finding at the threshold or above, 1 unaccepted
// findings (or an accepted entry that is malformed, expired, for another
// version, or no longer verifies), 3 the input is not an npm audit report
// (network failure; the workflow retries on this).
//
// An exception lives in <tree>/audit-accepted.json and names one advisory, the
// one package and exact version it is accepted for, why, a command that proves
// the mitigation is in place, and an ignoreUntil date (YYYY-MM-DD) after which
// it stops applying. The gate runs that command first and fails if it fails,
// so an exception can never outlive its mitigation. This is the npm-tree
// counterpart of osv-scanner.toml; see SECURITY-POSTURE.md, "Vulnerability
// suppression".
//
// Severity is judged per advisory: a node fails when an advisory that is not
// accepted, reached anywhere through npm's dependency chain, is itself at the
// threshold or above. A node whose reported severity comes only from accepted
// advisories, plus unaccepted ones below the threshold, passes, which is what
// plain npm audit would do once the accepted advisory is gone.

import { execSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const SEVERITY = { info: 0, low: 1, moderate: 2, high: 3, critical: 4 }
const REQUIRED = ['advisory', 'package', 'version', 'reason', 'verify', 'ignoreUntil']

export const ghsaOf = url => (/GHSA(-[a-z0-9]{4}){3}/i.exec(url || '') || [''])[0].toUpperCase()

const rank = severity => SEVERITY[severity] ?? SEVERITY.critical

// Returns { failing: [names], accepted: [names], stale: [advisory ids],
// details: { name: [unaccepted advisories at or above the threshold] } }.
export function evaluate(report, accepted, level = 'moderate') {
  const vulns = report.vulnerabilities || {}
  const threshold = rank(level)
  const isAcceptedAdvisory = (pkg, via) =>
    accepted.some(a => a.package === pkg && a.advisory.toUpperCase() === ghsaOf(via.url))

  // The unaccepted advisories each node reaches through its chain, found by
  // iterating to a fixed point, so a dependency cycle needs no special case
  // and no node is ever decided on a provisional answer.
  const reach = new Map()
  for (const [name, v] of Object.entries(vulns)) {
    const own = new Map()
    for (const via of v.via || []) {
      if (typeof via === 'string' || isAcceptedAdvisory(name, via)) continue
      own.set(via.url || via.title || JSON.stringify(via), via)
    }
    reach.set(name, own)
  }
  for (let changed = true; changed;) {
    changed = false
    for (const [name, v] of Object.entries(vulns)) {
      const mine = reach.get(name)
      for (const via of v.via || []) {
        if (typeof via !== 'string') continue
        for (const [key, adv] of reach.get(via) || []) {
          if (!mine.has(key)) {
            mine.set(key, adv)
            changed = true
          }
        }
      }
    }
  }

  const failing = []
  const acceptedHits = []
  const details = {}
  for (const [name, v] of Object.entries(vulns)) {
    if (rank(v.severity) < threshold) continue
    const blocking = [...reach.get(name).values()].filter(adv => rank(adv.severity) >= threshold)
    if (blocking.length) {
      failing.push(name)
      details[name] = blocking
    } else {
      acceptedHits.push(name)
    }
  }
  const seenAdvisories = new Set(Object.values(vulns).flatMap(v =>
    (v.via || []).filter(via => typeof via !== 'string').map(via => ghsaOf(via.url))))
  const stale = accepted.map(a => a.advisory.toUpperCase()).filter(id => !seenAdvisories.has(id))
  return { failing, accepted: acceptedHits, stale, details }
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

// Structural problems with the accepted entries, including expiry.
export function checkEntries(accepted, today = new Date().toISOString().slice(0, 10)) {
  if (!Array.isArray(accepted)) return ['audit-accepted.json: "accepted" must be an array']
  const problems = []
  for (const a of accepted) {
    if (!a || typeof a !== 'object') {
      problems.push('audit-accepted.json: every entry must be an object')
      continue
    }
    const missing = REQUIRED.filter(key => typeof a[key] !== 'string' || !a[key])
    if (missing.length) {
      problems.push(`audit-accepted.json: entry ${a.advisory || '?'} has no ${missing.map(k => `"${k}"`).join(', ')}`)
      continue
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(a.ignoreUntil)) {
      problems.push(`audit-accepted.json: entry ${a.advisory} has ignoreUntil "${a.ignoreUntil}", expected YYYY-MM-DD`)
    } else if (a.ignoreUntil < today) {
      problems.push(`${a.advisory}: the exception expired on ${a.ignoreUntil}; re-verify it and extend ignoreUntil, or remove it`)
    }
  }
  return problems
}

const isReport = report =>
  report !== null && typeof report === 'object' && !Array.isArray(report) &&
  report.vulnerabilities !== null && typeof report.vulnerabilities === 'object' &&
  !Array.isArray(report.vulnerabilities)

function describe(name, v, advisories) {
  const lines = [`${name} (${v.severity}${v.range ? `, ${v.range}` : ''})`]
  for (const adv of advisories) {
    const id = ghsaOf(adv.url) || adv.url || 'advisory'
    lines.push(`  ${adv.severity} ${id} in ${adv.name || adv.dependency || name}: ${adv.title || ''}`.trimEnd())
    if (adv.url) lines.push(`    ${adv.url}`)
  }
  const fix = v.fixAvailable
  if (fix === false) lines.push('  fix: none available')
  else if (fix === true) lines.push('  fix: npm audit fix')
  else if (fix && fix.name) lines.push(`  fix: ${fix.name}@${fix.version}${fix.isSemVerMajor ? ' (semver-major)' : ''}`)
  return lines.join('\n')
}

export function main(argv, { treeDir = process.cwd(), log = console.log, run = execSync, today } = {}) {
  const levelArg = argv.find(a => a.startsWith('--audit-level='))
  const level = levelArg ? levelArg.slice('--audit-level='.length) : 'moderate'
  if (!Object.hasOwn(SEVERITY, level)) {
    log(`::error::unknown --audit-level "${level}"; expected one of ${Object.keys(SEVERITY).join(', ')}`)
    return 1
  }
  const input = argv.find(a => !a.startsWith('--'))
  let report
  try {
    report = JSON.parse(readFileSync(input, 'utf8'))
  } catch {
    return 3
  }
  if (!isReport(report)) return 3

  const file = join(treeDir, 'audit-accepted.json')
  let accepted = []
  if (existsSync(file)) {
    try {
      accepted = JSON.parse(readFileSync(file, 'utf8')).accepted ?? []
    } catch (err) {
      log(`::error::audit-accepted.json is not valid JSON: ${err.message}`)
      return 1
    }
  }
  const entryProblems = checkEntries(accepted, today)
  for (const p of entryProblems) log(`::error::${p}`)
  if (entryProblems.length) return 1
  const versionProblems = checkVersions(treeDir, accepted)
  for (const p of versionProblems) log(`::error::${p}`)
  if (versionProblems.length) return 1
  for (const a of accepted) {
    try {
      run(a.verify, { cwd: treeDir, stdio: 'inherit' })
    } catch {
      log(`::error::${a.advisory} is accepted only while "${a.verify}" passes, and it failed`)
      return 1
    }
  }

  const { failing, accepted: hits, stale, details } = evaluate(report, accepted, level)
  const total = Object.keys(report.vulnerabilities).length
  log(`npm audit: ${total} vulnerable package(s) reported; threshold ${level}`)
  for (const id of stale) {
    log(`::warning::${id} is in audit-accepted.json but npm audit no longer reports it; remove the entry`)
  }
  if (hits.length) {
    const ids = accepted.map(a => a.advisory).join(', ')
    log(`::warning::Accepted with a verified mitigation (${ids}): ${hits.join(', ')}`)
  }
  if (failing.length) {
    for (const name of failing) log(describe(name, report.vulnerabilities[name], details[name]))
    log(`::error::npm audit reported vulnerabilities at or above the ${level} level: ${failing.join(', ')}`)
    return 1
  }
  return 0
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.exit(main(process.argv.slice(2)))
}
