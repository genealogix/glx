import { test } from 'node:test'
import assert from 'node:assert/strict'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { checkEntries, checkVersions, evaluate, ghsaOf, main } from './npm-audit-gate.mjs'

const advisory = (id, severity = 'high') => ({ source: 1, url: `https://github.com/advisories/${id}`, severity })
const ENTRY = {
  advisory: 'GHSA-fc5q-39xc-vjpf',
  package: 'picomatch',
  version: '4.0.7',
  reason: 'test',
  verify: 'true',
  ignoreUntil: '2099-01-01',
}
const ACCEPTED = [ENTRY]

// The website tree as npm audit reported it on 2026-11-18.
const picomatchChain = () => ({
  vulnerabilities: {
    picomatch: { severity: 'high', via: [advisory('GHSA-fc5q-39xc-vjpf')] },
    fdir: { severity: 'high', via: ['picomatch'] },
    tinyglobby: { severity: 'high', via: ['fdir', 'picomatch'] },
    vite: { severity: 'high', via: ['picomatch', 'tinyglobby'] },
    vitepress: { severity: 'high', via: ['vite'] },
  },
})

test('ghsaOf extracts the advisory id from its URL', () => {
  assert.equal(ghsaOf('https://github.com/advisories/GHSA-fc5q-39xc-vjpf'), 'GHSA-FC5Q-39XC-VJPF')
  assert.equal(ghsaOf(undefined), '')
})

test('an accepted advisory clears its whole dependency chain', () => {
  const r = evaluate(picomatchChain(), ACCEPTED)
  assert.deepEqual(r.failing, [])
  assert.equal(r.accepted.length, 5)
  assert.deepEqual(r.stale, [])
})

test('with no exception the chain fails as before', () => {
  assert.equal(evaluate(picomatchChain(), []).failing.length, 5)
})

test('a second advisory in the chain still fails', () => {
  const report = picomatchChain()
  report.vulnerabilities.vite.via.push(advisory('GHSA-aaaa-bbbb-cccc', 'moderate'))
  const r = evaluate(report, ACCEPTED)
  assert.deepEqual(r.failing.sort(), ['vite', 'vitepress'])
  assert.equal(ghsaOf(r.details.vitepress[0].url), 'GHSA-AAAA-BBBB-CCCC')
})

test('an unaccepted advisory below the threshold does not fail its chain', () => {
  // vite is reported high only because of the accepted picomatch advisory;
  // its own low advisory would pass plain npm audit at moderate.
  const report = picomatchChain()
  report.vulnerabilities.vite.via.push(advisory('GHSA-aaaa-bbbb-cccc', 'low'))
  const r = evaluate(report, ACCEPTED)
  assert.deepEqual(r.failing, [])
  assert.equal(r.accepted.length, 5)
  assert.equal(evaluate(report, ACCEPTED, 'low').failing.sort().join(), 'vite,vitepress')
})

test('an exception applies only to the package it names', () => {
  const report = { vulnerabilities: { other: { severity: 'high', via: [advisory('GHSA-fc5q-39xc-vjpf')] } } }
  assert.deepEqual(evaluate(report, ACCEPTED).failing, ['other'])
})

test('findings below moderate are ignored', () => {
  const report = { vulnerabilities: { x: { severity: 'low', via: [advisory('GHSA-aaaa-bbbb-cccc', 'low')] } } }
  assert.deepEqual(evaluate(report, []).failing, [])
})

test('an exception npm no longer reports is flagged stale', () => {
  assert.deepEqual(evaluate({ vulnerabilities: {} }, ACCEPTED).stale, ['GHSA-FC5Q-39XC-VJPF'])
})

test('a dependency cycle terminates', () => {
  const report = { vulnerabilities: {
    a: { severity: 'high', via: ['b'] },
    b: { severity: 'high', via: ['a', advisory('GHSA-aaaa-bbbb-cccc')] },
  } }
  assert.deepEqual(evaluate(report, []).failing.sort(), ['a', 'b'])
})

test('a cycle never reports a node as accepted when its chain fails, whatever the order', () => {
  for (const order of [['a', 'b'], ['b', 'a']]) {
    const nodes = {
      a: { severity: 'high', via: ['b'] },
      b: { severity: 'high', via: ['a', advisory('GHSA-aaaa-bbbb-cccc')] },
    }
    const report = { vulnerabilities: Object.fromEntries(order.map(n => [n, nodes[n]])) }
    const r = evaluate(report, ACCEPTED)
    assert.deepEqual(r.accepted, [], order.join())
    assert.deepEqual(r.failing.sort(), ['a', 'b'], order.join())
  }
})

test('checkEntries requires every field and a future ignoreUntil', () => {
  assert.deepEqual(checkEntries(ACCEPTED, '2026-11-18'), [])
  assert.match(checkEntries([{ ...ENTRY, verify: '' }], '2026-11-18')[0], /"verify"/)
  assert.match(checkEntries([{ ...ENTRY, ignoreUntil: undefined }], '2026-11-18')[0], /"ignoreUntil"/)
  assert.match(checkEntries([{ ...ENTRY, ignoreUntil: 'soon' }], '2026-11-18')[0], /YYYY-MM-DD/)
  assert.match(checkEntries([{ ...ENTRY, ignoreUntil: '2026-11-17' }], '2026-11-18')[0], /expired/)
  assert.deepEqual(checkEntries([{ ...ENTRY, ignoreUntil: '2026-11-18' }], '2026-11-18'), [])
  assert.match(checkEntries([null])[0], /object/)
  assert.match(checkEntries({})[0], /array/)
})

