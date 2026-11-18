import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { createRequire } from 'node:module'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { patchParser } from './patch-picomatch.mjs'

const require = createRequire(import.meta.url)
const pm = require('picomatch')
const parserPath = require.resolve('picomatch/lib/parse.js')
const patch = JSON.parse(readFileSync(new URL('../patches/picomatch-4.0.7.json', import.meta.url)))

test('installed parser is patched; application is idempotent and fails closed', () => {
  const installed = readFileSync(parserPath, 'utf8')
  let original = installed
  for (const { before, after } of [...patch.edits].reverse()) {
    assert.equal(original.split(after).length, 2)
    original = original.replace(after, () => before)
  }
  assert.equal(patchParser(original), installed)
  assert.equal(patchParser(installed), installed)
  assert.throws(() => patchParser(original + '\n'), /Unexpected picomatch parser/)
})

test('ordinary repeated wildcard matches remain compatible', () => {
  for (const options of [{}, { dot: true }, { windows: true }, { bash: true }, { nocase: true }]) {
    for (const [pattern, input] of [
      ['*(a*|b)', 'ax'], ['+(a*|b)', 'bbab'], ['+(ab*|c)', 'ccabcc'],
      ['*(a*|b)', 'bbb'], ['*/*(W*, *)*', 'my/folder (Work, Accts)'],
      ['+(W*, *)', 'W, W, '], ['+(*a)', 'aaa'], ['+(a*b)', 'abab'],
      ['+([*])', '***'], ['+("*")', '***'], ['+(\\*)', '***']
    ]) assert.equal(pm(pattern, options)(input), true, pattern)
    assert.equal(pm('*(a*|b)', options)('cx'), false)
  }
  assert.equal(pm('*(A*|b)', { flags: 'i' })('ax'), true)
  assert.equal(pm('+(a*| b)')('ab'), false)
  assert.equal(pm('+(a*| b)')('+(a*| b)'), true)
  assert.equal(pm('+(*a)', { capture: true })('+(*a)'), true)
  assert.equal(pm('+(*a)', { capture: true })('aaa'), false)
  // The upstream opt-out remains intentional; never execute it on long input.
  assert.match(pm.makeRe('+(*a)', { maxExtglobRecursion: false }).source, /\)[+*]/)
})

test('adversarial matcher, makeRe and parse calls finish within subprocess deadlines', () => {
  const cases = [
    ['+(*)', 'a'.repeat(60000) + '/'], ['*(*a)', 'a'.repeat(60000) + '!'],
    ['+(*a|b)', 'a'.repeat(60000) + '!'], ['+(a*a*a*a*a*)', 'a'.repeat(60000) + '/'],
    ['+([*a)', '[a'.repeat(10000) + '!'], ['+([*a])', '[a]'.repeat(10000) + '!'],
    ['+(W*, *)z', 'W, '.repeat(10000) + '!'], ['+(A*|a)', 'a'.repeat(60000) + '/']
  ]
  for (const [pattern, input] of cases) {
    for (const options of [{}, { windows: true }, { bash: true }, { capture: true }, { nobracket: true }, { flags: 'i' }]) {
      const script = `const assert=require('node:assert/strict'); const pm=require(process.argv[1]);
        const [pattern,input,options]=JSON.parse(process.argv[2]);
        assert.equal(typeof pm(pattern,options)(input),'boolean');
        assert.equal(typeof pm.makeRe(pattern,options).test(input),'boolean');
        assert.equal(typeof pm.compileRe(pm.parse(pattern,options),options).test(input),'boolean');`
      const result = spawnSync(process.execPath, ['--max-old-space-size=128', '-e', script,
        require.resolve('picomatch'), JSON.stringify([pattern, input, options])], {
        encoding: 'utf8', timeout: 2000, killSignal: 'SIGKILL', maxBuffer: 8192
      })
      assert.ifError(result.error)
      assert.equal(result.status, 0, `${pattern} ${JSON.stringify(options)}: ${result.stderr}`)
    }
  }
})
