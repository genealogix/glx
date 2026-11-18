import { test } from 'node:test'
import assert from 'node:assert/strict'
import { evaluate, ghsaOf } from './npm-audit-gate.mjs'

const advisory = (id, severity = 'high') => ({ source: 1, url: `https://github.com/advisories/${id}`, severity })
const ACCEPTED = [{ advisory: 'GHSA-fc5q-39xc-vjpf', package: 'picomatch', version: '4.0.7' }]

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
  assert.deepEqual(evaluate(report, ACCEPTED).failing.sort(), ['vite', 'vitepress'])
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