const tree = t => {
  const dir = mkdtempSync(join(tmpdir(), 'npm-audit-gate-'))
  t.after(() => rmSync(dir, { recursive: true, force: true }))
  const write = (file, value) => writeFileSync(join(dir, file), typeof value === 'string' ? value : JSON.stringify(value))
  return { dir, write }
}
const lockWith = packages => ({ packages: { '': {}, ...packages } })

test('checkVersions matches root and nested copies against the accepted version', t => {
  const { dir, write } = tree(t)
  write('package-lock.json', lockWith({ 'node_modules/picomatch': { version: '4.0.7' } }))
  assert.deepEqual(checkVersions(dir, ACCEPTED), [])

  write('package-lock.json', lockWith({
    'node_modules/picomatch': { version: '4.0.7' },
    'node_modules/vite/node_modules/picomatch': { version: '4.0.8' },
    'node_modules/not-picomatch': { version: '1.0.0' },
  }))
  assert.deepEqual(checkVersions(dir, ACCEPTED), [
    'GHSA-fc5q-39xc-vjpf: node_modules/vite/node_modules/picomatch is 4.0.8, accepted only for 4.0.7',
  ])

  write('package-lock.json', lockWith({ 'node_modules/not-picomatch': { version: '1.0.0' } }))
  assert.match(checkVersions(dir, ACCEPTED)[0], /not installed/)
})

const runMain = (dir, argv, opts = {}) => {
  const logs = []
  const calls = []
  const status = main(argv, {
    treeDir: dir, log: l => logs.push(l), today: '2026-11-18',
    run: cmd => { calls.push(cmd); if (opts.verifyFails) throw new Error('failed') },
    ...opts.main,
  })
  return { status, logs: logs.join('\n'), calls }
}

test('main returns 3 for input that is not an audit report', t => {
  const { dir, write } = tree(t)
  for (const body of ['', 'not json', 'null', '[]', '{"error":{"code":"ETIMEDOUT"}}',
    '{"vulnerabilities":null}', '{"vulnerabilities":[]}']) {
    write('audit.json', body)
    assert.equal(runMain(dir, [join(dir, 'audit.json')]).status, 3, body)
  }
  assert.equal(runMain(dir, [join(dir, 'missing.json')]).status, 3)
})

test('main passes an accepted chain and runs its verify command', t => {
  const { dir, write } = tree(t)
  write('audit.json', picomatchChain())
  write('audit-accepted.json', { accepted: ACCEPTED })
  write('package-lock.json', lockWith({ 'node_modules/picomatch': { version: '4.0.7' } }))
  const r = runMain(dir, [join(dir, 'audit.json')])
  assert.equal(r.status, 0)
  assert.deepEqual(r.calls, ['true'])
  assert.match(r.logs, /Accepted with a verified mitigation/)
})

test('main fails when the verify command fails', t => {
  const { dir, write } = tree(t)
  write('audit.json', picomatchChain())
  write('audit-accepted.json', { accepted: ACCEPTED })
  write('package-lock.json', lockWith({ 'node_modules/picomatch': { version: '4.0.7' } }))
  const r = runMain(dir, [join(dir, 'audit.json')], { verifyFails: true })
  assert.equal(r.status, 1)
  assert.match(r.logs, /accepted only while "true" passes, and it failed/)
})

test('main fails on a malformed, expired or mis-versioned entry before verifying', t => {
  const { dir, write } = tree(t)
  write('audit.json', picomatchChain())
  write('package-lock.json', lockWith({ 'node_modules/picomatch': { version: '4.0.7' } }))
  for (const [entry, pattern] of [
    [{ ...ENTRY, reason: undefined }, /"reason"/],
    [{ ...ENTRY, ignoreUntil: '2026-01-01' }, /expired/],
    [{ ...ENTRY, version: '4.0.6' }, /accepted only for 4.0.6/],
  ]) {
    write('audit-accepted.json', { accepted: [entry] })
    const r = runMain(dir, [join(dir, 'audit.json')])
    assert.equal(r.status, 1)
    assert.match(r.logs, pattern)
    assert.deepEqual(r.calls, [])
  }
  write('audit-accepted.json', '{ not json')
  assert.match(runMain(dir, [join(dir, 'audit.json')]).logs, /not valid JSON/)
})

test('main reports each failing advisory and honours --audit-level', t => {
  const { dir, write } = tree(t)
  write('audit.json', { vulnerabilities: {
    foo: {
      severity: 'moderate', range: '<2.0.0', fixAvailable: { name: 'foo', version: '2.0.0', isSemVerMajor: true },
      via: [{ ...advisory('GHSA-aaaa-bbbb-cccc', 'moderate'), name: 'foo', title: 'Prototype pollution' }],
    },
  } })
  const r = runMain(dir, [join(dir, 'audit.json')])
  assert.equal(r.status, 1)
  assert.match(r.logs, /foo \(moderate, <2\.0\.0\)/)
  assert.match(r.logs, /GHSA-AAAA-BBBB-CCCC in foo: Prototype pollution/)
  assert.match(r.logs, /fix: foo@2\.0\.0 \(semver-major\)/)
  assert.match(r.logs, /at or above the moderate level: foo/)

  const high = runMain(dir, [join(dir, 'audit.json'), '--audit-level=high'])
  assert.equal(high.status, 0)
  assert.match(high.logs, /threshold high/)
  assert.equal(runMain(dir, [join(dir, 'audit.json'), '--audit-level=severe']).status, 1)
})
