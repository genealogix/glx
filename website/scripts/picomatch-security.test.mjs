import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { createRequire } from 'node:module'
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { test } from 'node:test'
import { installPatch, patchParser } from './patch-picomatch.mjs'

const require = createRequire(import.meta.url)
const pm = require('picomatch')
const parserPath = require.resolve('picomatch/lib/parse.js')
const installedVersion = require('picomatch/package.json').version
const patch = JSON.parse(readFileSync(new URL('../patches/picomatch-4.0.7.json', import.meta.url)))

// Patched, every case finishes in well under a second; the unpatched 4.0.7
// parser runs each of them for minutes. The deadline sits far between the two
// so a slow or loaded runner does not fail the audit gate, while a regression
// still does. PICOMATCH_TEST_TIMEOUT_MS overrides it.
const DEADLINE_MS = Number(process.env.PICOMATCH_TEST_TIMEOUT_MS) || 15000

const sha = file => require('node:crypto').createHash('sha256').update(readFileSync(file, 'utf8')).digest('hex')

// Reverses the backport's edits, giving the original 4.0.7 parser.
const unpatch = installed => {
  let original = installed
  for (const { before, after } of [...patch.edits].reverse()) {
    assert.equal(original.split(after).length, 2)
    original = original.replace(after, () => before)
  }
  return original
}

test('installed parser is patched; application is idempotent and fails closed', {
  skip: installedVersion !== patch.version &&
    `picomatch ${installedVersion} is installed; the backport targets ${patch.version} only`,
}, () => {
  const installed = readFileSync(parserPath, 'utf8')
  const original = unpatch(installed)
  assert.equal(patchParser(original), installed)
  assert.equal(patchParser(installed), installed)
  assert.throws(() => patchParser(original + '\n'), /Unexpected picomatch parser/)
})

test('installPatch patches only copies at the targeted version and tolerates missing ones', {
  skip: installedVersion !== patch.version && 'needs the original parser, derived from the installed patched copy',
}, t => {
  const original = unpatch(readFileSync(parserPath, 'utf8'))
  const dir = mkdtempSync(join(tmpdir(), 'patch-picomatch-'))
  t.after(() => rmSync(dir, { recursive: true, force: true }))
  const copy = (path, version, source) => {
    mkdirSync(join(dir, path, 'lib'), { recursive: true })
    writeFileSync(join(dir, path, 'package.json'), JSON.stringify({ version }))
    writeFileSync(join(dir, path, 'lib', 'parse.js'), source)
  }
  copy('node_modules/picomatch', patch.version, original)
  copy('node_modules/vite/node_modules/picomatch', '4.0.8', 'fixed upstream')
  writeFileSync(join(dir, 'package-lock.json'), JSON.stringify({ packages: {
    '': {},
    'node_modules/picomatch': { version: patch.version },
    'node_modules/vite/node_modules/picomatch': { version: '4.0.8' },
    'node_modules/tinyglobby/node_modules/picomatch': { version: patch.version, dev: true }, // omitted
  } }))
  const logs = []
  const root = pathToFileURL(dir + '/')
  assert.deepEqual(installPatch(root, l => logs.push(l)), ['node_modules/picomatch'])
  assert.equal(sha(join(dir, 'node_modules/picomatch/lib/parse.js')), patch.afterSha256)
  assert.equal(readFileSync(join(dir, 'node_modules/vite/node_modules/picomatch/lib/parse.js'), 'utf8'), 'fixed upstream')
  assert.match(logs.join('\n'), /vite\/node_modules\/picomatch is 4\.0\.8/)
  assert.deepEqual(installPatch(root, () => {}), ['node_modules/picomatch']) // idempotent

  writeFileSync(join(dir, 'node_modules/picomatch/lib/parse.js'), original + '\n')
  assert.throws(() => installPatch(root, () => {}), /Unexpected picomatch parser/)

  rmSync(join(dir, 'node_modules'), { recursive: true })
  assert.deepEqual(installPatch(root, () => {}), [])
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
  // The case goes in on stdin: a 60 KB argument exceeds the Windows command
  // line limit (ENAMETOOLONG), which would fail `make check` there.
  const script = `const assert=require('node:assert/strict'); const pm=require(process.argv[1]);
    const [pattern,input,options]=JSON.parse(require('node:fs').readFileSync(0,'utf8'));
    assert.equal(typeof pm(pattern,options)(input),'boolean');
    assert.equal(typeof pm.makeRe(pattern,options).test(input),'boolean');
    assert.equal(typeof pm.compileRe(pm.parse(pattern,options),options).test(input),'boolean');`
  for (const [pattern, input] of cases) {
    for (const options of [{}, { windows: true }, { bash: true }, { capture: true }, { nobracket: true }, { flags: 'i' }]) {
      const result = spawnSync(process.execPath, ['--max-old-space-size=128', '-e', script,
        require.resolve('picomatch')], {
        input: JSON.stringify([pattern, input, options]),
        encoding: 'utf8', timeout: DEADLINE_MS, killSignal: 'SIGKILL', maxBuffer: 8192
      })
      const label = `${pattern} ${JSON.stringify(options)}`
      assert.ifError(result.error && result.error.code !== 'ETIMEDOUT' ? result.error : null)
      assert.notEqual(result.signal, 'SIGKILL', `${label}: still running after ${DEADLINE_MS} ms`)
      assert.equal(result.status, 0, `${label}: ${result.stderr}`)
    }
  }
})
